// This file provides functionality specific to terraform.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	tfBin     = "terraform"
	passedMsg = "[32mPASSED[0m"
	failedMsg = "[31mFAILED[0m"
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
					for i := 0; i < maxDepth+1; i++ {
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
