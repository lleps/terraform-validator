package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	// cmd flags
	hostFlag := flag.String("host", "http://localhost:8080", "The host to connect to")
	// features
	validateFlag := flag.String("validate", "", "Validate the given terraform plan file.")
	featureListFlag := flag.Bool("feature-list", false, "List all features")
	featureAddFlag := flag.String("feature-add", "", "Add a new feature from the given file. The name will be the file name.")
	featureRemoveFlag := flag.String("remove-remove", "", "Remove the feature with the given name")
	featureDetailsFlag := flag.String("feature-details", "", "Get the source code of the given feature.")
	featureReplaceFlag := flag.Bool("replace", false, "For -add, to replace the feature if it already exists.")
	// logs
	logListFlag := flag.Bool("log-list", false, "List all logs")
	logGetFlag := flag.String("log-details", "", "Get the info of the given log")
	logRemoveFlag := flag.String("log-remove", "", "Remove the log entry with the given id")
	// tfstates
	tfStateListFlag := flag.Bool("tfstate-list", false, "List all tfstates monitored")
	tfStateGetFlag := flag.String("tfstate-details", "", "Get the info of the given tfsatte")
	tfStateAddFlag := flag.Bool("tfstate-add", false, "Adds a new tfstate (along with -bucket and -path)")
	tfStateRemoveFlag := flag.String("tfstate-remove", "", "Remove the tfstate entry with the given id")
	tfStateBucket := flag.String("bucket", "", "When -tfstate-add. To specify the bucket to add.")
	tfStatePath := flag.String("path", "", "When -tfstate-add. To specify the path to add.")

	flag.Parse()

	host := *hostFlag

	var res string
	var code int
	var resErr error

	switch {

	// validation

	case *validateFlag != "":
		content, err := ioutil.ReadFile(*validateFlag)
		if err != nil {
			log.Fatal("Can't read file:", err)
			return
		}

		asB64 := base64.StdEncoding.EncodeToString(content)
		res, code, resErr = execRequest(host, "/validate", "POST", asB64)

	// -feature-*

	case *featureListFlag:
		res, code, resErr = execRequest(host, "/features", "GET", "")
	case *featureAddFlag != "":
		content, err := ioutil.ReadFile(*featureAddFlag)
		if err != nil {
			fmt.Println("Can't read file:", err)
			return
		}

		if !strings.HasSuffix(*featureAddFlag, ".feature") {
			fmt.Println("File must end in .feature.")
			return
		}

		featureFileName := extractNameFromPath(*featureAddFlag)
		featureName := strings.TrimSuffix(featureFileName, ".feature")
		exists, err := checkIfFeatureExists(host, featureName)
		resErr = err
		if resErr == nil {
			if exists && !*featureReplaceFlag {
				fmt.Printf("Feature '%s' already exists. Pass --replace to overwrite it.\n", featureName)
				return
			}
			body := map[string]string { "name": featureName, "source": string(content) }
			res, code, resErr = execRequest(host, "/features", "POST", body)
		}

	case *featureRemoveFlag != "":
		fileWithoutExt := strings.TrimSuffix(*featureRemoveFlag, ".feature")
		res, code, resErr = execRequest(host, "/features/" + url.QueryEscape(fileWithoutExt), "DELETE", "")

	case *featureDetailsFlag != "":
		res, code, resErr = execRequest(host, "/features/" + url.QueryEscape(*featureDetailsFlag), "GET", "")

	// -log-*

	case *logListFlag:
		res, code, resErr = execRequest(host, "/logs", "GET", "")
	case *logRemoveFlag != "":
		res, code, resErr = execRequest(host, "/logs/" + url.QueryEscape(*logRemoveFlag), "DELETE", "")
	case *logGetFlag != "":
		res, code, resErr = execRequest(host, "/logs/" + url.QueryEscape(*logGetFlag), "GET", "")

	// -tfstate-*
	case *tfStateListFlag:
		res, code, resErr = execRequest(host, "/tfstates", "GET", "")
	case *tfStateRemoveFlag != "":
		res, code, resErr = execRequest(host, "/tfstates/" + url.QueryEscape(*tfStateRemoveFlag), "DELETE", "")
	case *tfStateGetFlag != "":
		res, code, resErr = execRequest(host, "/tfstates/" + url.QueryEscape(*tfStateGetFlag), "GET", "")
	case *tfStateAddFlag:
		if *tfStateBucket == "" || *tfStatePath == "" {
			fmt.Printf("Please specify -bucket and -path when adding a tfstate.\n")
			return
		}
		body := map[string]string { "bucket": *tfStateBucket, "path": *tfStatePath }
		res, code, resErr = execRequest(host, "/tfstates", "POST", body)

	default:
		fmt.Println("No option given. Check -h to see options.")
		return
	}

	if resErr != nil {
		fmt.Println("Error during request:", resErr)
		return
	}

	if code != http.StatusOK {
		fmt.Println("Invalid HTTP Response:", code)
		if res != "" {
			fmt.Println(res)
		}
	} else {
		fmt.Print(res)
	}
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
	body interface{},
) (result string, code int, err error) {
	url := host + endpoint
	marshaled, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	bodyJson := string(marshaled)
	var resp *http.Response

	switch reqType {
	case "POST":
		resp, err = http.Post(url, "text/plain", strings.NewReader(bodyJson))
	case "GET":
		resp, err = http.Get(url)
	case "DELETE":
		client := &http.Client{}
		req, err := http.NewRequest("DELETE", url, strings.NewReader(bodyJson))
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

	result = string(bodyBytes)
	code = resp.StatusCode
	return
}

func reversed(s string) string {
	chars := []rune(s)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

// extractNameFromPath takes the file name from the whole path,
// for example "path/to/my/file" returns "file", and "myfile" returns "myfile".
func extractNameFromPath(path string) string {
	if len(path) == 0 {
		return ""
	}

	chars := []rune(path)
	sb := strings.Builder{}
	for i := len(chars) - 1; i >= 0; i-- {
		if chars[i] == os.PathSeparator {
			break
		}

		sb.WriteRune(chars[i])
	}
	return reversed(sb.String())
}