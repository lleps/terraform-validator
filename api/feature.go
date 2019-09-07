package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// ComplianceFeature stores a feature to test terraform code against.
type ComplianceFeature struct {
	Id            string // name of the feature
	FeatureSource string // gherkin source code of the feature
}

// restObject methods

func (f *ComplianceFeature) id() string {
	return f.Id
}

func (f *ComplianceFeature) topLevel() string {
	return f.Id
}

func (f *ComplianceFeature) details() string {
	return f.FeatureSource
}

// Database methods

const complianceFeatureTable = "features"

var complianceFeatureAttributes = []string{"FeatureSource"}

func (db *database) loadAllFeatures() ([]*ComplianceFeature, error) {
	var result []*ComplianceFeature
	err := db.loadAllGeneric(
		db.tableFor(complianceFeatureTable),
		complianceFeatureAttributes,
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem ComplianceFeature
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				result = append(result, &elem)
			}
			return err
		})

	return result, err
}

func (db *database) insertOrUpdateFeature(feature *ComplianceFeature) error {
	return db.insertOrUpdateGeneric(db.tableFor(complianceFeatureTable), feature)
}

func (db *database) removeFeature(id string) error {
	return db.removeGeneric(db.tableFor(complianceFeatureTable), id)
}
