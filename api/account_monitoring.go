// This file contains the logic that monitors account resources
package main

import (
	"api/resources"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"strings"
	"time"
)

// initAccountResourcesMonitoring starts a goroutine that periodically checks if there are
// resources in the account that don't belong to any registered tfstate, and reports them.
func initAccountResourcesMonitoring(sess *session.Session, db *database) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			// Load all tfstates and current foreign resources.
			// Quick. tfstates contains maybe a lot of data,
			// but foreign resources contains only a few fields.
			tfStates, err1 := db.loadAllTFStates()
			foreignResources, err2 := db.loadAllForeignResources()
			if err1 != nil {
				log.Printf("Can't load tfstates to monitor for resources outside terraform states: %v", err1)
				continue
			}
			if err2 != nil {
				log.Printf("Can't load foreignresources to monitor for resources outside terraform states: %v", err2)
				continue
			}

			// This is the slow part.
			// Should do some kind of parallelism.
			resourceList, err := resources.ListAllResources(sess)
			if err != nil {
				log.Printf("Can't list aws resources: %v", err)
				continue
			}

			// Ensure all resources are in at least one tfstate.
			findForeignResourceEntry := func(resourceId string) *ForeignResource {
				for _, fr := range foreignResources {
					if fr.ResourceId == resourceId {
						return fr
					}
				}
				return nil
			}
			findResourceInBuckets := func(id string) *TFState {
				for _, tfstate := range tfStates {
					if strings.Contains(tfstate.State, id) {
						return tfstate
					}
				}
				return nil
			}

			// This is the fast part. Just memory accesses.

			for _, r := range resourceList {
				// For new discovered resources, should check if findResourceInBuckets. If it is,
				// insert to db and log.

				existingFr := findForeignResourceEntry(r.Resource.ID())
				resourceBucket := findResourceInBuckets(r.Resource.ID())
				if existingFr == nil {
					if resourceBucket == nil {
						fr := newForeignResource(r.Type, r.Resource.ID(), r.Resource.Details())
						if err := db.saveForeignResource(fr); err != nil {
							log.Printf("Can't insert fr: %v", err)
							continue
						}
						log.Printf("New foreign resource registered (type: %s, ID: '%s' entryID: %s)", r.Type, r.Resource.ID(), fr.Id)
					}
				} else {
					// The resource is not new. Gotta check if the resource is still foreign.
					// if it isn't, log and delete from DB.
					if resourceBucket != nil {
						// not foreign anymore. Delete this.
						if err := db.removeForeignResource(existingFr.id()); err != nil {
							log.Printf("Can't delete fr: %v", err)
							continue
						}
						log.Printf("Foreign resource #%s (%s) not foreign anymore! Found in bucket %s:%s. Deleted!",
							existingFr.id(), existingFr.ResourceId,
							resourceBucket.Bucket, resourceBucket.Path)
					}
				}
			}
		}
	}()
}
