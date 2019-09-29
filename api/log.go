package main

import (
	"bytes"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/sergi/go-diff/diffmatchpatch"
	"html"
	"strings"
)

// ValidationLog stores a validation event information.
type ValidationLog struct {
	Id            string
	Timestamp     int64
	Kind          string // "tfstate" or "validation".
	InputJson     string // the plan file json
	Output        string // the compliance tool raw output
	Details       string // For kind tfstate, is bucket:path.
	PrevInputJson string // for Kind tfstate, the previous json input.
	PrevOutput    string // For Kind tfstate, the previous compliance output.
}

const (
	logKindValidation = "validation"
	logKindTFState    = "tfstate"
)

func newValidationLog(inputJSON string, output string) *ValidationLog {
	return &ValidationLog{
		Id:        generateId(),
		Timestamp: generateTimestamp(),
		Kind:      logKindValidation,
		InputJson: inputJSON,
		Output:    output,
	}
}

func newTFStateLog(
	inputJSON string,
	output string,
	prevInputJSON string,
	prevOutput string,
	bucket string,
	path string,
) *ValidationLog {
	return &ValidationLog{
		Id:            generateId(),
		Timestamp:     generateTimestamp(),
		Kind:          logKindTFState,
		InputJson:     inputJSON,
		Output:        output,
		PrevInputJson: prevInputJSON,
		PrevOutput:    prevOutput,
		Details:       bucket + ":" + path,
	}
}

// restObject methods

func (l *ValidationLog) id() string {
	return l.Id
}

func (l *ValidationLog) timestamp() int64 {
	return l.Timestamp
}

func (l *ValidationLog) writeBasic(dst map[string]interface{}) {
	dst["kind"] = l.Kind
	dst["details"] = l.Details
	parsed, _ := parseComplianceOutput(l.Output)
	dst["compliance_errors"] = parsed.ErrorCount()
	dst["compliance_tests"] = parsed.TestCount()
	dst["compliance_errors_prev"] = 0
	dst["compliance_tests_prev"] = 0

	if l.Kind == "tfstate" {
		added, removed := diffBetweenTFStates(l.PrevInputJson, l.InputJson)
		dst["lines_added"] = len(added)
		dst["lines_removed"] = len(removed)

		if l.PrevOutput != "" {
			parsedPrev, _ := parseComplianceOutput(l.PrevOutput)
			dst["compliance_errors_prev"] = parsedPrev.ErrorCount()
			dst["compliance_tests_prev"] = parsedPrev.TestCount()
		}
	}
}

func (l *ValidationLog) writeDetailed(dst map[string]interface{}) {
	l.writeBasic(dst)

	// Write state diff
	diff := diffmatchpatch.New()
	diffs := diff.DiffMain(l.PrevInputJson, l.InputJson, false)
	result := diffsToPrettyHtml(diff, diffs)            // as html
	result = strings.Replace(result, "	", "&emsp;", -1) // Replace regular tabs with html tabs
	dst["state_diff_html"] = result

	// Write feature details.
	// Frontend can check for prev output presence using compliance_prev == true.
	// All other fields ending with _prev are present only if l.PrevOutput != "".
	parsed, _ := parseComplianceOutput(l.Output) // TODO: err?
	dst["compliance_features"] = parsed.featurePassed
	dst["compliance_fail_messages"] = parsed.failMessages
	dst["compliance_prev"] = l.PrevOutput != ""
	if dst["compliance_prev"] == true {
		parsedPrev, _ := parseComplianceOutput(l.PrevOutput) // TODO: err?
		dst["compliance_features_prev"] = parsedPrev.featurePassed
		dst["compliance_fail_messages_prev"] = parsedPrev.failMessages
	}
}

// diffsToPrettyHtml converts a []Diff into a pretty HTML report.
func diffsToPrettyHtml(dmp *diffmatchpatch.DiffMatchPatch, diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		text := strings.Replace(html.EscapeString(diff.Text), "\n", "<br>", -1)
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString("<ins style=\"background:#1B5E20;\">")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</ins>")
		case diffmatchpatch.DiffDelete:
			_, _ = buff.WriteString("<del style=\"background:#C62828;\">")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</del>")
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString("<span>")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</span>")
		}
	}
	return buff.String()
}

// database methods

const validationLogTable = "logs"

var validationLogAttributes = []string{"Kind", "InputJson", "Output", "Details", "PrevInputJson", "PrevOutput"}

func (db *database) loadAllLogs() ([]*ValidationLog, error) {
	var result []*ValidationLog
	err := db.loadAllGeneric(
		db.tableFor(validationLogTable),
		validationLogAttributes,
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem ValidationLog
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				result = append(result, &elem)
			}
			return err
		})

	return result, err
}

func (db *database) insertLog(element *ValidationLog) error {
	return db.insertOrUpdateGeneric(db.tableFor(validationLogTable), element)
}

func (db *database) removeLog(id string) error {
	return db.removeGeneric(db.tableFor(validationLogTable), id)
}
