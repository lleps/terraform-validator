package main

import (
	"bytes"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/sergi/go-diff/diffmatchpatch"
	"html"
	"strings"
)

// ValidationLog stores a validation event information.
type ValidationLog struct {
	Id                   string
	Timestamp            int64
	Kind                 string           // "tfstate" or "validation"
	StateJSON            string           // current state json
	ComplianceResult     ComplianceResult // current compliance result
	PrevStateJSON        string           // for Kind tfstate, the previous state json.
	PrevComplianceResult ComplianceResult // For Kind tfstate, the previous compliance result
	Account              string           // For kind tfstate, the account affected.
	Details              string           // For kind tfstate, is bucket:path
}

const (
	logKindValidation = "validation"
	logKindTFState    = "tfstate"
)

func newValidationLog(inputJSON string, complianceResult ComplianceResult) *ValidationLog {
	return &ValidationLog{
		Id:               generateId(),
		Timestamp:        generateTimestamp(),
		Kind:             logKindValidation,
		StateJSON:        inputJSON,
		ComplianceResult: complianceResult,
	}
}

func newTFStateLog(
	stateJSON string,
	complianceResult ComplianceResult,
	prevStateJSON string,
	prevComplianceResult ComplianceResult,
	account string,
	bucket string,
	path string,
) *ValidationLog {
	return &ValidationLog{
		Id:                   generateId(),
		Timestamp:            generateTimestamp(),
		Kind:                 logKindTFState,
		StateJSON:            stateJSON,
		ComplianceResult:     complianceResult,
		PrevStateJSON:        prevStateJSON,
		PrevComplianceResult: prevComplianceResult,
		Account:              account,
		Details:              bucket + ":" + path,
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
	dst["compliance_result"] = l.ComplianceResult
	dst["prev_compliance_result"] = l.PrevComplianceResult
	dst["account"] = l.Account
	dst["details"] = l.Details
}

func (l *ValidationLog) writeDetailed(dst map[string]interface{}) {
	l.writeBasic(dst)

	// In detailed, write state diff too (as a html string). Its pretty implementation-dependent.
	// But the easier way to make this html diff is through this nice lib diffmatchpatch.
	// May send the two states and calculate the diff in the frontend. A bit more fexible.
	diff := diffmatchpatch.New()
	diffs := diff.DiffMain(l.PrevStateJSON, l.StateJSON, false)
	result := diffsToPrettyHtml(diff, diffs)            // as html
	result = strings.Replace(result, "	", "&emsp;", -1) // Replace regular tabs with html tabs
	dst["state_diff_html"] = result
}

// diffsToPrettyHtml converts a []Diff into a pretty HTML report.
func diffsToPrettyHtml(_ *diffmatchpatch.DiffMatchPatch, diffs []diffmatchpatch.Diff) string {
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

var validationLogAttributes = []string{"Kind", "StateJSON", "ComplianceResult", "Account", "Details", "PrevStateJSON", "PrevComplianceResult"}

func (db *database) loadAllLogs() ([]*ValidationLog, error) {
	var result []*ValidationLog
	err := db.loadGeneric(
		db.tableFor(validationLogTable),
		validationLogAttributes,
		false,
		expression.ConditionBuilder{},
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

func (db *database) findLogById(id string) (*ValidationLog, error) {
	var result *ValidationLog = nil
	err := db.loadGeneric(
		db.tableFor(validationLogTable),
		validationLogAttributes,
		true,
		expression.Name("Id").Equal(expression.Value(id)),
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem ValidationLog
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				result = &elem
			}
			return err
		})

	return result, err
}

func (db *database) saveLog(element *ValidationLog) error {
	return db.insertOrUpdateGeneric(db.tableFor(validationLogTable), element)
}

func (db *database) removeLog(id string) error {
	return db.removeGeneric(db.tableFor(validationLogTable), id)
}
