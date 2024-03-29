// This file provides functionality related to the terraform-compliance tool, like
// parsing results or executing the tool. Also depends on terraform to perform
// bin-json conversions.

package main

import (
	"fmt"
	"github.com/acarl005/stripansi"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
)

// ComplianceResult contains the information extracted from a compliance output.
type ComplianceResult struct {
	Initialized      bool                // if this struct was generated parsing something or is uninitialized
	Error            bool                // if some error occurred during parsing
	ErrorMessage     string              // if the above is true, the error
	FeaturesResult   map[string]bool     // for each feature, true if passed or false otherwise.
	FeaturesFailures map[string][]string // for each failed feature, lists all the error messages.
	PassCount        int                 // the number of tests passing
	FailCount        int                 // the number of tests failing
	TestCount        int                 // the total number of tests
}

func cmpSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func mapOfSlicesEq(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if w, ok := b[k]; !ok || !cmpSlices(v, w) {
			return false
		}
	}

	return true
}

func (co ComplianceResult) equals(other ComplianceResult) bool {
	return co.Initialized == other.Initialized &&
		co.Error == other.Error &&
		co.ErrorMessage == other.ErrorMessage &&
		reflect.DeepEqual(co.FeaturesResult, other.FeaturesResult) &&
		mapOfSlicesEq(co.FeaturesFailures, other.FeaturesFailures)
}

// parseComplianceOutput takes an output of the tool and extracts the useful
// information (ie which features passed and which failed) in a structured way.
func parseComplianceOutput(output string) ComplianceResult {
	currentFeature := "" // current iterating feature

	result := ComplianceResult{}
	result.Initialized = true
	result.FeaturesResult = make(map[string]bool)
	result.FeaturesFailures = make(map[string][]string)

	lines := strings.Split(output, "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "Feature:") {
			fields := strings.Split(l, "#")
			if len(fields) != 2 {
				continue
			}

			currentFeature = strings.TrimSpace(fields[1])
			currentFeature = extractNameFromPath(currentFeature)
			currentFeature = strings.TrimSuffix(currentFeature, ".feature")
			result.FeaturesResult[currentFeature] = true // true. Later may encounter a failure and set to false
			result.FeaturesFailures[currentFeature] = make([]string, 0)
		} else {
			if currentFeature != "" {
				trimmed := strings.TrimSpace(l)
				if strings.HasPrefix(trimmed, "Failure:") && len(strings.Split(trimmed, ":")) == 2 {
					errorMessage := strings.TrimSpace(strings.Split(trimmed, ":")[1])

					result.FeaturesResult[currentFeature] = false
					result.FeaturesFailures[currentFeature] = append(result.FeaturesFailures[currentFeature], errorMessage)
				}
			}
		}
	}

	for _, passing := range result.FeaturesResult {
		if passing {
			result.PassCount++
		} else {
			result.FailCount++
		}
		result.TestCount++
	}

	if result.TestCount == 0 {
		result.Error = true
		result.ErrorMessage = "No tests parsed.\nOutput:\n" + stripansi.Strip(output)
	}

	return result
}

// getFeaturesForTags returns only the features that
// contains any of the given tags.
func getEnabledFeaturesContainingTags(features []*ComplianceFeature, tags []string) []*ComplianceFeature {
	hasElementInCommon := func(slice1 []string, slice2 []string) bool {
		for _, s1 := range slice1 {
			for _, s2 := range slice2 {
				if s1 == s2 {
					return true
				}
			}
		}
		return false
	}

	result := make([]*ComplianceFeature, 0)
	if tags != nil {
		for _, f := range features {
			if f.Disabled || f.Tags == nil {
				continue
			}

			if hasElementInCommon(f.Tags, tags) {
				result = append(result, f)
			}
		}
	}
	return result
}

// runComplianceToolForTags runs the compliance tool using only
// the features from db that contains any of the given tags.
func runComplianceToolForTags(db *database, fileContent []byte, tags []string) (string, string, error) {
	allFeatures, err := db.loadAllFeaturesFull()
	if err != nil {
		return "", "", fmt.Errorf("can't get features from db: %v", err)
	}

	features := getEnabledFeaturesContainingTags(allFeatures, tags)
	return runComplianceTool(fileContent, features)
}

// runComplianceTool runs the tfComplianceBin against the given file content.
// fileContent may be either a json string, or a terraform binary file format.
// Returns the input and output of the tool if successful.
func runComplianceTool(fileContent []byte, features []*ComplianceFeature) (string, string, error) {
	if len(fileContent) == 0 {
		return "", "", fmt.Errorf("empty file content")
	}

	var complianceToolInput []byte

	// Only for plan.out files:
	// In case the content is not already a json (doesn't starts with "{"), may be in
	// tf bin format (like plan.out). Try to convert it to json.
	// This fails when trying to convert a .tfstate, since tfstates starts with { too.
	if fileContent[0] != '{' {
		asJson, err := convertTerraformBinToJSON(fileContent)
		if err != nil {
			return "", "", fmt.Errorf("cntent given can't be converted to json: %v", err)
		}
		complianceToolInput = []byte(asJson)
	} else {
		complianceToolInput = fileContent
	}

	// Everything written to this directory
	baseDirectory := os.TempDir() + "/" + strconv.FormatUint(rand.Uint64(), 16)
	if err := os.Mkdir(baseDirectory, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("can't make tmp dir: %v", err)
	}
	defer os.RemoveAll(baseDirectory)
	inputJSONPath := baseDirectory + "/compliance_input.json"
	featuresPath := baseDirectory + "/features"

	// Write input file
	if err := ioutil.WriteFile(inputJSONPath, complianceToolInput, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("can't create tmp file: %v", err)
	}

	// Write features directory
	if err := makeAndFillFeaturesDirectory(featuresPath, features); err != nil {
		return "", "", fmt.Errorf("can't write features to directory %s: %v", baseDirectory, err)
	}

	// run the compliance tool against the created file
	toolOutputBytes, err := exec.Command("terraform-compliance", "-p", inputJSONPath, "-f", featuresPath).CombinedOutput()
	toolOutput := stripansi.Strip(string(toolOutputBytes))
	if err != nil {
		_, ok := err.(*exec.ExitError)
		if !ok { // ignore exit code errors, compliance throws them all the time.
			return "", "", fmt.Errorf("bad tool exit code (%v) output: %v", err, toolOutput)
		}
	}

	return string(complianceToolInput), toolOutput, nil
}

// makeAndFillFeaturesDirectory writes all the feature files that terraform-compliance requires.
func makeAndFillFeaturesDirectory(path string, features []*ComplianceFeature) error {
	// Delete the directory if exists
	if err := os.RemoveAll(path); err != nil {
		if os.IsNotExist(err) {
			// ok. Not created yet
		} else {
			// somewhat with permissions maybe
			return err
		}
	}

	// Make the directory
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	// Write all feature files here
	for _, f := range features {
		filePath := path + "/" + f.Name + ".feature"
		if err := ioutil.WriteFile(filePath, []byte(f.Source), os.ModePerm); err != nil {
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
