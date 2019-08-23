
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

// Layout of an entry in the DynamoDB table.
type ComplianceFeature struct {
	FeatureName string
	FeatureSource string
}

// Creates a DynamoDB client using the default authentication method.
func createDynamoDBClientAndTable(tableName string) (*dynamodb.DynamoDB, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(sess)

	// create table schema, only 2 string fields
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
		TableName: aws.String(tableName),
	}

	_, err := svc.CreateTable(input)
	if err != nil {
		errAws := err.(awserr.Error)
		if strings.Contains(errAws.Message(), "Table already exists") {
			// ignore this error.
		} else {
			return nil, err
		}
	} else {
		// The table is being created. If an upcoming query to this table follows this
		// call immediately, may fail because the table is not yet created. Wait a few seconds.
		log.Printf("Sleep 10 sec to wait until table '%s' is created in DynamoDB...", tableName)
		time.Sleep(10 * time.Second)
		log.Printf("Done!")
	}

	return svc, nil
}

// Inserts the given feature in dynamo, in the given tableName.
func insertFeatureInDynamoDB(svc *dynamodb.DynamoDB, tableName string, feature ComplianceFeature) error {
	av, err := dynamodbattribute.MarshalMap(feature)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}
	_, err = svc.PutItem(input)
	if err != nil {
		return err
	}

	return nil
}

// Read and parse into []ComplianceFeature all the features present in the given tableName.
func loadAllFeaturesFromDynamoDB(svc *dynamodb.DynamoDB, tableName string) ([]ComplianceFeature, error) {

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
		TableName:                 aws.String(tableName),
	}

	// Exec the request
	result, err := svc.Scan(params)
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