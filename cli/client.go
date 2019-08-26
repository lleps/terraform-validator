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
	listFeaturesFlag := flag.Bool("list", false, "List all features")
	listLogsFlag := flag.Bool("logs", false, "List all logs")
	logGetFlag := flag.String("log", "", "Get the info of the given log")
	addFeatureFlag := flag.String("add", "", "Add a new feature from the given file. The name will be the file name.")
	featureSourceFlag := flag.String("read", "", "Get the source code of the given feature.")
	removeFeatureFlag := flag.String("remove", "", "Remove the feature with the given name")
	replaceFlag := flag.Bool("replace", false, "For -add, to replace the feature if it already exists.")

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
		resContent, resCode, resErr = execRequest(host, "/features/source/"+*featureSourceFlag, "GET", "")
	} else if *addFeatureFlag != "" { // --add-feature
		content, err := ioutil.ReadFile(*addFeatureFlag)
		if err != nil {
			fmt.Println("Can't read file:", err)
			return
		}

		if !strings.HasSuffix(*addFeatureFlag, ".feature") {
			fmt.Println("File must end in .feature.")
			return
		}

		featureName := strings.TrimSuffix(*addFeatureFlag, ".feature")
		exists, err := checkIfFeatureExists(host, featureName)
		resErr = err
		if resErr == nil {
			if exists && !*replaceFlag {
				fmt.Printf("Feature '%s' already exists. Pass --replace to overwrite it.\n", featureName)
				return
			}
			resContent, resCode, resErr = execRequest(host, "/features/add/"+featureName, "POST", string(content))
		}
	} else if *removeFeatureFlag != "" { // --remove-feature
		fileWithoutExt := strings.TrimSuffix(*removeFeatureFlag, ".feature")
		resContent, resCode, resErr = execRequest(host, "/features/remove/"+fileWithoutExt, "DELETE", "")
	} else if *listLogsFlag { // --logs
		resContent, resCode, resErr = execRequest(host, "/logs", "GET", "")
	} else if *logGetFlag != "" { // --log
		resContent, resCode, resErr = execRequest(host, "/logs/" + *logGetFlag, "GET", "")
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
		fmt.Println("HTTP code:", resCode)
		fmt.Println(resContent)
		return
	}

	fmt.Print(resContent)
}

func checkIfFeatureExists(host, name string) (bool, error) {
	content, code, err := execRequest(host, "/features", "GET", "")
	if err != nil {
		return false, nil
	}

	if code == 200 {
		for _, s := range strings.Split(content, "\n") {
			if s == name {
				return true, nil
			}
		}
	} else {
		return false, fmt.Errorf("code not 200: %d", code)
	}

	return false, nil
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
