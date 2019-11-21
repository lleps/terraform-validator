// This file contains tfstate monitoring logic
package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

var mLastFullPull = time.Time{}
var mPanelUrl, mSlackHookUrl = "", ""

// enableSlackPosts will enable slack posts to report failed
// state validations to a slack channel.
func enableSlackPosts(panelUrl, slackHookUrl string) {
	mPanelUrl = panelUrl
	mSlackHookUrl = slackHookUrl
}

// initStateChangeMonitoring starts a goroutine that periodically checks if
// tfstates changed, and if they did runs the compliance tool and logs results.
func initStateChangeMonitoring(sess *session.Session, db *database, frequency time.Duration) {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			// Check if its time to do a full pull, or
			// just pull forced validation objs.
			var objs []*TFState
			var err error
			if time.Since(mLastFullPull) >= frequency {
				mLastFullPull = time.Now()
				objs, err = db.loadAllTFStatesFull()
			} else {
				objs, err = db.loadTFStatesWithForceValidation()
			}
			if err != nil {
				log.Printf("can't pull tfstates: %v", err)
				continue
			}

			for _, obj := range objs {
				changed, logEntry, err := checkTFState(sess, db, obj)
				if err != nil {
					log.Printf("can't check TFState %s (%s:%s): %v", obj.Id, obj.Bucket, obj.Path, err)
					continue
				}

				if mSlackHookUrl != "" && logEntry != nil && logEntry.ComplianceResult.FailCount > 0 {
					err = reportFailedValidationToSlack(mSlackHookUrl, mPanelUrl, obj, logEntry)
					if err != nil {
						log.Printf("can't send to slack: %v", err)
					}
				}

				if changed && logEntry != nil {
					log.Printf("Bucket %s:%s changed state. Registered in log %s", obj.Bucket, obj.Path, logEntry.Id)
				}
			}
		}
	}()
}

func reportFailedValidationToSlack(slackUrl string, panelUrl string, state *TFState, logEntry *ValidationLog) error {
	var postBody struct {
		Text string `json:"text"`
	}

	postBody.Text = fmt.Sprintf(
		"Automatic state validation failed for state at %s:%s. See details at %s.",
		state.Bucket, state.Path, panelUrl+"/logs/"+logEntry.Id)

	marshaled, err := json.Marshal(postBody)
	if err != nil {
		return fmt.Errorf("can't marshal into json: %v", err)
	}

	fullBody := "payload=" + string(marshaled)
	req, err := http.NewRequest("POST", slackUrl, strings.NewReader(fullBody))
	if err != nil {
		return fmt.Errorf("can't build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("can't do request: %v", err)
	}

	defer resp.Body.Close()
	respContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("can't read body bytes: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("invalid response code %d: %s", resp.StatusCode, string(respContent))
	}

	return nil
}

// checkTFState checks the given tfstate for compliance.
func checkTFState(
	sess *session.Session,
	db *database,
	tfstate *TFState,
) (changed bool, logEntry *ValidationLog, err error) {
	checked, lastModification, stateJSON, complianceResult, err := checkTFStateIfNecessary(sess, db, tfstate)
	if err != nil {
		log.Printf("Can't check tfstate. Will update error status and move on: %v", err)
		tfstate.ForceValidation = false
		tfstate.ComplianceResult.Initialized = true
		tfstate.ComplianceResult.Error = true
		tfstate.ComplianceResult.ErrorMessage = "failed: " + err.Error()
		err = db.saveTFState(tfstate)
		return
	}

	if !checked { // if this wasn't checked its because the check isn't forced and the bucket didn't change
		return
	}

	changed = tfstate.State != stateJSON || !tfstate.ComplianceResult.equals(complianceResult)

	// Register log entry
	now := time.Now().Format(timestampFormat)
	logEntry = newTFStateLog(stateJSON, complianceResult, tfstate.State, tfstate.ComplianceResult, tfstate.Account, tfstate.Bucket, tfstate.Path)
	if err = db.saveLog(logEntry); err != nil {
		err = fmt.Errorf("can't insert logEntry on DB: %v", err)
		return
	}

	// Update the state with timestamps and result. Unmark the force check flag as well.
	tfstate.ForceValidation = false
	tfstate.LastUpdate = now
	tfstate.State = stateJSON
	tfstate.ComplianceResult = complianceResult
	tfstate.S3LastModification = lastModification
	if err = db.saveTFState(tfstate); err != nil {
		err = fmt.Errorf("can't update tfstate on DB: %v", err)
		return
	}

	return
}

// performStateCheck checks if the given state needs to be checked (either
// because the state has the ForceValidation flag or the bucket data changed).
// If it does, pulls from S3 and returns the compliance result for it.
func checkTFStateIfNecessary(
	sess *session.Session,
	db *database,
	state *TFState,
) (checked bool, lastModification string, stateJSON string, complianceResult ComplianceResult, err error) {

	bucket := state.Bucket
	path := state.Path

	// When ForceValidation set lastModification to "", so this always fetches from s3.
	prevLastModification := state.S3LastModification
	if state.ForceValidation {
		prevLastModification = ""
	}

	var itemBytes []byte
	changed, itemBytes, lastModification, err := getItemFromS3IfChanged(sess, bucket, path, prevLastModification)
	if err != nil {
		err = fmt.Errorf("can't get tfstate from s3: %v", err)
		return
	}

	if !changed {
		return
	}
	checked = true

	stateJSON, err = convertTerraformBinToJSON(itemBytes)
	if err != nil {
		err = fmt.Errorf("can't convert to json: %v", err)
		return
	}

	_, output, err := runComplianceToolForTags(db, []byte(stateJSON), state.Tags)
	if err != nil {
		err = fmt.Errorf("can't run compliance tool: %v", err)
		return
	}

	complianceResult = parseComplianceOutput(output)
	return
}

// getItemFromS3IfChanged fetches the content of the given bucket-item only
// if the item last modification date is different from the passed prevLastUpdate.
// Otherwise, sets the changed bool to false and returns an empty string.
func getItemFromS3IfChanged(
	sess *session.Session,
	bucket string,
	item string,
	prevLastModification string,
) (changed bool, content []byte, lastModification string, err error) {
	svc := s3.New(sess)

	// Get object head
	head, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(item),
	})
	if err != nil {
		err = fmt.Errorf("can't get object head data for %s:%s: %v", bucket, item, err)
		return
	}

	// Check if changed. Return now if didn't. Also set return value lastModification in any case.
	timeFormat := time.ANSIC
	lastModification = head.LastModified.Format(timeFormat)
	changed = lastModification != prevLastModification
	if !changed {
		return
	}

	// Download
	downloader := s3manager.NewDownloader(sess)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})

	if err != nil {
		err = fmt.Errorf("can't download item for %s:%s: %v", bucket, item, err)
		return
	}

	content = buf.Bytes()
	return
}
