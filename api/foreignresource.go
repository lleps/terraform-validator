package main

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// ForeignResource defines an AWS resource that is outside terraform.
type ForeignResource struct {
	Id              string
	Timestamp       int64
	ResourceType    string // resource type (example ec2-instance, ec2-eip)
	ResourceId      string // resource id (example i-abc123)
	ResourceDetails string // type-specific details
	IsException     bool   // if the resource is intentionally ok being outside terraform.
}

func newForeignResource(resourceType, resourceId, resourceDetails string) *ForeignResource {
	return &ForeignResource{
		Id:              generateId(),
		Timestamp:       generateTimestamp(),
		ResourceType:    resourceType,
		ResourceId:      resourceId,
		ResourceDetails: resourceDetails,
	}
}

// dbObject methods

func (r *ForeignResource) id() string {
	return r.Id
}

func (r *ForeignResource) timestamp() int64 {
	return r.Timestamp
}

func (r *ForeignResource) writeBasic(dst map[string]interface{}) {
	dst["resource_id"] = r.ResourceId
	dst["resource_type"] = r.ResourceType
	dst["resource_details"] = r.ResourceDetails
	dst["is_exception"] = r.IsException
}

func (r *ForeignResource) writeDetailed(dst map[string]interface{}) {

}

// database methods

const foreignResourcesTable = "foreignresources"

var foreignResourcesAttributes = []string{"ResourceType", "ResourceId", "ResourceDetails", "IsException"}

func (db *database) loadAllForeignResources() ([]*ForeignResource, error) {
	var result []*ForeignResource
	err := db.loadGeneric(
		db.tableFor(foreignResourcesTable),
		foreignResourcesAttributes,
		false,
		expression.ConditionBuilder{},
		func(i map[string]*dynamodb.AttributeValue) error {
			var elem ForeignResource
			err := dynamodbattribute.UnmarshalMap(i, &elem)
			if err == nil {
				result = append(result, &elem)
			}
			return err
		})

	return result, err
}

func (db *database) saveForeignResource(element *ForeignResource) error {
	return db.insertOrUpdateGeneric(db.tableFor(foreignResourcesTable), element)
}

func (db *database) removeForeignResource(id string) error {
	return db.removeGeneric(db.tableFor(foreignResourcesTable), id)
}
