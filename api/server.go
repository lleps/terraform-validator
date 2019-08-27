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

var db dynamoDB

func main() {
	// parse flags
	listenFlag := flag.String("listen", ":8080", "On which address to listen")
	dynamoPrefixFlag := flag.String("dynamodb-prefix", "terraformvalidator", "The dynamoDB table prefix to use")
	flag.Parse()

	// init dynamoDB tables, sync features
	log.Printf("Init DynamoDB table '%s'...", *dynamoPrefixFlag)
	db = newDynamoDB(*dynamoPrefixFlag)
	if err := db.initTables(); err != nil {
		log.Fatalf("Can't make dynamoDB table: %v", err)
	}
	if err := syncFeaturesFolderFromDB(); err != nil {
		log.Fatalf("Can't sync features from db (features path: '%s'): %v", featuresPath, err)
	}

	// init monitoring tasks
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			validateCurrentTerraformState()
		}
	}()

	// register requests
	r := mux.NewRouter()
	registerRequest(r, "/validate", validateReq, "POST")
	registerRequest(r, "/features", featuresReq, "GET")
	registerRequest(r, "/features/source/{name}", featureSourceReq, "GET")
	registerRequest(r, "/features/add/{name}", featureAddReq, "POST")
	registerRequest(r, "/features/remove/{name}", featureRemoveReq, "DELETE")
	registerRequest(r, "/logs", logsReq, "GET")
	registerRequest(r, "/logs/{id}", logsGetReq, "GET")
	log.Printf("Will listen on '%s'...", *listenFlag)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(*listenFlag, nil))
}

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

// registerRequest registers in the router an HTTP request with proper error handling and logging.
func registerRequest(
	router *mux.Router,
	endpoint string,
	handler func(string, map[string]string) (string, int, error),
	method string,
) {
	router.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		log.Println()
		log.Printf("%s %s [from %s]", r.Method, r.URL, r.RemoteAddr)

		// parse body and vars
		vars := mux.Vars(r)
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("Can't read body:", err)
			return
		}

		// execute the handler
		response, code, err := handler(string(bodyBytes), vars)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, response)
			log.Println("Handler error:", err)
			return
		}

		// write response
		w.WriteHeader(code)
		_, err = fmt.Fprint(w, response)
		if err != nil {
			log.Println("Can't write response:", err)
		}

		// log request and response code
		log.Printf("HTTP Response: %d", code)
	}).Methods(method)
}

// validateReq takes a base64 string in the body with the plan file content
// or terraform json, run the tfComplianceBin tool against it, and responds
// the tool output as a response.
func validateReq(body string, _ map[string]string) (string, int, error) {
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

	if err := db.insertOrUpdateValidationLog(record); err != nil {
		return "", 0, fmt.Errorf("can't put record in db: %v", err)
	}

	return complianceOutput, http.StatusOK, nil
}

// featuresReq responds the list of features actually on the database.
func featuresReq(_ string, _ map[string]string) (string, int, error) {
	features, err := db.loadAllFeatures()
	if err != nil {
		return "", 0, err
	}

	sb := strings.Builder{}
	for _, f := range features {
		sb.WriteString(f.Id)
		sb.WriteRune('\n')
	}

	return sb.String(), http.StatusOK, nil
}

// featureSourceReq responds the source code of the given feature.
func featureSourceReq(_ string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	features, err := db.loadAllFeatures()
	if err != nil {
		return "", 0, err
	}

	for _, f := range features {
		if f.Id == featureName {
			return f.FeatureSource, http.StatusOK, nil
		}
	}

	return "Feature not found", http.StatusNotFound, nil
}

// featureAddReq adds a new feature in the database, with the source code specified in the body.
func featureAddReq(body string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	if err := db.insertOrUpdateFeature(ComplianceFeature{featureName, body}); err != nil {
		return "", 0, err
	}

	if err := syncFeaturesFolderFromDB(); err != nil {
		return "", 0, err
	}

	return "", http.StatusOK, nil
}

// featureRemoveReq removes a feature from the database.
func featureRemoveReq(_ string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	exists, err := checkFeatureExists(featureName)
	if err != nil {
		return "", 0, err
	}

	if !exists {
		return "Feature not found", http.StatusNotFound, nil
	}

	if err := db.removeFeature(featureName); err != nil {
		return "", 0, err
	}

	if err := syncFeaturesFolderFromDB(); err != nil {
		return "", 0, err
	}

	return "", http.StatusOK, nil
}

// logsReq responds the list of log entries in the database.
func logsReq(_ string, _ map[string]string) (string, int, error) {
	logs, err := db.loadAllValidationLogs()
	if err != nil {
		return "", 0, err
	}

	// reverse output
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	// build response
	sb := strings.Builder{}
	for _, l := range logs {
		if l.WasSuccessful {
			state := "successful"
			if l.FailedCount > 0 {
				state = "failed"
			}
			sb.WriteString(fmt.Sprintf("#%s - %s - %s (%d passed, %d failed, %d skipped)",
				l.Id, l.DateTime, state, l.PassedCount, l.FailedCount, l.SkippedCount))
		} else {
			sb.WriteString(fmt.Sprintf("#%s - %s - can't execute)",
				l.Id, l.DateTime))
		}
		sb.WriteRune('\n')
	}

	return sb.String(), http.StatusOK, nil
}

// logsGetReq responds the content of the log with the given id.
func logsGetReq(_ string, vars map[string]string) (string, int, error) {
	logId := vars["id"]
	logs, err := db.loadAllValidationLogs()
	if err != nil {
		return "", 0, err
	}

	var logEntry *ValidationLog
	for _, l := range logs {
		if l.Id == logId {
			logEntry = &l
			break
		}
	}

	if logEntry == nil {
		return "Log entry not found", http.StatusNotFound, nil
	}

	sb := strings.Builder{}
	sb.WriteString("\n======\nInput:======\n")
	sb.WriteString(logEntry.InputJson)
	sb.WriteString("\n======\nOutput:======\n")
	sb.WriteString(logEntry.Output)
	sb.WriteRune('\n')
	return sb.String(), http.StatusOK, nil
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

// checkFeatureExists returns true if the given feature name exists in the database.
func checkFeatureExists(name string) (bool, error) {
	features, err := db.loadAllFeatures()
	if err != nil {
		return false, err
	}

	for _, f := range features {
		if f.Id == name {
			return true, nil
		}
	}
	return false, nil
}
