package main

import (
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
const planTmpFile = "./plan.out"                                       // the plan.out is created here to test, and deleted after that.
const documentation = `
This tool is used to manipulate the terraform-compliance tool
and its state through a REST API to test terraform plan files,
and edit the requirements as well.

	USAGE:

 {program-name} --help       Show this
 {program-name} address      Start listening at address (example '0.0.0.0:80')
 {program-name}              Start Listening at the default address (:8080)

	API:

POST /test
Test a terraform plan file with the current features. The plan file
is passed as a raw base64 string in the body.

GET /features
List all the used features.

GET /features/source/:name
Get the source code of the feature named :name.

PUT /features/add/:name
Add a new feature with :name. The feature source code is also passed
raw in the request body.
The syntax used to describe features is specified at:
https://github.com/eerkunt/terraform-compliance/blob/master/README.md
New calls to /validate will use the new feature to test the plan file.

DELETE /features/delete/:name
Delete the feature with :name (if any). New calls to /validate won't use
the deleted feature to test the plan file.
`

func main() {
	args := os.Args
	if len(args) == 2 && args[1] == "--help" {
		fmt.Print(strings.Replace(documentation, "{program-name}", args[0], -1))
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
	r.HandleFunc("/validate", ValidateReq).Methods("GET")
	r.HandleFunc("/features", FeaturesReq).Methods("GET")
	r.HandleFunc("/features/source/{name}", FeaturesSourceReq).Methods("GET")
	r.HandleFunc("/features/add/{name}", FeaturesAddReq).Methods("PUT")
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

func ValidateReq(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	output, err := exec.
		Command(tfComplianceBin, "-p", planTmpFile, "-f", featuresPath).
		CombinedOutput()

	if checkError("/validate", err, w) {
		return
	}

	_, err = fmt.Fprintf(w, string(output))
	checkError("/validate", err, w)
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
}

func FeaturesRemoveReq(w http.ResponseWriter, r *http.Request) {
}