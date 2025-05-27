Feature: Update and Create Movements Management
  As a system user
  I want to create and update recurrent movements
  So that I can manage my recurring financial transactions

  Scenario: Create recurrent movement and validate future occurrences
    Given I have a recurrent movement with amount 2000.00 and description "Salary"
    When I create the movement
    Then the operation should be successful
    When I search for movements in month 0
    Then I should find the movement with amount 2000.00
    When I search for movements in month 2
    Then I should find the movement with amount 2000.00

  Scenario: Update one occurrence of recurrent movement
    Given I have a recurrent movement with amount 2000.00 and description "Salary"
    When I create the movement
    Then the operation should be successful
    When I update the movement in month 2 with amount 2500.00
    Then the operation should be successful
    When I search for movements in month 0
    Then I should find the movement with amount 2000.00
    When I search for movements in month 2
    Then I should find the movement with amount 2500.00
    When I search for movements in month 3
    Then I should find the movement with amount 2000.00

  Scenario: Update all next occurrences of recurrent movement
    Given I have a recurrent movement with amount 2000.00 and description "Salary"
    When I create the movement
    Then the operation should be successful
    When I update all next occurrences from month 2 with amount 2500.00
    Then the operation should be successful
    When I search for movements in month 0
    Then I should find the movement with amount 2000.00
    When I search for movements in month 2
    Then I should find the movement with amount 2500.00
    When I search for movements in month 3
    Then I should find the movement with amount 2500.00
    When I search for movements in month 4
    Then I should find the movement with amount 2500.00