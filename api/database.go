// This file provides an easy-to-use interface to store and
// retrieve some defined items from a database, without
// having to worry about dynamo-specific code.

package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"log"
	"strings"
	"time"
)

// Item type definitions. Every persistent item should have an "Id"!

// ComplianceFeature stores a feature to test terraform code against.
type ComplianceFeature struct {
	Id            string // name of the feature
	FeatureSource string // gherkin source code of the feature
}

func (f *ComplianceFeature) id() string {
	return f.Id
}

func (f *ComplianceFeature) topLevel() string {
	return f.Id
}

func (f *ComplianceFeature) details() string {
	return f.FeatureSource
}

// ValidationLog stores a validation event information.
type ValidationLog struct {
	Id            string // number of the log entry
	DateTime      string // when this plan was validated
	InputJson     string // the plan file json
	Output        string // the compliance tool raw output
}

func (l *ValidationLog) id() string {
	return l.Id
}

func (l *ValidationLog) topLevel() string {
	return fmt.Sprintf("#%s [%s]", l.Id, l.DateTime)
}

func (l *ValidationLog) details() string {
	return "details"
}

// defines table names for each type
const (
	complianceFeatureTable = "features"
	validationLogTable     = "logs"
)

// defines the attributes for each type, used to build projections in dynamo.
var (
	complianceFeatureAttributes = []string{"FeatureSource"}
	validationLogAttributes     = []string{"DateTime", "InputJson", "Output"}
)

type database struct {
	svc         *dynamodb.DynamoDB
	tablePrefix string
}

// newDynamoDB creates a DynamoDB instance using the default aws authentication method.
func newDynamoDB(tablePrefix string) *database {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return &database{dynamodb.New(sess), tablePrefix}
}

// Generic table methods. Those should not be used outside this file.
// Instead, type-specific methods (defined below) should be used.

// tableFor returns the full database table ({prefix}_{name}).
func (db *database) tableFor(name string) string {
	return db.tablePrefix + "_" + name
}

// initTable creates tableName on the given database session if it does not exists.
func (db *database) initTable(tableName string) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Id"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Id"),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		TableName: aws.String(tableName),
	}

	_, err := db.svc.CreateTable(input)
	if err != nil {
		errAws := err.(awserr.Error)
		if strings.Contains(errAws.Message(), "Table already exists") {
			// ignore this error.
		} else {
			return err
		}
	} else {
		// The table is being created. If an upcoming query to this table follows this
		// call immediately, may fail because the table is not yet created. Wait a few seconds.
		log.Printf("Sleep 5 sec to wait until table '%s' is created in DynamoDB...", tableName)
		time.Sleep(5 * time.Second)
		log.Printf("Done!")
	}

	return nil
}

// insertOrUpdateGeneric inserts or updates the given item on the table.
func (db *database) insertOrUpdateGeneric(tableName string, item interface{}) error {
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}
	_, err = db.svc.PutItem(input)
	if err != nil {
		return err
	}

	return nil
}

// loadAllGeneric provides a generic way to load all items from a table.
func (db *database) loadAllGeneric(
	tableName string,
	attributes []string, // list of the item attribute names (apart from "Id")
	onItemLoaded func(map[string]*dynamodb.AttributeValue) error, // called for each loaded item
) error {
	projection := expression.NamesList(expression.Name("Id"))
	for _, attr := range attributes {
		projection = projection.AddNames(expression.Name(attr))
	}

	expr, err := expression.NewBuilder().WithProjection(projection).Build()
	if err != nil {
		return err
	}

	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(tableName),
	}

	result, err := db.svc.Scan(params)
	if err != nil {
		return err
	}

	for _, i := range result.Items {
		if err := onItemLoaded(i); err != nil {
			return nil
		}
	}
	return nil
}

// removeGeneric removes all the items in the given table whose Id equals id.
func (db *database) removeGeneric(tableName string, id string) error {
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Id": {
				S: aws.String(id),
			},
		},
		TableName: aws.String(tableName),
	}

	_, err := db.svc.DeleteItem(input)
	if err != nil {
		return err
	}

	return nil
}

// Type-Specific method definitions.

// initTables will ensure all the necessary DynamoDB tables exists.
func (db *database) initTables() error {
	if err := db.initTable(db.tableFor(complianceFeatureTable)); err != nil {
		return err
	}
	if err := db.initTable(db.tableFor(validationLogTable)); err != nil {
		return err
	}
	return nil
}

// insertOrUpdateFeature inserts or updates the given feature on the database.
func (db *database) insertOrUpdateFeature(feature *ComplianceFeature) error {
	return db.insertOrUpdateGeneric(db.tableFor(complianceFeatureTable), feature)
}

// insertOrUpdateValidationLog inserts or updates the given validation log on the database.
func (db *database) insertOrUpdateValidationLog(validationLog *ValidationLog) error {
	return db.insertOrUpdateGeneric(db.tableFor(validationLogTable), validationLog)
}

// loadAllFeatures returns all the ComplianceFeature items on the database.
func (db *database) loadAllFeatures() ([]*ComplianceFeature, error) {
	var features []*ComplianceFeature
	err := db.loadAllGeneric(
		db.tableFor(complianceFeatureTable),
		complianceFeatureAttributes,
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem ComplianceFeature
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				features = append(features, &elem)
			}
			return err
		})

	return features, err
}

// loadAllValidationLogs returns all the ValidationLog items on the database.
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

// removeFeature removes the first feature whose Id is id.
func (db *database) removeFeature(id string) error {
	return db.removeGeneric(db.tableFor(complianceFeatureTable), id)
}

// removeValidationLog removes the first validation log whose Id is id.
func (db *database) removeValidationLog(id string) error {
	return db.removeGeneric(db.tableFor(validationLogTable), id)
}
