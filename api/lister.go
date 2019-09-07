// This file provides an interface for listable resources.
// Every listable resource should register itself with the
// register function in init(), and every instance of the
// listable resource should provide an ID() method.

package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Resource interface {
	ID() string
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

func ListAllResources(s *session.Session) ([]Resource, error) {
	result := make([]Resource, 0)
	for resourceType, lister := range resourceListers {
		list, err := lister(s)
		if err != nil {
			return nil, fmt.Errorf("fetching failed for resource type %s: %v", resourceType, err)
		}
		for _, e := range list {
			result = append(result, e)
		}
	}
	return result, nil
}
