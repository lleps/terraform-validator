package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	listenFlag       = flag.String("listen", ":8080", "On which address to listen")
	dynamoPrefixFlag = flag.String("dynamodb-prefix", "terraformvalidator", "The database table prefix to use")
	timestampFormat  = time.Stamp
)

func main() {
	flag.Parse()

	log.Println("Init aws session... (using shared config)")
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	log.Printf("Init DynamoDB tables at prefix '%s_*'...", *dynamoPrefixFlag)
	db := initDB(sess, *dynamoPrefixFlag)

	log.Printf("Init state and resource monitoring tickers...")
	initStateChangeMonitoring(sess, db)
	initAccountResourcesMonitoring(sess, db)

	log.Printf("Listening on '%s'...", *listenFlag)
	router := initEndpoints(db)

	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})

	corsHandler := handlers.CORS(headersOk, originsOk, methodsOk)(router)
	loggingHandler := handlers.LoggingHandler(LogWriter{}, corsHandler)
	log.Fatal(http.ListenAndServe(*listenFlag, loggingHandler))
}

// LogWriter used to log requests as regular calls to log.Printf
type LogWriter struct{}

func (_ LogWriter) Write(bytes []byte) (n int, err error) {
	log.Print(string(bytes))
	return
}

func initDB(sess *session.Session, prefix string) *database {
	result := newDynamoDB(sess, prefix)
	if err := result.initTables(complianceFeatureTable, validationLogTable, tfStateTable, foreignResourcesTable); err != nil {
		log.Fatalf("Can't make database table: %v", err)
	}
	return result
}

// initAccountResourcesMonitoring starts a goroutine that periodically checks if there are
// resources in the account that don't belong to any registered tfstate, and reports them.
func initAccountResourcesMonitoring(sess *session.Session, db *database) {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			// Load all tfstates and current foreign resources.
			// Quick. tfstates contains maybe a lot of data,
			// but foreign resources contains only a few fields.
			tfStates, err1 := db.loadAllTFStates()
			foreignResources, err2 := db.loadAllForeignResources()
			if err1 != nil {
				log.Printf("Can't load tfstates to monitor for resources outside terraform states: %v", err1)
				continue
			}
			if err2 != nil {
				log.Printf("Can't load foreignresources to monitor for resources outside terraform states: %v", err2)
				continue
			}

			// This is the slow part.
			// Should do some kind of parallelism.
			resources, err := ListAllResources(sess)
			if err != nil {
				log.Printf("Can't list aws resources: %v", err)
				continue
			}

			// Ensure all resources are in at least one tfstate.
			findForeignResourceEntry := func(resourceId string) *ForeignResource {
				for _, fr := range foreignResources {
					if fr.ResourceId == resourceId {
						return fr
					}
				}
				return nil
			}
			findResourceInBuckets := func(id string) *TFState {
				for _, tfstate := range tfStates {
					if strings.Contains(tfstate.State, id) {
						return tfstate
					}
				}
				return nil
			}

			// This is the fast part. Just memory accesses.

			for _, r := range resources {
				// For new discovered resources, should check if findResourceInBuckets. If it is,
				// insert to db and log.
				existingFr := findForeignResourceEntry(r.ID())
				resourceBucket := findResourceInBuckets(r.ID())
				if existingFr == nil {
					if resourceBucket == nil {
						fr := newForeignResource("ec2-instance", r.ID(), "type: ec2-micro\nami: abcde-123456")
						if err := db.insertForeignResource(fr); err != nil {
							log.Printf("Can't insert fr: %v", err)
							continue
						}
						log.Printf("New foreign resource registered: '%s' #%s", r.ID(), fr.Id)
					}
				} else {
					// The resource is not new. Gotta check if the resource is still foreign.
					// if it isn't, log and delete from DB.
					if resourceBucket != nil {
						// not foreign anymore. Delete this.
						if err := db.removeForeignResource(existingFr.id()); err != nil {
							log.Printf("Can't delete fr: %v", err)
							continue
						}
						log.Printf("Foreign resource #%s (%s) not foreign anymore! Found in bucket %s:%s. Deleted!",
							existingFr.id(), existingFr.ResourceId,
							resourceBucket.Bucket, resourceBucket.Path)
					}
				}
			}
		}
	}()
}

// initStateChangeMonitoring starts a goroutine that periodically checks if
// tfstates changed, and if they did runs the compliance tool and logs results.
func initStateChangeMonitoring(sess *session.Session, db *database) {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			objs, err := db.loadAllTFStates()
			if err != nil {
				log.Printf("can't get tfstates to check: %v", err)
				continue
			}

			for _, obj := range objs {
				changed, logEntry, err := checkTFState(sess, db, obj)
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

func initEndpoints(db *database) *mux.Router {
	r := mux.NewRouter()
	registerEndpoint(r, db, "/validate", validateHandler, "POST")

	// '/features' supports all methods.
	registerObjEndpoints(r, "/features", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllFeatures()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		deleteHandler: func(db *database, id string) error { return db.removeFeature(id) },
		postHandler: func(db *database, body string) (restObject, error) {
			type BodyFields struct {
				Name   string   `json:"name"`
				Source string   `json:"source"`
				Tags   []string `json:"tags"`
			}
			var f BodyFields
			if err := json.Unmarshal([]byte(body), &f); err != nil {
				return nil, fmt.Errorf("can't unmarshal into f: %v", err)
			}

			if f.Tags == nil || f.Name == "" || f.Source == "" {
				return nil, fmt.Errorf("'name', 'tags' or 'source' not given")
			}

			if !validateFeatureName(f.Name) {
				return nil, fmt.Errorf("invalid feature name: '%s'", f.Name)
			}

			feature := newFeature(f.Name, f.Source, f.Tags)
			err := db.insertOrUpdateFeature(feature)
			if err != nil {
				return nil, err
			}

			return feature, nil
		},
		putHandler: func(db *database, obj restObject, body string) error {
			type BodyFields struct {
				Source string   `json:"source"`
				Tags   []string `json:"tags"`
			}
			var f BodyFields
			if err := json.Unmarshal([]byte(body), &f); err != nil {
				return fmt.Errorf("can't unmarshal into f: %v", err)
			}

			feature := obj.(*ComplianceFeature)
			feature.Source = f.Source
			feature.Tags = f.Tags
			return db.insertOrUpdateFeature(feature)
		},
	})

	// '/logs' supports just GET and DELETE, since they're generated automatically.
	registerObjEndpoints(r, "/logs", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllLogs()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		deleteHandler: func(db *database, id string) error { return db.removeLog(id) },
	})

	// '/tfstates' supports all methods.
	registerObjEndpoints(r, "/tfstates", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllTFStates()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		deleteHandler: func(db *database, id string) error { return db.removeTFState(id) },
		postHandler: func(db *database, body string) (restObject, error) {
			type BodyFields struct {
				Account string   `json:"account"`
				Bucket  string   `json:"bucket"`
				Path    string   `json:"path"`
				Tags    []string `json:"tags"`
			}
			var f BodyFields
			if err := json.Unmarshal([]byte(body), &f); err != nil {
				return nil, fmt.Errorf("can't unmarshal into f: %v", err)
			}
			if f.Account == "" || f.Bucket == "" || f.Path == "" {
				return nil, fmt.Errorf("'account', 'bucket' or 'path' not given")
			}

			tfstate := newTFState(f.Account, f.Bucket, f.Path, f.Tags)
			if err := db.insertOrUpdateTFState(tfstate); err != nil {
				return nil, err
			}

			return tfstate, nil
		},
		putHandler: func(db *database, obj restObject, body string) error {
			type BodyFields struct {
				Account string   `json:"account"`
				Bucket  string   `json:"bucket"`
				Path    string   `json:"path"`
				Tags    []string `json:"tags"`
			}
			var f BodyFields
			if err := json.Unmarshal([]byte(body), &f); err != nil {
				return fmt.Errorf("can't unmarshal into f: %v", err)
			}
			if f.Account == "" || f.Bucket == "" || f.Path == "" {
				return fmt.Errorf("'account', 'bucket' or 'path' not given")
			}

			tfstate := obj.(*TFState)
			tfstate.Account = f.Account
			tfstate.Bucket = f.Bucket
			tfstate.Path = f.Path
			tfstate.Tags = f.Tags
			return db.insertOrUpdateTFState(tfstate)
		},
	})

	// /foreignresources supports just GET.
	registerObjEndpoints(r, "/foreignresources", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllForeignResources()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
	})

	http.Handle("/", r)
	return r
}

// checkTFState checks if the given tfstate changed in S3.
// if it did, runs the compliance tool and adds a new log entry.
func checkTFState(sess *session.Session, db *database, state *TFState) (changed bool, logEntry *ValidationLog, err error) {
	// Fetch bucket item if changed.
	bucket := state.Bucket
	path := state.Path
	changed, itemBytes, lastModification, err := getItemFromS3IfChanged(sess, bucket, path, state.S3LastModification)
	if err != nil {
		return false, nil, fmt.Errorf("can't get tfstate from s3: %v", err)
	}

	if !changed {
		return false, nil, nil
	}

	// Assume changed (item last modification changed).
	// Convert to json to run compliance.
	changed = true
	actualState, err := convertTerraformBinToJson(itemBytes)
	if err != nil {
		return false, nil, fmt.Errorf("can't convert to json: %v", err)
	}

	// Run compliance
	_, output, err := runComplianceToolForTags(db, []byte(actualState), state.Tags)
	if err != nil {
		return true, nil, fmt.Errorf("can't run compliance tool: %v", err)
	}

	// Register the log entry
	now := time.Now().Format(timestampFormat)
	logEntry = newTFStateLog(actualState, output, state.State, state.ComplianceResult, state.Account, state.Bucket, state.Path)
	if err := db.insertLog(logEntry); err != nil {
		return true, nil, fmt.Errorf("can't insert logEntry on DB: %v", err)
	}

	// update the state
	state.LastUpdate = now
	state.State = actualState
	state.ComplianceResult = output
	state.S3LastModification = lastModification
	if err := db.insertOrUpdateTFState(state); err != nil {
		return true, nil, fmt.Errorf("can't update tfstate on DB: %v", err)
	}
	return
}

// validateHandler takes a base64 string in the body with the plan file content
// or terraform json, run the tfComplianceBin tool against it, and responds
// the tool output as a response.
func validateHandler(db *database, body string, _ map[string]string) (string, int, error) {
	var base64data string
	if err := json.Unmarshal([]byte(body), &base64data); err != nil {
		return "", 0, fmt.Errorf("can't decode into json string: %v", err)
	}

	planFileBytes, err := base64.StdEncoding.DecodeString(base64data)
	if err != nil {
		return "", 0, err
	}

	complianceInput, complianceOutput, err := runComplianceToolForTags(db, planFileBytes, []string{"validation"})
	if err != nil {
		return "", 0, fmt.Errorf("can't run compliance tool: %v", err)
	}

	logEntry := newValidationLog(complianceInput, complianceOutput)
	if err := db.insertLog(logEntry); err != nil {
		return "", 0, fmt.Errorf("can't insert logEntry: %v", err)
	}

	return complianceOutput, http.StatusOK, nil
}

// validateFeatureName returns true if the given feature name is valid (doesn't contains invalid file characters).
func validateFeatureName(name string) bool {
	return !strings.ContainsAny(name, "./* ")
}
