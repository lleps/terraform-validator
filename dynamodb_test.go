package main

import "testing"

func TestDynamoDB(t *testing.T) {
	// Insert in the DB some features
	features := []ComplianceFeature { {"abc", "123"}, {"jjj", "456"}, }
	tableName := "tf-compliance-features-test" // TODO: append "-test". Should create the table clientside.
	svc, err := createDynamoDBClientAndTable(tableName)
	if err != nil {
		t.Fatalf("Can't create dynamoDB client or table: %v", err)
	}

	for _, f := range features {
		err := insertFeatureInDynamoDB(svc, tableName, f)
		if err != nil {
			t.Fatalf("err inserting: %v", err)
		}
	}
	t.Logf("Inserted %d features.", len(features))

	// Now fetch them, and ensure they're the same as the inserted ones.
	loadedFeatures, err := loadAllFeaturesFromDynamoDB(svc, tableName)
	if err != nil {
		t.Fatalf("err loading: %v", err)
	}

	if len(loadedFeatures) != len(features) {
		t.Fatalf("error: len(loadedFeatures): %d, len(features): %d.\nloadedFeatures: %v",
			len(loadedFeatures), len(features), loadedFeatures)
	}

	for i, f := range features {
		if f != loadedFeatures[i] {
			t.Errorf("Feature mismatch at idx %d. Loaded: %v. Original: %v", i, loadedFeatures[i], f)
		}
	}
}