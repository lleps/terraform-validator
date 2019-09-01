// This file provides methods that invoke external terraform commands.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/acarl005/stripansi"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	tfComplianceBin = "terraform-compliance"
	tfBin           = "terraform"
	passedMsg       = "[32mPASSED[0m"
	failedMsg       = "[31mFAILED[0m"
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

// diffBetweenTFStates returns the list of added and removed lines in the newJson, relative to oldJson.
func diffBetweenTFStates(oldJson, newJson string) (added []string, removed []string) {
	sliceContains := func(element string, slice []string) bool {
		for _, e := range slice {
			if e == element {
				return true
			}
		}
		return false
	}
	oldLines := strings.Split(oldJson, "\n")
	newLines := strings.Split(newJson, "\n")

	// lines in new, but not in old
	added = make([]string, 0)
	for _, line := range newLines {
		if !sliceContains(line, oldLines) {
			added = append(added, line)
		}
	}

	// lines in old, but not in new
	removed = make([]string, 0)
	for _, line := range oldLines {
		if !sliceContains(line, newLines) {
			removed = append(removed, line)
		}
	}
	return
}

// getLineIndentationLevel returns how deeply indented is a line.
func getLineIndentationLevel(s string) (deep int) {
	for _, c := range s {
		if c != '\t' {
			break
		}
		deep++
	}
	return
}

// trimIndentationLevel only keeps the lines at level
// or more levels of indentations.
func trimIndentationLevel(lines []string, level int) []string {
	result := make([]string, 0)
	sb := strings.Builder{}
	for i := 0; i < level; i++ {
		sb.WriteRune('\t')
	}
	trimmedIndentation := sb.String()
	for _, line := range lines {
		depth := getLineIndentationLevel(line)
		if depth >= level {
			result = append(result, strings.TrimPrefix(line, trimmedIndentation))
		}
	}
	return result
}

// resumeDiff makes sure the given diff (ie lines of json) is
// at least limit characters, by compacting deeper levels
// as ... n fields omitted.
func resumeDiff(lines []string, limit int) []string {
	// get deepest level
	maxDepth := 0
	for _, line := range lines {
		depth := getLineIndentationLevel(line)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	result := lines // on which array to iterate.
	for {
		// we're done if this is true.
		if len(result) <= limit {
			return result
		}

		// decrease tolerance
		maxDepth--
		if maxDepth <= 0 {
			return result
		}

		// add here only lines whose depth <= maxDepth
		newResult := make([]string, 0)
		ignoredLines := 0 // counter for "omitted fields".
		for _, line := range result {
			depth := getLineIndentationLevel(line)
			if depth > maxDepth {
				ignoredLines++
			} else {
				if ignoredLines > 0 {
					// maybe just ignore those. later may be added.
					sb := strings.Builder{}
					for i := 0; i < maxDepth + 1; i++ {
						sb.WriteRune('\t')
					}
					sb.WriteString(fmt.Sprintf("... %d lines omitted", ignoredLines))
					newResult = append(newResult, sb.String())
					ignoredLines = 0
				}
				newResult = append(newResult, line)
			}
		}
		result = newResult
	}
}

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
				failMsgs = append(failMsgs, name + ": " + msg)
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
		_, ok := err.(*exec.ExitError)
		if !ok { // ignore exit code errors, compliance throws them all the time.
			return "", "", fmt.Errorf("tool execution error: %v", err)
		}
	}

	return string(complianceToolInput), toolOutput, nil
}
