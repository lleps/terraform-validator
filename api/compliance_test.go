package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseComplianceOutput(t *testing.T) {
	got := parseComplianceOutput(runComplianceExpectedOut)

	expected := ComplianceResult{
		FeaturesResult: map[string]bool{
			"credentials":  true,
			"data.example": true,
			"other":        false,
		},
		FeaturesFailures: map[string][]string{
			"credentials":  {},
			"data.example": {},
			"other": {
				"aws_instance.example (aws_instance) does not have tags property.",
				"aws_instance.example2 (resource that supports tags) does not have Name property.",
				"aws_instance.example2 (resource that supports tags) does not have application property.",
				"aws_instance.example2 (resource that supports tags) does not have role property.",
				"aws_instance.example2 (resource that supports tags) does not have environment property.",
			},
		},
	}
	assert.Equal(t, expected.FeaturesResult, got.FeaturesResult, "bad FeaturesResult")
	assert.Equal(t, expected.FeaturesFailures, got.FeaturesFailures, "bad FeaturesFailures")
	assert.Equal(t, 1, got.ErrorCount(), "bad ErrorCount")
	assert.Equal(t, 2, got.PassedCount(), "bad PassedCount")
	assert.Equal(t, 3, got.TestCount(), "bad TestCount")
}

const runComplianceExpectedOut = `
terraform-compliance v1.0.37 initiated

. Converting terraform plan file.
* Features  : /media/lleps/Compartido/Dev/terraform-compliance/example/example_01/aws
* Plan File : /media/lleps/Compartido/Dev/terraform-validator/plan.out.json

. Running tests.
Feature: Credentials should not be within the code  # /media/lleps/Compartido/Dev/terraform-compliance/example/example_01/aws/credentials.feature
    In order to prevent any credentials leakage
    As engineers
    We'll enforce credentials will not be hardcoded

    Scenario Outline: AWS Credentials should not be hardcoded
        Given I have aws provider configured
        When it contains <key>
        Then its value must not match the "<regex>" regex

    Examples:
        | key        | regex                                                   |
        SKIPPING: Skipping the step since provider type does not have access_key property.
        | access_key | (?<![A-Z0-9])[A-Z0-9]{20}(?![A-Z0-9])                   |
        SKIPPING: Skipping the step since provider type does not have secret_key property.
        | secret_key | (?<![A-Za-z0-9/+=])[A-Za-z0-9/+=]{40}(?![A-Za-z0-9/+=]) |

Feature: Data example feature  # /media/lleps/Compartido/Dev/terraform-compliance/example/example_01/aws/data.example.feature

    Scenario: Subnet Count
        SKIPPING: Can not find aws_availability_zones data defined in target terraform plan.
        Given I have aws_availability_zones data defined
        When it contains zone_ids
        And I count them
        Then I expect the result is greater than 2

Feature: Resources should be properly tagged  # /media/lleps/Compartido/Dev/terraform-compliance/example/example_01/aws/other.feature
    In order to keep track of resource ownership
    As engineers
    We'll enforce tagging on all resources

    Scenario: Ensure all resources have tags
        Given I have resource that supports tags defined
        Then it must contain tags
          Failure: aws_instance.example (aws_instance) does not have tags property.
        And its value must not be null

    Scenario Outline: Ensure that specific tags are defined
        Given I have resource that supports tags defined
        When it contains tags
        Then it must contain <tags>
        And its value must match the "<value>" regex

    Examples:
        | tags        | value            |
        | Name        | .+               |
          Failure: aws_instance.example2 (resource that supports tags) does not have Name property.
        | application | .+               |
          Failure: aws_instance.example2 (resource that supports tags) does not have application property.
        | role        | .+               |
          Failure: aws_instance.example2 (resource that supports tags) does not have role property.
        | environment | ^(prod|uat|dev)$ |
          Failure: aws_instance.example2 (resource that supports tags) does not have environment property.
`
