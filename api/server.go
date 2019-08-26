package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/mux"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	tfComplianceBin = "terraform-compliance"
	tfBin           = "terraform"
	featuresPath    = "./features"
)

var db dynamoDB

func main() {
	listenFlag := flag.String("listen", ":8080", "On which address to listen")
	dynamoTableFlag := flag.String("dynamodb-features-table", "terraformvalidator", "The dynamoDB table prefix to use")
	flag.Parse()

	log.Printf("Init DynamoDB table '%s'...", *dynamoTableFlag)
	db = newDynamoDB(*dynamoTableFlag)
	if err := db.initTables(); err != nil {
		log.Fatalf("Can't make dynamoDB table: %v", err)
	}

	if err := syncFeaturesFolderFromDB(); err != nil {
		log.Fatalf("Can't sync features from db (features path: '%s'): %v", featuresPath, err)
	}

	log.Printf("Will listen on '%s'...", *listenFlag)

	r := mux.NewRouter()
	registerRequest(r, "/validate", validateReq, "POST")
	registerRequest(r, "/features", featuresReq, "GET")
	registerRequest(r, "/features/source/{name}", featureSourceReq, "GET")
	registerRequest(r, "/features/add/{name}", featureAddReq, "POST")
	registerRequest(r, "/features/remove/{name}", featureRemoveReq, "DELETE")
	registerRequest(r, "/logs", logsReq, "GET")
	registerRequest(r, "/logs/{id}", logsGetReq, "GET")

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			fetchStateFromS3AndValidateIt()
		}
	}()

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(*listenFlag, nil))
}

func fetchStateFromS3AndValidateIt() {
	fileBytes, err := getFileFromS3("mybucket-gagagagagagag-2020", "path/to/my/key")
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

// convertTerraformBinToJson converts a TF file state (like plan.out) to a
// pretty json string by invoking internally "terraform show -json".
// Doesn't supports concurrent access, as uses a hardcoded temporary file.
func convertTerraformBinToJson(fileBytes []byte) (string, error) {
	// write the bytes to a tmp file
	path := os.TempDir() + "/" + "convertTfToJson.bin.tmp"
	if err := ioutil.WriteFile(path, fileBytes, os.ModePerm); err != nil {
		return "", fmt.Errorf("can't create tmp file '%s': %v", path, err)
	}
	defer os.Remove(path)

	// invoke the tool on that file
	outputBytes, err := exec.Command(tfBin, "show", "-json", path).CombinedOutput()
	if err != nil || string(outputBytes) == "" {
		return "", fmt.Errorf("can't exec the tool: %v. out: %s", err, string(outputBytes))
	}

	// prettify the json
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, outputBytes, "", "\t"); err != nil {
		return "", fmt.Errorf("can't prettify the json: %v", err)
	}

	return string(prettyJSON.Bytes()), nil
}

// parseComplianceToolOutput parses compliance tool output into a ValidationLog struct.
func parseComplianceToolOutput(output string, record *ValidationLog) {
	record.WasSuccessful = false

	for _, line := range strings.Split(output, "\n") {
		featureCount, passedCount, failedCount, skippedCount := 0, 0, 0, 0

		// "X features (X passed, X failed, X skipped)"
		count, err := fmt.Sscanf(line,
			"%d features (%d passed, %d failed, %d skipped)",
			&featureCount, &passedCount, &failedCount, &skippedCount)

		if err != nil { // above failed, maybe "X features (X passed, X skipped)"?
			count, err = fmt.Sscanf(line,
				"%d features (%d passed, %d skipped)",
				&featureCount, &passedCount, &skippedCount)
			failedCount = 0
		}

		// if any of them match, parse into record and break the loop
		if err == nil && count >= 3 {
			record.WasSuccessful = true
			record.FailedCount = failedCount
			record.PassedCount = passedCount
			record.SkippedCount = skippedCount
			break
		}
	}
}

// runComplianceTool runs the tfComplianceBin against the given file content.
// fileContent may be either a json string, or a terraform binary file format.
// Returns the input and output of the tool if successful.
func runComplianceTool(fileContent []byte) (string, string, error) {
	if len(fileContent) == 0 {
		return "", "", fmt.Errorf("empty file content")
	}

	var complianceToolInput []byte

	// in case the content is not already a json (doesn't starts with "{"), may be in
	// tf bin format (like plan.out or terraform.tfstate). Try to convert it to json.
	if fileContent[0] != '{' {
		asJson, err := convertTerraformBinToJson(fileContent)
		if err != nil {
			return "", "", fmt.Errorf("cntent given can't be converted to json: %v", err)
		}
		complianceToolInput = []byte(asJson)
	} else {
		complianceToolInput = fileContent
	}

	// write the json content to a tmp file
	jsonTmpPath := os.TempDir() + "/" + "compliance_input.json"
	if err := ioutil.WriteFile(jsonTmpPath, complianceToolInput, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("can't create tmp file: %v", err)
	}
	defer os.Remove(jsonTmpPath)

	// run the compliance tool against the created file
	toolOutputBytes, err := exec.Command(tfComplianceBin, "-p", jsonTmpPath, "-f", featuresPath).CombinedOutput()
	toolOutput := stripansi.Strip(string(toolOutputBytes))
	if err != nil {
		log.Printf("Tool output: \n%s\n", toolOutput)
		return "", "", fmt.Errorf("tool execution error: %v: %s", err, toolOutput)
	}

	return string(complianceToolInput), toolOutput, nil
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

func getFileFromS3(bucket, item string) ([]byte, error) {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	downloader := s3manager.NewDownloader(sess)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})

	if err != nil {
		return nil, fmt.Errorf("can't download item: '%s': %v", item, err)
	}

	return buf.Bytes(), nil
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
