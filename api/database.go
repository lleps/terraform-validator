// This file provides an easy-to-use interface to store and
// retrieve some items from a DynamoDB database, without
// having to worry about DynamoDB-specific code.

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

type database struct {
	svc         *dynamodb.DynamoDB
	tablePrefix string
}

// newDynamoDB creates a DynamoDB instance using the default aws authentication method.
func newDynamoDB(sess *session.Session, tablePrefix string) *database {
	return &database{dynamodb.New(sess), tablePrefix}
}

// tableFor returns the full database table ({prefix}_{name}).
func (db *database) tableFor(name string) string {
	return db.tablePrefix + "_" + name
}

// initTable creates tableName on the given database if it does not exists.
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

// loadGeneric provides a generic way to load all items from a table.
func (db *database) loadGeneric(
	tableName string,
	attributes []string, // list of the item attribute names (apart from "Id" and "Timestamp")
	conditionOrNil *expression.ConditionBuilder, // an optional filter for elements
	onItemLoaded func(map[string]*dynamodb.AttributeValue) error, // called for each loaded item
) error {
	projection := expression.NamesList(expression.Name("Id"), expression.Name("Timestamp"))
	for _, attr := range attributes {
		projection = projection.AddNames(expression.Name(attr))
	}

	builder := expression.NewBuilder().WithProjection(projection)
	if conditionOrNil != nil {
		builder = builder.WithCondition(*conditionOrNil)
	}
	expr, err := builder.Build()
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

// initTables will ensure all the necessary DynamoDB tables exists.
// tables should omit the prefix.
func (db *database) initTables(tables ...string) error {
	for _, table := range tables {
		if err := db.initTable(db.tableFor(table)); err != nil {
			return err
		}
	}
	return nil
}
