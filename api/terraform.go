// This file provides methods that invoke external terraform commands.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/acarl005/stripansi"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	tfComplianceBin = "terraform-compliance"
	tfBin           = "terraform"
)

// convertTerraformBinToJson converts a TF file state (like plan.out) to a
// pretty json string by invoking internally "terraform show -json".
// Doesn't supports concurrent access, as uses a hardcoded temporary file.
func convertTerraformBinToJson(fileBytes []byte) (string, error) {
	// write the bytes to a tmp file
	path := os.TempDir() + "/" + "convertTfToJson.bin.tmp"
	if err := ioutil.WriteFile(path, fileBytes, os.ModePerm); err != nil {
		return "", fmt.Errorf("can't create tmp file '%s': %v", path, err)
	}
	defer os.Remove(path)

	// invoke the tool on that file
	outputBytes, err := exec.Command(tfBin, "show", "-json", path).CombinedOutput()
	if err != nil || string(outputBytes) == "" {
		return "", fmt.Errorf("can't exec the tool: %v. out: %s", err, string(outputBytes))
	}

	// prettify the json
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, outputBytes, "", "\t"); err != nil {
		return "", fmt.Errorf("can't prettify the json: %v", err)
	}

	return string(prettyJSON.Bytes()), nil
}

// parseComplianceToolOutput parses compliance tool output into a ValidationLog struct.
func parseComplianceToolOutput(output string, record *ValidationLog) {
	record.WasSuccessful = false

	for _, line := range strings.Split(output, "\n") {
		featureCount, passedCount, failedCount, skippedCount := 0, 0, 0, 0

		// "X features (X passed, X failed, X skipped)"
		count, err := fmt.Sscanf(line,
			"%d features (%d passed, %d failed, %d skipped)",
			&featureCount, &passedCount, &failedCount, &skippedCount)

		if err != nil { // above failed, maybe "X features (X passed, X skipped)"?
			count, err = fmt.Sscanf(line,
				"%d features (%d passed, %d skipped)",
				&featureCount, &passedCount, &skippedCount)
			failedCount = 0
		}

		// if any of them match, parse into record and break the loop
		if err == nil && count >= 3 {
			record.WasSuccessful = true
			record.FailedCount = failedCount
			record.PassedCount = passedCount
			record.SkippedCount = skippedCount
			break
		}
	}
}

type complianceOutput struct {
	featurePassed map[string]bool // for each feature, true if passed or false otherwise.
	failMessages map[string][]string // for each failed feature, lists all the error messages.
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
		log.Printf("Tool output: \n%s\n", toolOutput)
		return "", "", fmt.Errorf("tool execution error: %v: %s", err, toolOutput)
	}

	return string(complianceToolInput), toolOutput, nil
}