package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	sess := session.Must(session.NewSession())
	expected := []*ComplianceFeature{{"abc", "123"}, {"jjj", "456"}}
	ddb := newDynamoDB(sess, "terraformvalidator_test")
	require.Nil(t, ddb.initTables(), "1: insertion: ensureTableExists")
	for _, f := range expected {
		require.Nil(t, ddb.insertOrUpdateFeature(f), "1: insertion: insertOrUpdateFeature")
	}

	// 2. Loading
	got, err := ddb.loadAllFeatures()
	require.Nil(t, err, "2: loading")
	assert.Equal(t, expected, got, "2: loading")

	// 3. Removing
	require.Nil(t, ddb.removeFeature("abc"), "3: removing")
	expected = expected[1:]

	// 4. Removing check
	got, err = ddb.loadAllFeatures()
	require.Nil(t, err, "4: removing check")
	assert.Equal(t, expected, got, "4: removing check")

	// 5. Updating
	expected[0] = &ComplianceFeature{"jjj", "999"}
	require.Nil(t, ddb.insertOrUpdateFeature(expected[0]), "4: removing check")

	// 6. Updating check
	got, err = ddb.loadAllFeatures()
	require.Nil(t, err, "5: updating check")
	assert.Equal(t, expected, got, "5: updating check")
}
