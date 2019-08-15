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
const helpMsg = `
	USAGE:

 {program-name} --help       Show this
 {program-name} address      Start listening at address (example '0.0.0.0:80')
 {program-name}              Start listening at the default address (:8080)
`

func main() {
	args := os.Args
	if len(args) == 2 && args[1] == "--help" {
		fmt.Print(strings.Replace(helpMsg, "{program-name}", args[0], -1))
		return
	}

	addr := ":8080"
	if len(args) == 2 {
		addr = args[1]
		log.Printf("Listen at %s...\n", addr)
	} else {
		log.Printf("If you want to use an specific address, pass it as the only flag.")
		log.Printf("Listen at default %s...\n", addr)
	}

	r := mux.NewRouter()
	r.HandleFunc("/validate", ValidateReq).Methods("POST")
	r.HandleFunc("/features", FeaturesReq).Methods("GET")
	r.HandleFunc("/features/source/{name}", FeaturesSourceReq).Methods("GET")
	r.HandleFunc("/features/add/{name}", FeaturesAddReq).Methods("POST")
	r.HandleFunc("/features/remove/{name}", FeaturesRemoveReq).Methods("DELETE")
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// Returns true if err is not nil, also logs the err and responds to the client
func checkError(endpoint string, err error, w http.ResponseWriter) bool {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error at", endpoint, ":", err)
		return true
	}
	return false
}

// Takes a base64 string in the body with the plan file content,
// run terraform-compliance against the file, and returns the
// raw tool output as a response.
func ValidateReq(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Parse body (a base64 string)
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if checkError("/validate", err, w) {
		return
	}

	planFileBytes, err := base64.StdEncoding.DecodeString(string(bodyBytes))
	if checkError("/validate", err, w) {
		return
	}

	// Write the file content on the given file
	err = ioutil.WriteFile(planTmpFile, planFileBytes, os.ModePerm)
	if checkError("/validate", err, w) {
		return
	}

	// Run terraform-compliance against the created file
	outputBytes, _ := exec.Command(tfComplianceBin, "-p", planTmpFile, "-f", featuresPath).CombinedOutput()

	// Return the validation result
	_, err = fmt.Fprintf(w, string(outputBytes))
	if checkError("/validate", err, w) {
		return
	}

	// Delete the tmp file
	checkError("/validate", os.Remove(planTmpFile), w)
}

func FeaturesReq(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	files, err := ioutil.ReadDir(featuresPath)
	if checkError("/features", err, w) {
		return
	}

	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, ".feature") {
			_, err = fmt.Fprintln(w, strings.TrimSuffix(name, ".feature"))
			if checkError("/features", err, w) {
				return
			}
		}
	}
}

func FeaturesSourceReq(w http.ResponseWriter, r *http.Request) {
}

func FeaturesAddReq(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadAll(r.Body)
	if checkError("/features/add", err, w) {
		return
	}

	bodyString := string(content)
	log.Printf("body:", bodyString)
	fmt.Fprintf(w, bodyString)
}

func FeaturesRemoveReq(w http.ResponseWriter, r *http.Request) {
}
