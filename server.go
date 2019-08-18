package main

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const tfComplianceBin = "terraform-compliance"
const featuresPath = "../terraform-compliance/example/example_01/aws/" // should use the same directory
const planTmpFile = "tmpplan.out"                                // the plan.out is created here to test, and deleted after that.

func main() {
	args := os.Args
	addr := ":8080"
	if len(args) == 2 && (args[1] == "-h" || args[1] == "--help")  {
		fmt.Printf("Usage: %s [listen address (default %s)]", args[0], addr)
		return
	}

	if len(args) == 2 {
		addr = args[1]
		log.Printf("Listen at %s...\n", addr)
	} else {
		log.Printf("If you want to use an specific address, pass it as a param.")
		log.Printf("Listen at default %s...\n", addr)
	}

	r := mux.NewRouter()
	registerRequest(r, "/validate", validateReq, "POST")
	registerRequest(r, "/features", featuresReq, "GET")
	registerRequest(r, "/features/source/{name}", featureSourceReq, "GET")
	registerRequest(r, "/features/add/{name}", featureAddReq, "POST")
	registerRequest(r, "/features/remove/{name}", featureRemoveReq, "DELETE")
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(addr, nil))
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

// Takes a base64 string in the body with the plan file content,
// run terraform-compliance against the file, and returns the
// raw tool output as a response.
func validateReq(body string, _ map[string]string) (string, int, error) {
	planFileBytes, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return "", 0, err
	}

	err = ioutil.WriteFile(planTmpFile, planFileBytes, os.ModePerm)
	if err != nil {
		return "", 0, err
	}

	outputBytes, err := exec.Command(tfComplianceBin, "-p", planTmpFile, "-f", featuresPath).CombinedOutput()
	outputString := string(outputBytes)
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

// List all files ending with ".feature" in featuresPath.
func featuresReq(_ string, _ map[string]string) (string, int, error) {
	files, err := ioutil.ReadDir(featuresPath)
	if err != nil {
		return "", 0, err
	}

	sb := strings.Builder{}
	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, ".feature") {
			sb.WriteString(strings.TrimSuffix(name, ".feature"))
			sb.WriteRune('\n')
		}
	}

	return sb.String(), http.StatusOK, nil
}

// Returns true if the feature name is ok (doesn't contains invalid file characters)
func validateFeatureName(name string) bool {
	return !strings.ContainsAny(name, "./* ")
}

// Read the source code of the given feature and returns it.
func featureSourceReq(_ string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	fullPath := featuresPath + "/" + featureName + ".feature"
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return "Feature not found", http.StatusNotFound, nil
	}

	return string(content), http.StatusOK, nil
}

// Add a new feature with the source code in the body
func featureAddReq(body string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	// just write the body to the file.
	// Will overwrite if the feature already exists
	fullPath := featuresPath + "/" + featureName + ".feature"
	err := ioutil.WriteFile(fullPath, []byte(body), os.ModePerm)
	if err != nil {
		return "", 0, err
	}

	return "", http.StatusOK, nil
}

// Remove the feature file with the given name
func featureRemoveReq(_ string, vars map[string]string) (string, int, error) {
	featureName := vars["name"]
	if !validateFeatureName(featureName) {
		return "Illegal feature name.", http.StatusBadRequest, nil
	}

	fullPath := featuresPath + "/" + featureName + ".feature"
	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "Feature not found", 404, nil
		} else {
			return "", 0, err
		}
	}

	return "", http.StatusOK, nil
}