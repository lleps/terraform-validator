package main

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestConvertTerraformBinToJson(t *testing.T) {
	if testing.Short() {
		t.Skipf("Don't test tfbin->json when short")
		return
	}

	planBytes, err := base64.StdEncoding.DecodeString(convertTFPlanDataB64)
	require.Nil(t, err, "cant decode plan data")

	asJson, err := convertTerraformBinToJSON(planBytes)
	require.Nil(t, err, "convertTerraformBinToJSON failed")
	assert.Equal(t, convertTFExpectedJson, asJson, "bad json")
}

func TestTrimPreIndentationLevel(t *testing.T) {
	given := `{
	"key": 123,
	"values": [
		"ids": [
			1,
			2,
			3
		]
	]
}`
	expected := `"ids": [
	1,
	2,
	3
]`
	givenLines := strings.Split(given, "\n")
	expectedLines := strings.Split(expected, "\n")
	assert.Equal(t, expectedLines, trimIndentationLevel(givenLines, 2))
}

func TestResumeDiff(t *testing.T) {
	givenLines := strings.Split(resumeDiffGiven, "\n")
	assert.Equal(t, resumeDiffExpected, resumeDiff(givenLines, 40))
}

func TestDiffBetweenTFStates(t *testing.T) {
	oldStr := `
	hey
	milan`
	newStr := `
	hey
	joe`
	addedExpected := []string{"\tjoe"}
	removedExpected := []string{"\tmilan"}
	addedGot, removedGot := diffBetweenTFStates(oldStr, newStr)
	assert.Equal(t, addedExpected, addedGot, "added")
	assert.Equal(t, removedExpected, removedGot, "removed")
}

// this one is duplicated in client_test.go. However, its
// cheaper to repeat this block of code than doing a common
// library. Its unlikely that in the future more functionality
// is going to be shared.
func TestExtractNameFromPath(t *testing.T) {
	cases := map[string]string{
		"/path/to/my/file": "file",
		"myfile":           "myfile",
	}
	for input, expected := range cases {
		assert.Equal(t, expected, extractNameFromPath(input), "for input: "+input)
	}
}

func TestParseComplianceOutput(t *testing.T) {
	got, err := parseComplianceOutput(runComplianceExpectedOut)
	require.Nil(t, err, "cant parse out")

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

// plan.out binary content as base64
const convertTFPlanDataB64 = "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"

// the plan.out output prettified
const convertTFExpectedJson = `{
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

const resumeDiffGiven = `{
	"format_version": "0.1",
	"terraform_version": "0.12.6",
	"values": {
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
						"allocation_id": null,
						"associate_with_private_ip": null,
						"association_id": "eipassoc-05285639a19feb265",
						"domain": "vpc",
						"id": "eipalloc-04061cf4360189403",
						"instance": "i-072f8ebcec3192bc9",
						"network_interface": "eni-040ac8eec3d099b5e",
						"private_dns": "ip-172-31-94-61.ec2.internal",
						"private_ip": "172.31.94.61",
						"public_dns": "ec2-3-227-221-13.compute-1.amazonaws.com",
						"public_ip": "3.227.221.13",
						"public_ipv4_pool": "amazon",
						"tags": null,
						"timeouts": null,
						"vpc": true
					},
					"depends_on": [
						"aws_instance.example"
					]
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
						"arn": "arn:aws:ec2:us-east-1:244514332448:instance/i-072f8ebcec3192bc9",
						"associate_public_ip_address": true,
						"availability_zone": "us-east-1c",
						"cpu_core_count": 1,
						"cpu_threads_per_core": 1,
						"credit_specification": [
							{
								"cpu_credits": "standard"
							}
						],
						"disable_api_termination": false,
						"ebs_block_device": [],
						"ebs_optimized": false,
						"ephemeral_block_device": [],
						"get_password_data": false,
						"host_id": null,
						"iam_instance_profile": "",
						"id": "i-072f8ebcec3192bc9",
						"instance_initiated_shutdown_behavior": null,
						"instance_state": "running",
						"instance_type": "t2.micro",
						"ipv6_address_count": 0,
						"ipv6_addresses": [],
						"key_name": "",
						"monitoring": false,
						"network_interface": [],
						"network_interface_id": null,
						"password_data": "",
						"placement_group": "",
						"primary_network_interface_id": "eni-040ac8eec3d099b5e",
						"private_dns": "ip-172-31-94-61.ec2.internal",
						"private_ip": "172.31.94.61",
						"public_dns": "ec2-52-90-76-38.compute-1.amazonaws.com",
						"public_ip": "52.90.76.38",
						"root_block_device": [
							{
								"delete_on_termination": true,
								"encrypted": false,
								"iops": 100,
								"kms_key_id": "",
								"volume_id": "vol-0d5f0b73be219f139",
								"volume_size": 8,
								"volume_type": "gp2"
							}
						],
						"security_groups": [
							"default"
						],
						"source_dest_check": true,
						"subnet_id": "subnet-77f46259",
						"tags": null,
						"tenancy": "default",
						"timeouts": null,
						"user_data": null,
						"user_data_base64": null,
						"volume_tags": {},
						"vpc_security_group_ids": [
							"sg-aa287de4"
						]
					}
				}
			]
		}
	}
}`

const resumeDiffExpected = `{
	"format_version": "0.1",
	"terraform_version": "0.12.6",
	"values": {
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
						... 15 lines omitted
					},
					"depends_on": [
						... 1 lines omitted
					]
				},
				{
					"address": "aws_instance.example",
					"mode": "managed",
					"type": "aws_instance",
					"name": "example",
					"provider_name": "aws",
					"schema_version": 1,
					"values": {
						... 50 lines omitted
					}
				}
			]
		}
	}
}`
