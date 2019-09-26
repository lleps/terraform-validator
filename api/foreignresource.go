package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"strings"
	"time"
)

// ForeignResource defines an AWS resource that is outside terraform.
type ForeignResource struct {
	Id                  string
	DiscoveredTimestamp string // when this resource was discovered to be outside terraform
	ResourceType        string // resource type (example ec2-instance, ec2-eip)
	ResourceId          string // resource id (example i-abc123)
	ResourceDetails     string // type-specific details
	IsException         bool   // if the resource is intentionally ok being outside terraform.
}

func newForeignResource(resourceType, resourceId, resourceDetails string) *ForeignResource {
	return &ForeignResource{
		Id:                  generateId(),
		DiscoveredTimestamp: time.Now().Format(timestampFormat),
		ResourceType:        resourceType,
		ResourceId:          resourceId,
		ResourceDetails:     resourceDetails,
	}
}

// dbObject methods

func (r *ForeignResource) id() string {
	return r.Id
}

func (r *ForeignResource) topLevel() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("#%s | %s | type: %s | id: %s | is exception: %v",
		r.Id,
		r.DiscoveredTimestamp,
		r.ResourceType,
		r.ResourceId,
		r.IsException))
	return sb.String()
}

func (r *ForeignResource) details() string {
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("      Id #%s, discovered %s\n", r.Id, r.DiscoveredTimestamp))
	sb.WriteString(fmt.Sprintf("      Type: %s, Id: %s\n", r.ResourceType, r.ResourceId))
	sb.WriteString("\n")
	sb.WriteString("Resource details:\n")
	sb.WriteString(r.ResourceDetails)
	sb.WriteString("\n")
	return sb.String()
}

func (r *ForeignResource) writeTopLevelFields(dst map[string]interface{}) {
	dst["date_time"] = r.DiscoveredTimestamp
	dst["resource_id"] = r.ResourceId
	dst["resource_type"] = r.ResourceType
	dst["resource_details"] = r.ResourceDetails
	dst["is_exception"] = r.IsException
}

func (r *ForeignResource) writeDetailedFields(dst map[string]interface{}) {

}

// database methods

const foreignResourcesTable = "foreignresources"

var foreignResourcesAttributes = []string{"DiscoveredTimestamp", "ResourceType", "ResourceId", "ResourceDetails", "IsException"}

func (db *database) loadAllForeignResources() ([]*ForeignResource, error) {
	var result []*ForeignResource
	err := db.loadAllGeneric(
		db.tableFor(foreignResourcesTable),
		foreignResourcesAttributes,
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

func (db *database) insertForeignResource(element *ForeignResource) error {
	return db.insertOrUpdateGeneric(db.tableFor(foreignResourcesTable), element)
}

func (db *database) removeForeignResource(id string) error {
	return db.removeGeneric(db.tableFor(foreignResourcesTable), id)
}
