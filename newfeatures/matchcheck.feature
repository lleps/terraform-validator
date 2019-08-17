Feature: VPC Peering needs to follow compliance rules
  In order to have a valid VPC Peering
  As engineers
  We'll enforce vpc peering and routes have right information

  Scenario: Owner and Peer Validation
    Given I have aws_vpc defined
    And I have aws_flow_log defined
    When its vpc_id is aws_vpc.id
    Then it fails
