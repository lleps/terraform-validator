This is a stateless API to monitor and validate terraform code against certain policies, using
the [terraform-compliance](https://github.com/eerkunt/terraform-compliance/) 
tool to define policies. Uses DynamoDB for persistence. There's a CLI tool that wraps the API calls seamlessly.

- Allows to validate single terraform plan files against defined policies.
- Monitor any number of terraform states for changes (only states in s3 currently supported), and check if they're compliant or not.

# Endpoints

### `/validate`
To validate a single terraform plan file against the current features and check if it's compliant or not. 

### `/features`.
To list, add or remove a terraform-compliance feature (depending on the method, GET, POST, and DELETE respectively)
The syntax used to define features is specified [here](https://github.com/eerkunt/terraform-compliance/blob/master/README.md).

### `/logs`
Every validation and monitoring event adds an entry to logs. Here you can check results of /validate or
if any terraform state change is not compliant anymore, for example. Also supports GET and DELETE.

### `/tfstates`
A TFState is a monitored state. It's address is an aws bucket + path.
All registered states will be periodically checked for compliance. Also all changes will be logged.
This also supports PUT, DELETE, GET and POST for adding/removing or getting info about a state.

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
