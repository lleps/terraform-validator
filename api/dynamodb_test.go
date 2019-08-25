package main

import (
	"log"
	"testing"
)

// TestDynamoDB tests the whole thing on a real AWS DynamoDB.
// Steps:
// 1. Insertion: Inserts a few features on DB.
// 2. Loading: Load all the inserted features (and check if they match the inserted ones)
// 3. Removing: Remove one of the features.
// 4. Removing check: Load them again and ensure the feature is not present anymore.
// 5. Updating: Change the source of a feature.
// 6. Updating check: Load the feature and see if the source changed.
func TestDynamoDB(t *testing.T) {
	if testing.Short() {
		t.Skipf("Don't run full DynamoDB test when short")
		return
	}

	// 1. Insertion
	features := []ComplianceFeature{{"abc", "123"}, {"jjj", "456"}}
	ddb := newDynamoDBFeaturesTable("tf-compliance-features-test")
	if err := ddb.ensureTableExists(); err != nil {
		t.Fatalf("ensureTableExists: %v", err)
	}

	for _, f := range features {
		if err := ddb.insertOrUpdate(f); err != nil {
			t.Fatalf("inserting: %v", err)
		}
	}

	// 2. Loading
	loadedFeatures, err := ddb.loadAll()
	if err != nil {
		log.Fatalf("loading: %v", err)
	}
	assertFeaturesMatch(features, loadedFeatures, t)

	// 3. Removing
	if err := ddb.removeByName("abc"); err != nil {
		t.Fatalf("removing: %v", err)
	}
	features = features[1:]

	// 4. Removing check
	loadedFeatures, err = ddb.loadAll()
	if err != nil {
		log.Fatalf("removing check: %v", err)
	}
	assertFeaturesMatch(features, loadedFeatures, t)

	// 5. Updating
	features[0] = ComplianceFeature{"jjj", "999"}
	if err := ddb.insertOrUpdate(features[0]); err != nil {
		log.Fatalf("update: %v", err)
	}

	// 6. Updating check
	loadedFeatures, err = ddb.loadAll()
	if err != nil {
		log.Fatalf("update check: %v", err)
	}
	assertFeaturesMatch(features, loadedFeatures, t)
}

func assertFeaturesMatch(expected []ComplianceFeature, actual []ComplianceFeature, t *testing.T) {
	if len(expected) != len(actual) {
		t.Fatalf("len(expected): %d != len(actual): %d.\n"+
			"expected: %v\n"+
			"actual: %v",
			len(expected), len(actual),
			expected,
			actual)
	}

	for i, f := range expected {
		if f != actual[i] {
			t.Errorf("Feature mismatch at idx %d. Expected: %v. Actual: %v", i, f, actual[i])
		}
	}
}
