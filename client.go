package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func main() {
	// cmd flags
	hostFlag := flag.String("host", "http://localhost:8080", "The host to connect to")
	validateFlag := flag.String("validate", "", "Validate the given terraform plan file.")
	listFeaturesFlag := flag.Bool("list-features", false, "List all features")
	addFeatureFlag := flag.String("add-feature", "", "Add a new feature from the given file. The name will be the file name.")
	featureSourceFlag := flag.String("feature-source", "", "Get the source code of the given feature.")
	removeFeatureFlag := flag.String("remove-feature", "", "Remove the feature with the given name")
	flag.Parse()
	host := *hostFlag

	// send request depending on the flags
	var resContent string
	var resCode int
	var resErr error
	if *validateFlag != "" { // --validate
		content, err := ioutil.ReadFile(*validateFlag)
		if err != nil {
			log.Fatal("Can't read file:", err)
			return
		}

		asB64 := base64.StdEncoding.EncodeToString(content)
		resContent, resCode, resErr = execRequest(host, "/validate", "POST", asB64)
	} else if *listFeaturesFlag { // --list-features
		resContent, resCode, resErr = execRequest(host, "/features", "GET", "")
	} else if *featureSourceFlag != "" { // --list-features
		resContent, resCode, resErr = execRequest(host, "/feature/source/" + *featureSourceFlag, "GET", "")
	} else if *addFeatureFlag != "" { // --add-feature
		content, err := ioutil.ReadFile(*addFeatureFlag)
		if err != nil {
			log.Fatal("Can't read file:", err)
			return
		}

		fileWithoutExt := strings.TrimSuffix(*addFeatureFlag, ".feature")
		resContent, resCode, resErr = execRequest(host, "/feature/add/" + fileWithoutExt, "POST", string(content))
	} else if *removeFeatureFlag != "" { // --remove-feature
		fileWithoutExt := strings.TrimSuffix(*removeFeatureFlag, ".feature")
		resContent, resCode, resErr = execRequest(host, "/feature/remove/" + fileWithoutExt, "REMOVE", "")
	} else {
		fmt.Println("No option given. Check -h to see options.")
		return
	}

	// show request result (or an error)
	if resErr != nil {
		fmt.Println("Error during request:", resErr)
		return
	}

	if resCode != http.StatusOK {
		fmt.Println("Invalid HTTP response code:", resCode)
		fmt.Println(resContent)
		return
	}

	fmt.Print(resContent)
}

func execRequest(
	host string,
	endpoint string,
	reqType string,
	content string,
) (resContent string, resCode int, err error) {
	url := host + endpoint
	var resp *http.Response

	switch reqType {
	case "POST":
		resp, err = http.Post(url, "text/plain", strings.NewReader(content))
	case "GET":
		resp, err = http.Get(url)
	case "DELETE":
		client := &http.Client{}
		req, err := http.NewRequest("DELETE", url, strings.NewReader(content))
		if err == nil {
			resp, err = client.Do(req)
		}
	default:
		panic(fmt.Sprintln("invalid reqType", reqType))
	}

	if err != nil {
		return
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	resContent = string(bodyBytes)
	resCode = resp.StatusCode
	return
}