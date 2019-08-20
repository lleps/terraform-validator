Feature: Security Groups should be utilized following security best practices
 
  In order to comply with security
 
  As engineers
 
  We'll use AWS Security Groups in a least permissive mode
 
 
 
 
 
  Scenario Outline: Well-known insecure protocol exposure on Public Network for ingress traffic
 
    Given I have AWS Security Group defined
 
    When it contains ingress
 
    Then it must not have <proto> protocol and port <portNumber> for 0.0.0.0/0
 
 
 
 
  Examples: 22,23,53,3389,123,3306,1433,25,67,135-139,161,445,520,1080,190,6881-6999
 
    | ProtocolName | proto | portNumber |
 
    | Telnet       | tcp   | 23         |
 
    | SSH          | tcp   | 22         |
 
    | MySQL        | tcp   | 3306       |
 
    | MSSQL        | tcp   | 1443       |
 
    | NetBIOS      | tcp   | 139        |
 
    | RDP          | tcp   | 3389       |
 
    | Jenkins Slave| tcp   | 50000      |
 
 
 
 
  Scenario: No publicly open ports
 
    Given I have AWS Security Group defined
 
    When it contains ingress
 
    Then it must not have tcp protocol and port 1024-65535 for 0.0.0.0/0
 
 
 
 
  Scenario: Only selected ports should be publicly open
 
    Given I have AWS Security Group defined
 
    When it contains ingress
 
    Then it must only have tcp protocol and port 443 for 0.0.0.0/0
