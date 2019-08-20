Feature: Resources should use encryption at rest while they are created
 
  In order to comply with security
 
  As engineers
 
  We'll enforce encryption at rest
 
 
 
 
  Scenario: RDS instances
 
    Given I have AWS RDS instance defined
 
    Then encryption at rest must be enabled
 
 
 
 
  Scenario: EBS volumes
 
    Given I have AWS EBS volume defined
 
    Then encryption at rest must be enabled
 
 
 
 
  Scenario: S3 Buckets (to be discussed)
 
    Given I have AWS S3 Bucket defined
 
    Then encryption at rest must be enabled
