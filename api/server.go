package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	featuresPath = "./features"
	s3Bucket = "mybucket-gagagagagagag-2020"
	s3Path = "path/to/my/key"
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
	if err := result.initTables(); err != nil {
		log.Fatalf("Can't make database table: %v", err)
	}
	return result
}

func initMonitoring() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			//validateCurrentTerraformState()
		}
	}()
}

func initEndpoints() {
	r := mux.NewRouter()
	registerEndpoint(r, "/validate", validateHandler, "POST")
	registerCollectionEndpoint(db, collectionEndpointBuilder{
		router: r,
		endpoint: "/features",
		dbFetcher: func(db *database) ([]restObject, error) {
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
		dbRemover: func(db *database, id string) error {
			defer func() { _ = syncFeaturesFolderFromDB() }()
			return db.removeFeature(id)
		},
		dbInserter: func(db *database, urlVars map[string]string, body string) error {
			featureName := urlVars["id"]
			if !validateFeatureName(featureName) {
				return fmt.Errorf("invalid feature name: '%s'", featureName)
			}

			defer func() { _ = syncFeaturesFolderFromDB() }()
			return db.insertOrUpdateFeature(&ComplianceFeature{featureName, body})
		},
	})
	registerCollectionEndpoint(db, collectionEndpointBuilder{
		router: r,
		endpoint: "/logs",
		dbFetcher: func(db *database) ([]restObject, error) {
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
		dbRemover: func(db *database, id string) error { return db.removeValidationLog(id) },
		dbInserter: nil, // POST not supported
	})
	http.Handle("/", r)
}

// validateCurrentTerraformState should fetch the state from the
// configured buckets, and log its changes (if any).
func validateCurrentTerraformState() {
	fileBytes, err := getFileFromS3(s3Bucket, s3Path)
	if err != nil {
		log.Printf("error getting tf state from s3: %v", err)
		return
	}

	_, output, err := runComplianceTool(fileBytes)
	if err != nil {
		log.Printf("can't run tool against the state: %v", err)
		return
	}
	fmt.Printf("result: %s", output)
}


// validateHandler takes a base64 string in the body with the plan file content
// or terraform json, run the tfComplianceBin tool against it, and responds
// the tool output as a response.
func validateHandler(body string, _ map[string]string) (string, int, error) {
	planFileBytes, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return "", 0, err
	}

	// run terraform-compliance
	complianceInput, complianceOutput, err := runComplianceTool(planFileBytes)
	if err != nil {
		return "", 0, fmt.Errorf("can't run compliance tool: %v", err)
	}

	// Calculate an ID for the validation
	maxId := 0
	records, err := db.loadAllValidationLogs()
	if err != nil {
		return "", 0, err
	}
	for _, record := range records {
		recordId, _ := strconv.ParseInt(record.Id, 10, 64)
		if int(recordId) > maxId {
			maxId = int(recordId)
		}
	}

	// log it
	record := ValidationLog{
		Id:        strconv.Itoa(maxId + 1),
		DateTime:  time.Now().Format(time.ANSIC),
		InputJson: complianceInput,
		Output:    complianceOutput,
	}
	parseComplianceToolOutput(complianceOutput, &record)
	if record.WasSuccessful {
		log.Printf("Validation result: %d scenarios passed, %d failed and %d skipped.",
			record.PassedCount,
			record.FailedCount,
			record.SkippedCount)
	} else {
		log.Printf("Validation failed. The tool wasn't executed successfully.")
		log.Printf("Tool output: \n%s", complianceOutput)
	}

	if err := db.insertOrUpdateValidationLog(&record); err != nil {
		return "", 0, fmt.Errorf("can't put record in db: %v", err)
	}

	return complianceOutput, http.StatusOK, nil
}

// syncFeaturesFolderFromDB writes all the features in the database onto featuresPath.
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