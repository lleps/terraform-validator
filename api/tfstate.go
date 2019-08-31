package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"strconv"
	"strings"
)

// TFState defines a remote TF state that must be checked for compliance
// periodically.
type TFState struct {
	Id               string // maybe id should be 1,2,3,4 etc. to easily remove them.
	Bucket, Path     string
	State            string // the current state (in json)
	ComplianceResult string // the output for the compliance tool
	LastUpdate       string // when was updated. "never" = not checked yet.
}

// dbObject methods

func (state *TFState) id() string {
	return state.Id
}

func (state *TFState) topLevel() string {
	sb := strings.Builder{}
	lastUpdate := "never"
	if state.LastUpdate != "" {
		lastUpdate = state.LastUpdate
	}
	sb.WriteString(fmt.Sprintf("#%s | %s:%s | last updated: %s | ", state.Id, state.Bucket, state.Path, lastUpdate))
	if state.ComplianceResult == "" {
		sb.WriteString("not checked yet")
	} else {
		parsed, err := parseComplianceOutput(state.ComplianceResult)
		if err != nil {
			sb.WriteString("<can't parse compliance result: ")
			sb.WriteString(err.Error() + ">")
		} else {
			if parsed.ErrorCount() > 0 {
				sb.WriteString(fmt.Sprintf("not compliant (%d of %d failing)", parsed.ErrorCount(), parsed.TestCount()))
			} else {
				sb.WriteString(fmt.Sprintf("compliant (%d features passing)", parsed.TestCount()))
			}
		}
	}
	return sb.String()
}

func (state *TFState) details() string {
	// todo: what to show herE?
	// idk. maybe nothing for now.
	return "todo: details"
}

// database methods

const tfStateTable= "tfstates"
var tfStateAttributes = []string{"Bucket", "Path", "State", "ComplianceResult", "LastUpdate"}

func (db *database) loadAllTFStates() ([]*TFState, error) {
	var tfStates []*TFState
	err := db.loadAllGeneric(
		db.tableFor(tfStateTable),
		tfStateAttributes,
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem TFState
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				tfStates = append(tfStates, &elem)
			}
			return err
		})

	return tfStates, err
}

func (db *database) insertOrUpdateTFState(tfState *TFState) error {
	return db.insertOrUpdateGeneric(db.tableFor(tfStateTable), tfState)
}

func (db *database) removeTFState(id string) error {
	return db.removeGeneric(db.tableFor(tfStateTable), id)
}

func (db *database) nextFreeTFStateId() (string, error) {
	maxId := 0
	records, err := db.loadAllTFStates()
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