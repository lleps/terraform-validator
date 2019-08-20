Feature: Resources that exposes to external networks should have encryption in transit
 
  In order to comply with security
 
  As engineers
 
  We'll enforce encryption at rest on resource creation
 
 
 
 
  Scenario: TLS enforcement on ELB resources
 
    Given I have AWS ELB resource defined
 
    When it contains listener
 
    Then it must contain ssl_certificate_id
