package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/sergi/go-diff/diffmatchpatch"
	"html"
	"strings"
	"time"
)

// ValidationLog stores a validation event information.
type ValidationLog struct {
	Id            string // number of the log entry
	Kind          string // "tfstate" or "validation".
	DateTime      string // when this plan was validated
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
		Kind:      logKindValidation,
		DateTime:  time.Now().Format(timestampFormat),
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
		Kind:          logKindTFState,
		DateTime:      time.Now().Format(timestampFormat),
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

func (l *ValidationLog) topLevel() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("#%s | %s | %s | ", l.Id, l.DateTime, l.Kind))

	if l.Kind == logKindTFState { // For tf state, show line changed and compliance changes.
		added, removed := diffBetweenTFStates(l.PrevInputJson, l.InputJson)
		sb.WriteString(fmt.Sprintf("%s | +%d, -%d lines | ", l.Details, len(added), len(removed)))
		msgFunc := func(out string) string {
			parsed, _ := parseComplianceOutput(out)
			if parsed.ErrorCount() > 0 {
				return fmt.Sprintf("%s %d/%d", failedMsg, parsed.ErrorCount(), parsed.TestCount())
			} else {
				return fmt.Sprintf("%s %d/%d", passedMsg, parsed.TestCount(), parsed.TestCount())
			}
		}

		if l.PrevOutput != "" {
			sb.WriteString(msgFunc(l.PrevOutput))
			sb.WriteString(" -> ")
		}
		sb.WriteString(msgFunc(l.Output))
	} else if l.Kind == logKindValidation { // For validations, just show compliance result
		parsed, _ := parseComplianceOutput(l.Output)
		if parsed.ErrorCount() > 0 {
			sb.WriteString(fmt.Sprintf("%s %d/%d", failedMsg, parsed.ErrorCount(), parsed.TestCount()))
		} else {
			sb.WriteString(fmt.Sprintf("%s %d/%d", passedMsg, parsed.TestCount(), parsed.TestCount()))
		}
	} else {
		sb.WriteString("<invalid kind: " + l.Kind + ">")
	}
	return sb.String()
}

func (l *ValidationLog) details() string {
	sb := strings.Builder{}

	// header
	sb.WriteString("\n")
	if l.Kind == logKindTFState {
		sb.WriteString(fmt.Sprintf("            %s (at %s)          \n", l.Details, l.DateTime))
	} else {
		sb.WriteString(fmt.Sprintf("            %s (at %s)          \n", l.Kind, l.DateTime))
	}
	sb.WriteString("\n")

	// tfstate only: json difference
	if l.Kind == logKindTFState {
		sb.WriteString("Differences:")
		sb.WriteString("\n")
		diff := diffmatchpatch.New()
		diffs := diff.DiffMain(l.PrevInputJson, l.InputJson, false)
		sb.WriteString(diff.DiffPrettyText(diffs))
		sb.WriteString("\n")
	}

	// Features
	parsed, _ := parseComplianceOutput(l.Output)
	sb.WriteString(parsed.String())
	return sb.String()
}

func (l *ValidationLog) writeTopLevelFields(dst map[string]interface{}) {
	dst["kind"] = l.Kind
	dst["date_time"] = l.DateTime
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

func (l *ValidationLog) writeDetailedFields(dst map[string]interface{}) {
	l.writeTopLevelFields(dst)

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

var validationLogAttributes = []string{"Kind", "DateTime", "InputJson", "Output", "Details", "PrevInputJson", "PrevOutput"}

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
