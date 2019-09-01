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
	sb.WriteString(fmt.Sprintf("%s #%s [%s] ", l.DateTime, l.Id, l.Kind))

	if l.Kind == logKindTFState {
		// 	+3, -4 lines. FAILING 2/6 -> FAILING 2/5
		// first calculate state diff
		added, removed := diffBetweenTFStates(l.PrevInputJson, l.InputJson)
		sb.WriteString(fmt.Sprintf("+%d, -%d lines, ", len(added), len(removed)))

		msgFunc := func(out string) string {
			parsed, err := parseComplianceOutput(out)
			if err != nil {
				return err.Error()
			}

			if parsed.ErrorCount() > 0 {
				return fmt.Sprintf("FAILING %d/%d", parsed.ErrorCount(), parsed.TestCount())
			} else {
				return fmt.Sprintf("FAILING %d/%d", parsed.TestCount(), parsed.TestCount())
			}
		}

		if l.PrevOutput != "" {
			sb.WriteString(msgFunc(l.PrevOutput))
			sb.WriteString(" -> ")
		}
		sb.WriteString(msgFunc(l.Output))
	} else if l.Kind == logKindValidation {
		parsed, err := parseComplianceOutput(l.Output)
		if err != nil {
			return sb.String() + "<can't parse output>"
		}
		if parsed.ErrorCount() > 0 {
			sb.WriteString(fmt.Sprintf("FAILED [%d of %d tests failed]", parsed.ErrorCount(), parsed.TestCount()))
		} else {
			sb.WriteString(fmt.Sprintf("PASSED [%d tests passed]", parsed.TestCount()))
		}
	} else {
		sb.WriteString("<invalid kind: " + l.Kind + ">")
	}
	return sb.String()
}

func (l *ValidationLog) details() string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("            %s (at %s)          \n", l.Kind, l.DateTime))
	sb.WriteString("\n")

	// print difference for tfstate only
	if l.Kind == logKindTFState {

		sb.WriteString("Differences:")
		sb.WriteString("\n")
		sb.WriteString("\n")
		diff := diffmatchpatch.New()
		diffs := diff.DiffMain(l.PrevInputJson, l.InputJson, false)
		sb.WriteString(diff.DiffPrettyText(diffs))
		sb.WriteString("\n")
	}

	// print features
	sb.WriteString("Features:\n")
	parsed, err := parseComplianceOutput(l.Output)
	if err != nil {
		return sb.String() + "<can't parse output: " + err.Error() + ">"
	}

	for feature, passing := range parsed.featurePassed {
		if !passing {
			sb.WriteString(fmt.Sprintf(" - %s FAILED", feature))
		} else {
			sb.WriteString(fmt.Sprintf(" - %s OK", feature))
		}
		sb.WriteRune('\n')
	}
	sb.WriteRune('\n')

	if parsed.ErrorCount() > 0 {
		sb.WriteString("Errors:\n")
		for k, errors := range parsed.failMessages {
			for _, e := range errors {
				sb.WriteString(fmt.Sprintf(" - %s: %s\n", k, e))
			}
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

// database methods

const validationLogTable = "logs"

var validationLogAttributes = []string{"Kind", "DateTime", "InputJson","Output", "PrevInputJson", "PrevOutput"}

func (db *database) loadAllValidationLogs() ([]*ValidationLog, error) {
	var validationLogs []*ValidationLog
	err := db.loadAllGeneric(
		db.tableFor(validationLogTable),
		validationLogAttributes,
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem ValidationLog
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				validationLogs = append(validationLogs, &elem)
			}
			return err
		})

	return validationLogs, err
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
