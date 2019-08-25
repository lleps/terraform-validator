// This file provides an easy-to-use interface to store and
// retrieve some defined items from a dynamoDB database, without
// having to write dynamoDB-specific code.

package main

import (
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

// ComplianceFeature stores a feature to test terraform code against.
type ComplianceFeature struct {
	FeatureName   string
	FeatureSource string
}

// ValidationLog stores a validation event information.
type ValidationLog struct {
	DateTime      string // when this plan was validated
	InputJson     string // the plan file json
	Output        string // the compliance tool raw output
	WasSuccessful bool   // if the compliance tool executed properly
	FailedCount   int    // the number of scenarios failed (if WasSuccessful)
	SkippedCount  int    // the number of scenarios skipped (if WasSuccessful)
	PassedCount   int    // the number of scenarios passed (if WasSuccessful)
}

type dynamoDBFeaturesTable struct {
	svc       *dynamodb.DynamoDB
	tableName string
}

// TODO: should not be feature-specific, as should also save logs.
// newDynamoDBFeaturesTable creates a DynamoDB instance using the default aws authentication method.
func newDynamoDBFeaturesTable(tableName string) dynamoDBFeaturesTable {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return dynamoDBFeaturesTable{dynamodb.New(sess), tableName}
}

// ensureTableExists will create the DynamoDB table if it does not exists.
func (ddb dynamoDBFeaturesTable) ensureTableExists() error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("FeatureName"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("FeatureName"),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		TableName: aws.String(ddb.tableName),
	}

	_, err := ddb.svc.CreateTable(input)
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
		log.Printf("Sleep 10 sec to wait until table '%s' is created in DynamoDB...", ddb.tableName)
		time.Sleep(10 * time.Second)
		log.Printf("Done!")
	}

	return nil
}

// insertOrUpdate inserts or updates the given feature on the table.
func (ddb dynamoDBFeaturesTable) insertOrUpdate(feature ComplianceFeature) error {
	av, err := dynamodbattribute.MarshalMap(feature)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(ddb.tableName),
	}
	_, err = ddb.svc.PutItem(input)
	if err != nil {
		return err
	}

	return nil
}

// loadAll returns all features currently in the table.
func (ddb dynamoDBFeaturesTable) loadAll() ([]ComplianceFeature, error) {
	// Create a projection (which "columns" we want to read)
	proj := expression.NamesList(expression.Name("FeatureName"), expression.Name("FeatureSource"))
	expr, err := expression.NewBuilder().WithProjection(proj).Build()
	if err != nil {
		return nil, err
	}

	// Build the query
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(ddb.tableName),
	}

	// Exec the request
	result, err := ddb.svc.Scan(params)
	if err != nil {
		return nil, err
	}

	// parse result into []ComplianceFeature
	var featuresParsed []ComplianceFeature
	for _, i := range result.Items {
		item := ComplianceFeature{}
		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			return nil, err
		}

		featuresParsed = append(featuresParsed, item)
	}

	return featuresParsed, nil
}

// removeByName removes all features whose FeatureName equals name.
func (ddb dynamoDBFeaturesTable) removeByName(name string) error {
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"FeatureName": {
				S: aws.String(name),
			},
		},
		TableName: aws.String(ddb.tableName),
	}

	_, err := ddb.svc.DeleteItem(input)
	if err != nil {
		return err
	}

	return nil
}