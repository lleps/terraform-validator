package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// ComplianceFeature stores a feature to test terraform code against.
type ComplianceFeature struct {
	Id     string
	Name   string   // name of the feature
	Source string   // gherkin source code of the feature
	Tags   []string // to specify which states this feature affects
}

func newFeature(name string, source string, tags []string) *ComplianceFeature {
	return &ComplianceFeature{Id: generateId(), Name: name, Source: source, Tags: tags}
}

// restObject methods

func (f *ComplianceFeature) id() string {
	return f.Id
}

func (f *ComplianceFeature) writeBasic(dst map[string]interface{}) {
	dst["name"] = f.Name
	dst["source"] = f.Source
	dst["tags"] = f.Tags
	dst["enabled"] = true
}

func (f *ComplianceFeature) writeDetailed(dst map[string]interface{}) {
	f.writeBasic(dst)
}

// Database methods

const complianceFeatureTable = "features"

var complianceFeatureAttributes = []string{"Name", "Source", "Tags"}

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
