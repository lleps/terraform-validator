package main

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConvertTerraformBinToJson(t *testing.T) {
	if testing.Short() {
		t.Skipf("Don't test tfbin->json when short")
		return
	}

	planBytes, err := base64.StdEncoding.DecodeString(planDataB64)
	require.Nil(t, err, "cant decode plan data")

	asJson, err := convertTerraformBinToJson(planBytes)
	require.Nil(t, err, "convertTerraformBinToJson failed")
	assert.Equal(t, expectedJson, asJson, "bad json")
}

func TestParseComplianceOutput(t *testing.T) {
	got, err := parseComplianceOutput(complianceOut)
	require.Nil(t, err, "cant parse out")

	expected := complianceOutput{
		featurePassed: map[string]bool{
			"credentials": true,
			"data.example": true,
			"other": false,
		},
		failMessages: map[string][]string{
			"credentials": {},
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
	assert.Equal(t, expected.featurePassed, got.featurePassed, "bad featurePassed")
	assert.Equal(t, expected.failMessages, got.failMessages, "bad failMessages")
	assert.Equal(t, 1, got.ErrorCount(), "bad ErrorCount")
	assert.Equal(t, 2, got.PassedCount(), "bad PassedCount")
	assert.Equal(t, 3, got.TestCount(), "bad TestCount")
}

var complianceOut = `
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

// plan.out binary content as base64
var planDataB64 = "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"

// the plan.out output prettified
var expectedJson = `{
	"format_version": "0.1",
	"terraform_version": "0.12.6",
	"planned_values": {
		"root_module": {
			"resources": [
				{
					"address": "aws_eip.ip",
					"mode": "managed",
					"type": "aws_eip",
					"name": "ip",
					"provider_name": "aws",
					"schema_version": 0,
					"values": {
						"associate_with_private_ip": null,
						"tags": null,
						"timeouts": null,
						"vpc": true
					}
				},
				{
					"address": "aws_instance.example",
					"mode": "managed",
					"type": "aws_instance",
					"name": "example",
					"provider_name": "aws",
					"schema_version": 1,
					"values": {
						"ami": "ami-2757f631",
						"credit_specification": [],
						"disable_api_termination": null,
						"ebs_optimized": null,
						"get_password_data": false,
						"iam_instance_profile": null,
						"instance_initiated_shutdown_behavior": null,
						"instance_type": "t2.micro",
						"monitoring": null,
						"source_dest_check": true,
						"tags": null,
						"timeouts": null,
						"user_data": null,
						"user_data_base64": null
					}
				}
			]
		}
	},
	"resource_changes": [
		{
			"address": "aws_eip.ip",
			"mode": "managed",
			"type": "aws_eip",
			"name": "ip",
			"provider_name": "aws",
			"change": {
				"actions": [
					"create"
				],
				"before": null,
				"after": {
					"associate_with_private_ip": null,
					"tags": null,
					"timeouts": null,
					"vpc": true
				},
				"after_unknown": {
					"allocation_id": true,
					"association_id": true,
					"domain": true,
					"id": true,
					"instance": true,
					"network_interface": true,
					"private_dns": true,
					"private_ip": true,
					"public_dns": true,
					"public_ip": true,
					"public_ipv4_pool": true
				}
			}
		},
		{
			"address": "aws_instance.example",
			"mode": "managed",
			"type": "aws_instance",
			"name": "example",
			"provider_name": "aws",
			"change": {
				"actions": [
					"create"
				],
				"before": null,
				"after": {
					"ami": "ami-2757f631",
					"credit_specification": [],
					"disable_api_termination": null,
					"ebs_optimized": null,
					"get_password_data": false,
					"iam_instance_profile": null,
					"instance_initiated_shutdown_behavior": null,
					"instance_type": "t2.micro",
					"monitoring": null,
					"source_dest_check": true,
					"tags": null,
					"timeouts": null,
					"user_data": null,
					"user_data_base64": null
				},
				"after_unknown": {
					"arn": true,
					"associate_public_ip_address": true,
					"availability_zone": true,
					"cpu_core_count": true,
					"cpu_threads_per_core": true,
					"credit_specification": [],
					"ebs_block_device": true,
					"ephemeral_block_device": true,
					"host_id": true,
					"id": true,
					"instance_state": true,
					"ipv6_address_count": true,
					"ipv6_addresses": true,
					"key_name": true,
					"network_interface": true,
					"network_interface_id": true,
					"password_data": true,
					"placement_group": true,
					"primary_network_interface_id": true,
					"private_dns": true,
					"private_ip": true,
					"public_dns": true,
					"public_ip": true,
					"root_block_device": true,
					"security_groups": true,
					"subnet_id": true,
					"tenancy": true,
					"volume_tags": true,
					"vpc_security_group_ids": true
				}
			}
		}
	],
	"configuration": {
		"provider_config": {
			"aws": {
				"name": "aws",
				"expressions": {
					"profile": {
						"constant_value": "default"
					},
					"region": {
						"constant_value": "us-east-1"
					}
				}
			}
		},
		"root_module": {
			"resources": [
				{
					"address": "aws_eip.ip",
					"mode": "managed",
					"type": "aws_eip",
					"name": "ip",
					"provider_config_key": "aws",
					"expressions": {
						"instance": {
							"references": [
								"aws_instance.example"
							]
						},
						"vpc": {
							"constant_value": true
						}
					},
					"schema_version": 0
				},
				{
					"address": "aws_instance.example",
					"mode": "managed",
					"type": "aws_instance",
					"name": "example",
					"provider_config_key": "aws",
					"expressions": {
						"ami": {
							"constant_value": "ami-2757f631"
						},
						"instance_type": {
							"constant_value": "t2.micro"
						}
					},
					"schema_version": 1
				}
			]
		}
	}
}
`
