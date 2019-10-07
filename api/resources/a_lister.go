// This file provides the interface for listable resources.
// Every listable resource should register itself with the
// register function in init(), and every instance of the
// listable resource should provide an ID() method.
// Starts with a_ to appear first on directory listing.

package resources

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Resource interface {
	ID() string
	Details() string
}

type ResourceLister func(s *session.Session) ([]Resource, error)

var resourceListers = make(map[string]ResourceLister)

func register(name string, lister ResourceLister) {
	_, exists := resourceListers[name]
	if exists {
		panic(fmt.Sprintf("a resource with the name %s already exists", name))
	}

	resourceListers[name] = lister
}

type ListedResource struct {
	Type     string
	Resource Resource
}

func ListAllResources(s *session.Session) ([]ListedResource, error) {
	result := make([]ListedResource, 0)
	for resourceType, lister := range resourceListers {
		list, err := lister(s)
		if err != nil {
			return nil, fmt.Errorf("fetch failed for resource type %s: %v", resourceType, err)
		}
		for _, e := range list {
			result = append(result, ListedResource{Type: resourceType, Resource: e})
		}
	}
	return result, nil
}
