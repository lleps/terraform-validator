package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const tfComplianceBin = "terraform-compliance"
const tfBin = "terraform"
const featuresPath = "./features"

var db dynamoDBFeaturesTable

func main() {
	listenFlag := flag.String("listen", ":8080", "On which address to listen")
	dynamoTableFlag := flag.String("dynamodb-features-table", "terraform-validator.features", "The dynamoDB table to use")
	flag.Parse()

	log.Printf("Init DynamoDB table '%s'...", *dynamoTableFlag)
	db = newDynamoDBFeaturesTable(*dynamoTableFlag)
	if err := db.ensureTableExists(); err != nil {
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
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(*listenFlag, nil))
}

// Register in the router a request with proper error handling and logging.
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

// Converts a TF file state (like plan.out, or terraform.tfstate) to pretty json
// using "terraform show -json {file}".
// Doesn't supports concurrent access, as uses a hardcoded temporary file.
func convertTerraformBinToJson(fileBytes []byte) (string, error) {
	// write the bytes to a tmp file
	path := os.TempDir() + "/" + "convertTfToJson.bin.tmp"
	if err := ioutil.WriteFile(path, fileBytes, os.ModePerm); err != nil {
		return "", fmt.Errorf("can't create tmp file '%s': %v", path, err)
	}
	defer os.Remove(path)

	// invoke the tool on that file
	outputBytes, err := exec.Command(tfBin, "show", "-json", path).Output()
	if err != nil || string(outputBytes) == "" {
		return "", fmt.Errorf("can't exec the tool: %v", err)
	}

	// prettify the json
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, outputBytes, "", "\t"); err != nil {
		return "", fmt.Errorf("can't prettify the json: %v", err)
	}

	return string(prettyJSON.Bytes()), nil

}

// Takes a base64 string in the body with the plan file content,
// run terraform-compliance against the file, and returns the
// raw tool output as a response.
func validateReq(body string, _ map[string]string) (string, int, error) {
	planFileBytes, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return "", 0, err
	}

	path := os.TempDir() + "/" + "compliance_plan.out"
	err = ioutil.WriteFile(path, planFileBytes, os.ModePerm)
	if err != nil {
		return "", 0, err
	}
	defer os.Remove(path)

	outputBytes, err := exec.Command(tfComplianceBin, "-p", path, "-f", featuresPath).CombinedOutput()
	outputString := stripansi.Strip(string(outputBytes))

	for _, line := range strings.Split(outputString, "\n") {
		scenarioCount, passedCount, failedCount, skippedCount := 0, 0, 0, 0
		count, err := fmt.Sscanf(line,
			"%d scenarios (%d passed, %d failed, %d skipped)",
			&scenarioCount, &passedCount, &failedCount, &skippedCount)

		if err == nil && count == 4 {

		}

	}

	log.Printf(" === %s output ===", tfComplianceBin)
	log.Printf(outputString)
	log.Printf(" === end output ===")

	if err != nil {
		return outputString, 0, err
	}

	if err = os.Remove(planTmpFile); err != nil {
		return "", 0, err
	}

	return string(outputBytes), http.StatusOK, nil
}

// List all features in the database.
func featuresReq(_ string, _ map[string]string) (string, int, error) {
	features, err := db.loadAll()
	if err != nil {
		return "", 0, err
	}

	sb := strings.Builder{}
	for _, f := range features {
		sb.WriteString(f.FeatureName)
		sb.WriteRune('\n')
	}

	return sb.String(), http.StatusOK, nil
}

// Get the source code of the given feature and returns it.
func featureSourceReq(_ string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	features, err := db.loadAll()
	if err != nil {
		return "", 0, err
	}

	for _, f := range features {
		if f.FeatureName == featureName {
			return f.FeatureSource, http.StatusOK, nil
		}
	}

	return "Feature not found", http.StatusNotFound, nil
}

// Add a new feature, with the source code specified in the body.
func featureAddReq(body string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	if err := db.insertOrUpdate(ComplianceFeature{ featureName, body }); err != nil {
		return "", 0, err
	}

	if err := syncFeaturesFolderFromDB(); err != nil {
		return "", 0, err
	}

	return "", http.StatusOK, nil
}

// Remove the feature with the given name.
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

	if err := db.removeByName(featureName); err != nil {
		return "", 0, err
	}

	if err := syncFeaturesFolderFromDB(); err != nil {
		return "", 0, err
	}

	return "", http.StatusOK, nil
}


// This writes all the .feature files (required by compliance) in
// the features folder, based on the content from the DB.
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
	features, err := db.loadAll()
	if err != nil {
		return err
	}
	for _, f := range features {
		filePath := featuresPath + "/" + f.FeatureName + ".feature"
		if err := ioutil.WriteFile(filePath, []byte(f.FeatureSource), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

// Returns true if the feature name is ok (doesn't contains invalid file characters)
func validateFeatureName(name string) bool {
	return !strings.ContainsAny(name, "./* ")
}

// Returns true if the given feature name exists in the DB, false otherwise.
func checkFeatureExists(name string) (bool, error) {
	features, err := db.loadAll()
	if err != nil {
		return false, err
	}

	for _, f := range features {
		if f.FeatureName == name {
			return true, nil
		}
	}
	return false, nil
}