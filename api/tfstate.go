package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// TFState defines a remote TF state that must be checked
// for compliance periodically.
type TFState struct {
	Id                 string
	Timestamp          int64
	Account            string   // to categorize states
	Bucket, Path       string   // s3 bucket and item
	State              string   // the current state (in json)
	ComplianceResult   string   // the output for the compliance tool
	LastUpdate         string   // the last compliance check. "never" = not checked yet.
	S3LastModification string   // the s3 item last modification (to avoid pulling the state when it doesn't change)
	ForceValidation    bool     // if this state should be forcibly validated (omit change checks and doesn't wait)
	Tags               []string // to specify by which features this state should be checked
}

func newTFState(account string, bucket string, path string, tags []string) *TFState {
	return &TFState{
		Id:         generateId(),
		Timestamp:  generateTimestamp(),
		Account:    account,
		Bucket:     bucket,
		Path:       path,
		Tags:       tags,
		LastUpdate: "never",
	}
}

// dbObject methods

func (state *TFState) id() string {
	return state.Id
}

func (state *TFState) timestamp() int64 {
	return state.Timestamp
}

func (state *TFState) writeBasic(dst map[string]interface{}) {
	dst["account"] = state.Account
	dst["path"] = state.Path
	dst["bucket"] = state.Bucket
	dst["last_update"] = state.LastUpdate
	dst["force_validation"] = state.ForceValidation
	dst["tags"] = state.Tags
	if state.ComplianceResult == "" {
		dst["compliance_present"] = false
		return
	}
	dst["compliance_present"] = true
	parsed, _ := parseComplianceOutput(state.ComplianceResult)
	dst["compliance_errors"] = parsed.ErrorCount()
	dst["compliance_tests"] = parsed.TestCount()
	dst["compliance_features"] = parsed.featurePassed
	dst["compliance_fail_messages"] = parsed.failMessages
}

func (state *TFState) writeDetailed(dst map[string]interface{}) {
	state.writeBasic(dst)
	dst["state"] = state.State
	if _, exists := dst["compliance_present"]; exists {
		parsed, _ := parseComplianceOutput(state.ComplianceResult)
		dst["compliance_features_passed"] = parsed.featurePassed
		dst["compliance_fail_messages"] = parsed.failMessages
	}
}

// database methods

const tfStateTable = "tfstates"

var tfStateAttributes = []string{
	"Account", "Bucket", "Path", "State", "ComplianceResult",
	"LastUpdate", "S3LastModification", "ForceValidation", "Tags",
}

func (db *database) findTFStateById(id string) (*TFState, error) {
	var result *TFState = nil
	err := db.loadGeneric(
		db.tableFor(tfStateTable),
		tfStateAttributes,
		true,
		expression.Name("Id").Equal(expression.Value(id)),
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem TFState
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				result = &elem
			}
			return err
		})

	return result, err
}

func (db *database) loadAllTFStates() ([]*TFState, error) {
	var result []*TFState
	err := db.loadGeneric(
		db.tableFor(tfStateTable),
		tfStateAttributes,
		false,
		expression.ConditionBuilder{},
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem TFState
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				result = append(result, &elem)
			}
			return err
		})

	return result, err
}

func (db *database) insertOrUpdateTFState(element *TFState) error {
	return db.insertOrUpdateGeneric(db.tableFor(tfStateTable), element)
}

func (db *database) removeTFState(id string) error {
	return db.removeGeneric(db.tableFor(tfStateTable), id)
}
