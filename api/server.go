package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	listenFlag             = flag.String("listen", ":8080", "On which address to listen")
	dynamoPrefixFlag       = flag.String("dynamodb-prefix", "terraformvalidator", "The database table prefix to use")
	awsUseSharedConfig     = flag.Bool("aws-use-sharedconfig", false, "Use shared config files in AWS session")
	awsRegionFlag          = flag.String("aws-region", "", "AWS region to use for the session")
	awsAccessKeyIdFlag     = flag.String("aws-access-key-id", "", "credentials aws_access_key_id parameter")
	awsSecretAccessKeyFlag = flag.String("aws-secret-access-key", "", "credentials aws_secret_access_key")
	slackUrlFlagFlag       = flag.String("slack-url", "", "url to report failed validations")
	panelUrlFlag           = flag.String("panel-url", "", "panel url, for references.")
	oktaClientIdFlag       = flag.String("okta-client-id", "", "okta client id for authentication")
	oktaIssuerUrlFlag      = flag.String("okta-issuer-url", "", "okta issuer url")
	timestampFormat        = time.Stamp
)

func main() {
	flag.Parse()

	// Create session
	log.Printf("Create AWS session...")
	sess := createSession()
	log.Printf("Init DynamoDB tables at prefix '%s_*'...", *dynamoPrefixFlag)
	db := initDB(sess, *dynamoPrefixFlag)

	// parse okta credentials
	oktaClientId := *oktaClientIdFlag
	oktaIssuerUrl := *oktaIssuerUrlFlag
	if oktaClientId == "" || oktaIssuerUrl == "" {
		log.Fatalf("eiher -okta-client-id or -okta-issuer-url flags not given.")
	}
	InitOktaLoginCredentials(oktaClientId, oktaIssuerUrl)

	// Spawn monitoring routines
	log.Printf("Init state monitoring ticker...")
	initStateChangeMonitoring(sess, db, time.Second*60)
	if *slackUrlFlagFlag != "" {
		enableSlackPosts(*panelUrlFlag, *slackUrlFlagFlag)
		log.Println("Errors will be reported to slack. Panel url given: " + *panelUrlFlag)
	}

	// Init REST handlers
	log.Printf("Listening on '%s'...", *listenFlag)
	router := mux.NewRouter()
	registerPublicEndpoint(router, db, "/login-details", LoginDetailsHandler, "GET")
	registerAuthenticatedEndpoint(router, db, "/validate", validateHandler, "POST")
	initFeaturesEndpoint(router, db)
	initLogsEndpoint(router, db)
	initTFStatesEndpoint(router, db)
	http.Handle("/", router)

	// Start REST server (and CORS stuff)
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

func createSession() *session.Session {
	if *awsUseSharedConfig {
		log.Println("Init aws session... (using shared config)")
		return session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
	} else {
		log.Println("Init aws session... (using given flags)")
		flags := map[string]string{
			"region":            *awsRegionFlag,
			"access-key-id":     *awsAccessKeyIdFlag,
			"secret-access-key": *awsSecretAccessKeyFlag,
		}
		for f, v := range flags {
			if v == "" {
				log.Fatalf("Error: Some flag not given for aws authentication: -aws-" + f)
			}
		}
		config := aws.Config{
			Region: aws.String(flags["region"]),
			Credentials: credentials.NewStaticCredentialsFromCreds(
				credentials.Value{
					AccessKeyID:     flags["access-key-id"],
					SecretAccessKey: flags["secret-access-key"],
				}),
		}
		return session.Must(session.NewSessionWithOptions(session.Options{Config: config}))
	}
}

func initDB(sess *session.Session, prefix string) *database {
	result := newDynamoDB(sess, prefix)
	if err := result.initTables(complianceFeatureTable, validationLogTable, tfStateTable, foreignResourcesTable); err != nil {
		log.Fatalf("Can't make database table: %v", err)
	}
	return result
}

func initFeaturesEndpoint(router *mux.Router, db *database) {
	// '/features' supports all methods.
	registerAuthenticatedObjEndpoints(router, "/features", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllFeaturesFull()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		loadOneFunc:   func(db *database, id string) (restObject, error) { return db.findFeatureById(id) },
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
			err := db.saveFeature(feature)
			if err != nil {
				return nil, err
			}

			return feature, nil
		},
		putHandler: func(db *database, obj restObject, body string) error {
			type BodyFields struct {
				Source   string   `json:"source"`
				Tags     []string `json:"tags"`
				Disabled bool     `json:"disabled"`
			}
			var f BodyFields
			if err := json.Unmarshal([]byte(body), &f); err != nil {
				return fmt.Errorf("can't unmarshal into f: %v", err)
			}

			feature := obj.(*ComplianceFeature)
			feature.Source = f.Source
			feature.Tags = f.Tags
			feature.Disabled = f.Disabled
			return db.saveFeature(feature)
		},
	})
}

func initLogsEndpoint(router *mux.Router, db *database) {
	// '/logs' supports just GET and DELETE, since they're generated automatically.
	registerAuthenticatedObjEndpoints(router, "/logs", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllLogsMinimal()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		loadOneFunc:   func(db *database, id string) (restObject, error) { return db.findLogById(id) },
		deleteHandler: func(db *database, id string) error { return db.removeLog(id) },
	})
}

func initTFStatesEndpoint(router *mux.Router, db *database) {
	// Forcefully validation endpoint
	validationHandler := func(db *database, _ string, vars map[string]string) (string, int, error) {
		obj, err := db.findTFStateById(vars["id"])
		if err != nil {
			return "", 0, fmt.Errorf("can't find obj: %v", err)
		}
		if obj == nil {
			return "", http.StatusNotFound, nil
		}
		obj.ForceValidation = true
		if err := db.saveTFState(obj); err != nil {
			return "", 0, fmt.Errorf("can't save in db: %v", err)
		}
		return "", http.StatusOK, nil
	}
	registerAuthenticatedEndpoint(router, db, "/tfstates/{id}/validate", validationHandler, "POST")

	// '/tfstates' supports all methods.
	registerAuthenticatedObjEndpoints(router, "/tfstates", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllTFStatesMinimal()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		loadOneFunc:   func(db *database, id string) (restObject, error) { return db.findTFStateById(id) },
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
			if err := db.saveTFState(tfstate); err != nil {
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
			return db.saveTFState(tfstate)
		},
	})
}

func initForeignResourcesEndpoint(router *mux.Router, db *database) {
	// /foreignresources supports just GET.
	registerAuthenticatedObjEndpoints(router, "/foreignresources", db, restObjectHandler{
		loadAllFunc: func(db *database) ([]restObject, error) {
			objs, err := db.loadAllForeignResourcesMinimal()
			if err != nil {
				return nil, nil
			}
			result := make([]restObject, len(objs))
			for i, o := range objs {
				result[i] = o
			}
			return result, nil
		},
		loadOneFunc: func(db *database, id string) (restObject, error) { return db.findForeignResourceById(id) },
	})
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

	stateJSON, complianceOutput, err := runComplianceToolForTags(db, planFileBytes, []string{"validation"})
	if err != nil {
		return "", 0, fmt.Errorf("can't run compliance tool: %v", err)
	}

	complianceResult := parseComplianceOutput(complianceOutput)
	logEntry := newValidationLog(stateJSON, complianceResult)
	if err := db.saveLog(logEntry); err != nil {
		return "", 0, fmt.Errorf("can't insert logEntry: %v", err)
	}

	return complianceOutput, http.StatusOK, nil
}

// validateFeatureName returns true if the given feature name is valid (doesn't contains invalid file characters).
func validateFeatureName(name string) bool {
	return len(name) > 0 && len(name) < 30 && !strings.ContainsAny(name, "./* ")
}
