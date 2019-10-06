// This file contains tfstate monitoring logic
package main

import (
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"time"
)

var lastPullTime = time.Time{}

// initStateChangeMonitoring starts a goroutine that periodically checks if
// tfstates changed, and if they did runs the compliance tool and logs results.
// TODO: some way to retrieve only selected fields from DB to reduce bandwidth
//  usage with dynamo.
func initStateChangeMonitoring(sess *session.Session, db *database, frequency time.Duration) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			objs, err := db.loadAllTFStates()
			if err != nil {
				log.Printf("can't get tfstates to check: %v", err)
				continue
			}

			timeToPull := time.Since(lastPullTime) >= frequency
			if timeToPull {
				lastPullTime = time.Now()
			}

			for _, obj := range objs {
				if timeToPull || obj.ForceValidation {
					changed, logEntry, err := checkTFState(sess, db, obj)
					if err != nil {
						log.Printf("can't check TFState %s (%s:%s): %v", obj.Id, obj.Bucket, obj.Path, err)
						continue
					}

					if changed && logEntry != nil {
						log.Printf("Bucket %s:%s changed state. Registered in log %s", obj.Bucket, obj.Path, logEntry.Id)
					}
				}
			}
		}
	}()
}

// checkTFState checks the given tfstate for compliance.
func checkTFState(
	sess *session.Session,
	db *database,
	tfstate *TFState,
) (changed bool, logEntry *ValidationLog, err error) {
	changed, lastModification, stateJSON, complianceOutput, err := performTFStateCheckIfNecessary(sess, db, tfstate)
	log.Println("Check for log #", tfstate.Id)
	log.Println("Changed:", changed, "LastModification:", lastModification, "\nstateJSON:", stateJSON, "output", complianceOutput)
	if err != nil {
		// Errors here should be reported to the user too. Because they're likely produced
		// by bad feature input or compliance tool misconfiguration.
		tfstate.ComplianceResult = "Error: " + stripansi.Strip(err.Error())
		tfstate.ForceValidation = false
		if err2 := db.insertOrUpdateTFState(tfstate); err2 != nil {
			return true, nil, fmt.Errorf("can't update tfstate on DB: %v", err)
		}
		return
	}

	// State check went good. Register log entry
	now := time.Now().Format(timestampFormat)
	logEntry = newTFStateLog(stateJSON, complianceOutput, tfstate.State, tfstate.ComplianceResult, tfstate.Account, tfstate.Bucket, tfstate.Path)
	if err := db.insertLog(logEntry); err != nil {
		return true, nil, fmt.Errorf("can't insert logEntry on DB: %v", err)
	}

	// Update the state with timestamps and result. Unmark the force check flag as well.
	tfstate.ForceValidation = false
	tfstate.LastUpdate = now
	tfstate.State = stateJSON
	tfstate.ComplianceResult = complianceOutput
	tfstate.S3LastModification = lastModification
	if err := db.insertOrUpdateTFState(tfstate); err != nil {
		return true, nil, fmt.Errorf("can't update tfstate on DB: %v", err)
	}
	return
}

// performStateCheck checks if the given state needs to be checked (either
// because the state has the ForceValidation flag or the bucket data changed).
// If it does, pulls from S3 and returns the compliance result for it.
func performTFStateCheckIfNecessary(
	sess *session.Session,
	db *database,
	state *TFState,
) (changed bool, lastModification string, stateJSON string, output string, err error) {

	bucket := state.Bucket
	path := state.Path

	// When ForceValidation set lastModification to "", so this always fetches from s3.
	prevLastModification := state.S3LastModification
	if state.ForceValidation {
		prevLastModification = ""
	}

	var itemBytes []byte
	changed, itemBytes, lastModification, err = getItemFromS3IfChanged(sess, bucket, path, prevLastModification)
	if err != nil {
		err = fmt.Errorf("can't get tfstate from s3: %v", err)
		return
	}

	if !changed {
		return
	}

	stateJSON, err = convertTerraformBinToJSON(itemBytes)
	fmt.Println("stateJSON after cnvert: ", stateJSON)
	if err != nil {
		err = fmt.Errorf("can't convert to json: %v", err)
		return
	}

	_, output, err = runComplianceToolForTags(db, []byte(stateJSON), state.Tags)
	if err != nil {
		err = fmt.Errorf("can't run compliance tool: %v", err)
		return
	}

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
