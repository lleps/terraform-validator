package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/sergi/go-diff/diffmatchpatch"
	"strconv"
	"strings"
)

// ValidationLog stores a validation event information.
type ValidationLog struct {
	Id            string // number of the log entry
	Kind          string // "tfstate" or "validation".
	DateTime      string // when this plan was validated
	InputJson     string // the plan file json
	Output        string // the compliance tool raw output
	Details       string // optional. For kind tfstate, is bucket:path.
	PrevInputJson string // for Kind tfstate. The previous json input.
	PrevOutput    string // For Kind tfstate. The previous compliance output.
}

const (
	logKindValidation = "validation"
	logKindTFState    = "tfstate"
)

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

}

// database methods

const validationLogTable = "logs"

var validationLogAttributes = []string{"Kind", "DateTime", "InputJson", "Output", "Details", "PrevInputJson", "PrevOutput"}

func (db *database) loadAllValidationLogs() ([]*ValidationLog, error) {
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

func (db *database) insertOrUpdateValidationLog(validationLog *ValidationLog) error {
	return db.insertOrUpdateGeneric(db.tableFor(validationLogTable), validationLog)
}

func (db *database) removeValidationLog(id string) error {
	return db.removeGeneric(db.tableFor(validationLogTable), id)
}

func (db *database) nextFreeValidationLogId() (string, error) {
	maxId := 0
	records, err := db.loadAllValidationLogs()
	if err != nil {
		return "", err
	}
	for _, record := range records {
		recordId, _ := strconv.ParseInt(record.Id, 10, 64)
		if int(recordId) > maxId {
			maxId = int(recordId)
		}
	}
	return strconv.Itoa(maxId + 1), nil
}
