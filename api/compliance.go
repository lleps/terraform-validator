// This file provides functionality related to the terraform-compliance tool, like
// parsing results or executing the tool. Also depends on terraform to perform
// bin-json conversions.

package main

import (
	"fmt"
	"github.com/acarl005/stripansi"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	featuresPath    = "./features"           // on which directory will save features necessary to run compliance.
	tfComplianceBin = "terraform-compliance" // path to terraform-compliance

)

// complianceOutput contains the information extracted from a compliance output.
type complianceOutput struct {
	featurePassed map[string]bool     // for each feature, true if passed or false otherwise.
	failMessages  map[string][]string // for each failed feature, lists all the error messages.
}

func (co complianceOutput) ErrorCount() int {
	result := 0
	for _, v := range co.featurePassed {
		if !v {
			result++
		}
	}
	return result
}

func (co complianceOutput) TestCount() int {
	return len(co.featurePassed)
}

func (co complianceOutput) PassedCount() int {
	return co.TestCount() - co.ErrorCount()
}

func (co complianceOutput) String() string {
	errors := co.ErrorCount()
	tests := co.TestCount()
	sb := strings.Builder{}
	sb.WriteString("Features:\n")
	failMsgs := make([]string, 0)
	for name, passed := range co.featurePassed {
		if passed {
			sb.WriteString(fmt.Sprintf("- %s %s", name, passedMsg))
		} else {
			sb.WriteString(fmt.Sprintf("- %s %s", name, failedMsg))
			for _, msg := range co.failMessages[name] {
				failMsgs = append(failMsgs, name+": "+msg)
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	if errors > 0 {
		sb.WriteString("Errors:\n")
		for _, msg := range failMsgs {
			sb.WriteString(fmt.Sprintf("- %s\n", msg))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("%s\n", failedMsg))
	} else {
		sb.WriteString(fmt.Sprintf("%s (%d tests)\n", passedMsg, tests))
	}
	return sb.String()
}

// parseComplianceOutput takes an output of the tool and extracts the useful
// information (ie which features passed and which failed) in a structured way.
func parseComplianceOutput(output string) (complianceOutput, error) {
	currentFeature := "" // current iterating feature

	result := complianceOutput{}
	result.featurePassed = make(map[string]bool)
	result.failMessages = make(map[string][]string)

	lines := strings.Split(output, "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "Feature:") {
			fields := strings.Split(l, "#")
			if len(fields) != 2 {
				return complianceOutput{}, fmt.Errorf("can't parse line: '%s': len of fields must be 2, is %d", l, len(fields))
			}

			currentFeature = strings.TrimSpace(fields[1])
			currentFeature = extractNameFromPath(currentFeature)
			currentFeature = strings.TrimSuffix(currentFeature, ".feature")
			result.featurePassed[currentFeature] = true // true. Later may encounter a failure and set to false
			result.failMessages[currentFeature] = make([]string, 0)
		} else {
			if currentFeature != "" {
				trimmed := strings.TrimSpace(l)
				if strings.HasPrefix(trimmed, "Failure:") && len(strings.Split(trimmed, ":")) == 2 {
					errorMessage := strings.TrimSpace(strings.Split(trimmed, ":")[1])

					result.featurePassed[currentFeature] = false
					result.failMessages[currentFeature] = append(result.failMessages[currentFeature], errorMessage)
				}
			}
		}
	}

	return result, nil
}

// runComplianceTool runs the tfComplianceBin against the given file content.
// fileContent may be either a json string, or a terraform binary file format.
// Returns the input and output of the tool if successful.
func runComplianceTool(fileContent []byte) (string, string, error) {
	if len(fileContent) == 0 {
		return "", "", fmt.Errorf("empty file content")
	}

	var complianceToolInput []byte

	// TODO: this conversion.. should be here? I guess in terraform.go,
	//  something like asTerraformJson(content []byte)... "asTerraformJson ensure the content is in terraform json format."
	//  also.. this will fail for terraform states, since they're
	//  json-like, but are not directly parseable...

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
		_, ok := err.(*exec.ExitError)
		if !ok { // ignore exit code errors, compliance throws them all the time.
			return "", "", fmt.Errorf("tool execution error: %v", err)
		}
	}

	return string(complianceToolInput), toolOutput, nil
}

// syncFeaturesFolderFromDB writes all the feature files that terraform-compliance requires.
func syncFeaturesFolderFromDB(db *database) error {
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

// extractNameFromPath takes the file name from the whole path,
// for example "path/to/my/file" returns "file", and "myfile" returns "myfile".
func extractNameFromPath(path string) string {
	if len(path) == 0 {
		return ""
	}

	reversed := func(s string) string {
		chars := []rune(s)
		for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
			chars[i], chars[j] = chars[j], chars[i]
		}
		return string(chars)
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