package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
			res, code, resErr = execRequest(host, "/features/"+featureName, "POST", string(content))
		}

	case *featureRemoveFlag != "":
		fileWithoutExt := strings.TrimSuffix(*featureRemoveFlag, ".feature")
		res, code, resErr = execRequest(host, "/features/"+fileWithoutExt, "DELETE", "")

	case *featureDetailsFlag != "":
		res, code, resErr = execRequest(host, "/features/"+*featureDetailsFlag, "GET", "")

	// -log-*

	case *logListFlag:
		res, code, resErr = execRequest(host, "/logs", "GET", "")
	case *logRemoveFlag != "":
		res, code, resErr = execRequest(host, "/logs/"+*logRemoveFlag, "DELETE", "")
	case *logGetFlag != "":
		res, code, resErr = execRequest(host, "/logs/" + *logGetFlag, "GET", "")

	default:
		fmt.Println("No option given. Check -h to see options.")
		return
	}

	if resErr != nil {
		fmt.Println("Error during request:", resErr)
		return
	}

	if code != http.StatusOK {
		fmt.Println("Request not OK: invalid HTTP Response:", code)
		if res != "" {
			fmt.Println("Details:")
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
	content string,
) (result string, code int, err error) {
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