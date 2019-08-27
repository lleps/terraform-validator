Feature: VPC Flow Logs must be enabled

  Scenario: Owner and Peer Validation
    Given I have aws_vpc defined
    Then aws_flow_log must be enabled
