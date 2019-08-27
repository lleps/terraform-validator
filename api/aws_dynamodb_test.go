package main

import (
	"log"
	"testing"
)

// TestDynamoDB tests the whole thing on a real AWS DynamoDB.
// Note: only persistence for features is tested.
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
	ddb := newDynamoDB("terraformvalidator_test")
	if err := ddb.initTables(); err != nil {
		t.Fatalf("ensureTableExists: %v", err)
	}

	for _, f := range features {
		if err := ddb.insertOrUpdateFeature(f); err != nil {
			t.Fatalf("inserting: %v", err)
		}
	}

	// 2. Loading
	loadedFeatures, err := ddb.loadAllFeatures()
	if err != nil {
		log.Fatalf("loading: %v", err)
	}
	assertFeaturesMatch(features, loadedFeatures, "loading", t)

	// 3. Removing
	if err := ddb.removeFeature("abc"); err != nil {
		t.Fatalf("removing: %v", err)
	}
	features = features[1:]

	// 4. Removing check
	loadedFeatures, err = ddb.loadAllFeatures()
	if err != nil {
		log.Fatalf("removing check: %v", err)
	}
	assertFeaturesMatch(features, loadedFeatures, "removing check", t)

	// 5. Updating
	features[0] = ComplianceFeature{"jjj", "999"}
	if err := ddb.insertOrUpdateFeature(features[0]); err != nil {
		log.Fatalf("update: %v", err)
	}

	// 6. Updating check
	loadedFeatures, err = ddb.loadAllFeatures()
	if err != nil {
		log.Fatalf("update check: %v", err)
	}
	assertFeaturesMatch(features, loadedFeatures, "update check", t)
}

func assertFeaturesMatch(expected []ComplianceFeature, actual []ComplianceFeature, msg string, t *testing.T) {
	if len(expected) != len(actual) {
		t.Fatalf("%s: \nlen(expected): %d != len(actual): %d.\n"+
			"expected: %v\n"+
			"actual: %v",
			msg,
			len(expected), len(actual),
			expected,
			actual)
	}

	for i, f := range expected {
		if f != actual[i] {
			t.Errorf("%s:\nFeature mismatch at idx %d. Expected: %v. Actual: %v", msg, i, f, actual[i])
		}
	}
}
