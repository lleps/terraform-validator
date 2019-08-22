This tool is used to manipulate the [terraform-compliance](https://github.com/eerkunt/terraform-compliance/) 
tool and its state through a REST API, allowing devs to test terraform 
plan files against specific security requirements.

It contains a client to execute the requests, and a server which wraps
the tool and implements the API.

# API specification

### `POST /validate`
Validate a terraform plan file with the current features. The plan file
content must be passed as a base64 string in the body.

### `GET /features`
List all the used features.

### `GET /features/source/{name}`
Get the source code of the `name` feature.

### `POST /features/add/{name}`
Add a new feature with `name`. The feature source code is also passed
in the request body, as a raw string.

The syntax used to describe features is specified [here](https://github.com/eerkunt/terraform-compliance/blob/master/README.md).

New calls to /validate will use this new feature to test the plan file.

### `DELETE /features/delete/{name}`
Delete the `name` feature, so it's not used to test plan
files anymore.

# Features to be added
1) AWS Credentials should not be hardcoded. 
2) VPC Flow Logs should be enabled 
3) Remove all rules associated with default route tables, ACLs, SGs, in the default VPC in all regions
4) Resources should use encryption when they are created
5) Resources should use encryption in transit 
6) Resources should follow basic naming standards
7) S3 buckets should not be public unless approved by Security Engineering 
8) S3 buckets should have logging enabled
9) Security groups should not be overly open. Exposure of insecure protocols should be limited for ingress traffic. Only selected ports and approved by Security Engineering should be open
10) CloudTrail must be enabled in all Regions 
11) GuardDuty must be enabled in all VPCs, in all regions
