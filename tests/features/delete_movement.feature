Feature: Delete Movements Management
  As a system user
  I want to delete movements
  So that I can manage my financial transactions by removing specific occurrences

  Scenario: Delete one occurrence of recurrent movement
    Given I have a recurrent movement with amount 1500.00 and description "Rent"
    When I create the movement
    Then the operation should be successful
    When I delete the movement in month 2
    Then the operation should be successful
    When I search for movements in month 0
    Then I should find the movement with amount 1500.00
    When I search for movements in month 2
    Then I should not find any movement
    When I search for movements in month 3
    Then I should find the movement with amount 1500.00

  Scenario: Delete all future occurrences of recurrent movement
    Given I have a recurrent movement with amount 1500.00 and description "Rent"
    When I create the movement
    Then the operation should be successful
    When I delete all future occurrences from month 2
    Then the operation should be successful
    When I search for movements in month 0
    Then I should find the movement with amount 1500.00
    When I search for movements in month 2
    Then I should not find any movement
    When I search for movements in month 3
    Then I should not find any movement
