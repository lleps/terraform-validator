package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	featuresPath = "./features"
)

var db *database

func main() {
	listenFlag := flag.String("listen", ":8080", "On which address to listen")
	dynamoPrefixFlag := flag.String("dynamodb-prefix", "terraformvalidator", "The database table prefix to use")
	flag.Parse()

	log.Printf("Init DynamoDB table '%s'...", *dynamoPrefixFlag)
	db = initDB(*dynamoPrefixFlag)

	log.Println("Sync feature files from DB...")
	if err := syncFeaturesFolderFromDB(); err != nil {
		log.Fatalf("Can't sync features from db (features path: '%s'): %v", featuresPath, err)
	}

	log.Printf("Init state monitoring...")
	initMonitoring()

	log.Printf("Listening on '%s'...", *listenFlag)
	initEndpoints()
	log.Fatal(http.ListenAndServe(*listenFlag, nil))
}

func initDB(prefix string) *database {
	result := newDynamoDB(prefix)
	if err := result.initTables(complianceFeatureTable, validationLogTable, tfStateTable); err != nil {
		log.Fatalf("Can't make database table: %v", err)
	}
	return result
}

func initMonitoring() {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			objs, err := db.loadAllTFStates()
			if err != nil {
				log.Printf("can't get tfstates to check: %v", err)
				continue
			}

			for _, obj := range objs {
				changed, logEntry, err := checkTFState(obj)
				if err != nil {
					log.Printf("can't check TFState for state #%s (%s:%s): %v", obj.Id, obj.Bucket, obj.Path, err)
					continue
				}

				if changed {
					log.Printf("Bucket %s:%s changed state. Registered in log #%s", obj.Bucket, obj.Path, logEntry.Id)
				}
			}
		}
	}()
}

func initEndpoints() {
	r := mux.NewRouter()
	registerEndpoint(r, "/validate", validateHandler, "POST")
	registerCollectionEndpoint(db, collectionEndpointBuilder{
		router: r,
		endpoint: "/features",
		dbFetchFunc: func(db *database) ([]restObject, error) {
			objs, err :=  db.loadAllFeatures()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		dbRemoveFunc: func(db *database, id string) error {
			defer func() { _ = syncFeaturesFolderFromDB() }()
			return db.removeFeature(id)
		},
		dbInsertFunc: func(db *database, body string) error {
			var data map[string]string
			if err := json.Unmarshal([]byte(body), &data); err != nil {
				return fmt.Errorf("can't unmarshal into map[string]string: %v", err)
			}

			name := data["name"]
			source := data["source"]

			if name == "" || source == "" {
				return fmt.Errorf("'name' or 'source' not given")
			}
			if !validateFeatureName(name) {
				return fmt.Errorf("invalid feature name: '%s'", name)
			}

			defer func() { _ = syncFeaturesFolderFromDB() }()
			return db.insertOrUpdateFeature(&ComplianceFeature{name, source})
		},
	})
	registerCollectionEndpoint(db, collectionEndpointBuilder{
		router: r,
		endpoint: "/logs",
		dbFetchFunc: func(db *database) ([]restObject, error) {
			objs, err :=  db.loadAllValidationLogs()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		dbRemoveFunc: func(db *database, id string) error { return db.removeValidationLog(id) },
		dbInsertFunc: nil, // POST not supported
	})
	registerCollectionEndpoint(db, collectionEndpointBuilder{
		router: r,
		endpoint: "/tfstates",
		dbFetchFunc: func(db *database) ([]restObject, error) {
			objs, err :=  db.loadAllTFStates()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		dbRemoveFunc: func(db *database, id string) error { return db.removeTFState(id) },
		dbInsertFunc: func(db *database, body string) error {
			var data map[string]string
			if err := json.Unmarshal([]byte(body), &data); err != nil {
				return fmt.Errorf("can't unmarshal into map[string]string: %v", err)
			}

			bucket := data["bucket"]
			path := data["path"]

			// TODO: instead of adding this raw, should
			//  add this filled. Like, get the object
			//  from S3, run compliance on it, and log
			//  results.
			if bucket == "" || path == "" {
				return fmt.Errorf("'bucket' or 'path' not given")
			}

			id, err := db.nextFreeTFStateId()
			if err != nil {
				return fmt.Errorf("can't get id: %v", err)
			}

			return db.insertOrUpdateTFState(&TFState{
				Id: id,
				Bucket: bucket,
				Path: path,
			})
		},
	})

	http.Handle("/", r)
}

// checkTFState checks if the given tfstate changed in S3.
// if it did, runs the compliance and adds a new log entry.
func checkTFState(state *TFState) (changed bool, logEntry *ValidationLog, err error) {
	fileBytes, err := getFileFromS3(state.Bucket, state.Path)
	if err != nil {
		return false, nil, fmt.Errorf("can't get tfstate from s3: %v", err)
	}

	actualState, err := convertTerraformBinToJson(fileBytes)
	if err != nil {
		return false, nil, fmt.Errorf("can't convert to json: %v", err)
	}

	// TODO: if features changed, should run too.
	if actualState == state.State {
		return false, nil, nil
	}

	changed = true
	_, output, err := runComplianceTool([]byte(actualState))
	if err != nil {
		return true, nil, fmt.Errorf("can't run compliance tool: %v", err)
	}

	freeId, err := db.nextFreeValidationLogId()
	if err != nil {
		return true, nil, fmt.Errorf("can't get an id for a validationLog: %v", err)
	}

	// Register the log entry
	now := time.Now().Format(time.Stamp)
	logEntry = &ValidationLog{
		Id:            freeId,
		Kind:          logKindTFState,
		DateTime:      now,
		InputJson:     actualState,
		Output:        output,
		PrevInputJson: state.State,
		PrevOutput:    state.ComplianceResult,
	}
	if err := db.insertOrUpdateValidationLog(logEntry); err != nil {
		return true, nil, fmt.Errorf("can't insert logEntry on DB: %v", err)
	}

	// update the state
	state.LastUpdate = now
	state.State = actualState
	state.ComplianceResult = output
	if err := db.insertOrUpdateTFState(state); err != nil {
		return true, nil, fmt.Errorf("can't update tfstate on DB: %v", err)
	}
	return
}

// validateHandler takes a base64 string in the body with the plan file content
// or terraform json, run the tfComplianceBin tool against it, and responds
// the tool output as a response.
func validateHandler(body string, _ map[string]string) (string, int, error) {
	planFileBytes, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return "", 0, err
	}

	complianceInput, complianceOutput, err := runComplianceTool(planFileBytes)
	if err != nil {
		return "", 0, fmt.Errorf("can't run compliance tool: %v", err)
	}

	id, err := db.nextFreeValidationLogId()
	if err != nil {
		return "", 0, fmt.Errorf("can't get an id: %v", err)
	}

	logEntry := ValidationLog{
		Id:        id,
		Kind:      logKindValidation,
		DateTime:  time.Now().Format(time.Stamp),
		InputJson: complianceInput,
		Output:    complianceOutput,
	}

	if err := db.insertOrUpdateValidationLog(&logEntry); err != nil {
		return "", 0, fmt.Errorf("can't insert logEntry: %v", err)
	}

	return complianceOutput, http.StatusOK, nil
}

// syncFeaturesFolderFromDB writes all the feature files that terraform-compliance requires.
func syncFeaturesFolderFromDB() error {
	// Empty the folder
	if err := os.RemoveAll(featuresPath); err != nil {
		if os.IsNotExist(err) {
			// ok. Not created yet
		} else {
			// somewhat with permissions maybe
			return err
		}
	}
	if err := os.MkdirAll(featuresPath, os.ModePerm); err != nil {
		return err
	}

	// Write all feature files
	features, err := db.loadAllFeatures()
	if err != nil {
		return err
	}
	for _, f := range features {
		filePath := featuresPath + "/" + f.Id + ".feature"
		if err := ioutil.WriteFile(filePath, []byte(f.FeatureSource), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

// validateFeatureName returns true if the given feature name is valid (doesn't contains invalid file characters).
func validateFeatureName(name string) bool {
	return !strings.ContainsAny(name, "./* ")
}