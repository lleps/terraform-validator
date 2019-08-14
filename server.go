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

const documentation = `
This tool is used to manipulate the terraform-compliance tool
and its state through a REST API to test terraform plan files,
and edit the requirements as well.

    USAGE:

 {program-name} --help       Show this
 {program-name} address      Start listening at address (example '0.0.0.0:80')
 {program-name}              Start Listening at the default address (:8080)

	API:

POST /validate
Validate a terraform plan file with the current features. The plan file
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
	r.HandleFunc("/validate", ValidateReq).Methods("POST")
	r.HandleFunc("/features", FeaturesReq).Methods("GET")
	r.HandleFunc("/features/source/{name}", FeaturesSourceReq).Methods("GET")
	r.HandleFunc("/features/add/{name}", FeaturesAddReq).Methods("PUT")
	r.HandleFunc("/features/remove/{name}", FeaturesRemoveReq).Methods("DELETE")
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func ValidateReq(w http.ResponseWriter, r *http.Request) {
	output, err := exec.
		Command("terraform-compliance", "-p", "plan.out", "-f", "../terraform-compliance/example/example_01/aws/").
		CombinedOutput()

	if err != nil {
		os.Stderr.WriteString(err.Error())
	} else {
		fmt.Fprintf(w, string(output))
	}
}

func FeaturesReq(w http.ResponseWriter, r *http.Request) {
	path := "../terraform-compliance/example/example_01/aws/"
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Fprintln(w, f.Name())
	}
}

func FeaturesSourceReq(w http.ResponseWriter, r *http.Request) {
}

func FeaturesAddReq(w http.ResponseWriter, r *http.Request) {
}

func FeaturesRemoveReq(w http.ResponseWriter, r *http.Request) {
}