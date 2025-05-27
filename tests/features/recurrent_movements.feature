Feature: Recurrent Movements Management
  As a system user
  I want to manage recurrent movements
  So that I can create, edit and delete recurring financial transactions

#  Scenario: Create recurrent movement and validate future occurrences
#    Given I have a recurrent movement with amount 2000.00 and description "Salary"
#    When I create the movement
#    Then the operation should be successful
#    When I search for movements in month 0
#    Then I should find the movement with amount 2000.00
#    When I search for movements in month 2
#    Then I should find the movement with amount 2000.00

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
#
#  Scenario: Update all next occurrences of recurrent movement
#    Given I have a recurrent movement with amount 2000.00 and description "Salary"
#    When I create the movement
#    Then the operation should be successful
#    When I update all next occurrences from month 2 with amount 2500.00
#    Then the operation should be successful
#    When I search for movements in month 0
#    Then I should find the movement with amount 2000.00
#    When I search for movements in month 2
#    Then I should find the movement with amount 2500.00
#    When I search for movements in month 3
#    Then I should find the movement with amount 2500.00
#    When I search for movements in month 4
#    Then I should find the movement with amount 2500.00
#
#  Scenario: Delete one occurrence of recurrent movement
#    Given I have a recurrent movement with amount 1500.00 and description "Rent"
#    When I create the movement
#    Then the operation should be successful
#    When I delete the movement in month 2
#    Then the operation should be successful
#    When I search for movements in month 0
#    Then I should find the movement with amount 1500.00
#    When I search for movements in month 2
#    Then I should not find any movement
#    When I search for movements in month 3
#    Then I should find the movement with amount 1500.00
#
#  Scenario: Delete all future occurrences of recurrent movement
#    Given I have a recurrent movement with amount 1500.00 and description "Rent"
#    When I create the movement
#    Then the operation should be successful
#    When I delete all future occurrences from month 2
#    Then the operation should be successful
#    When I search for movements in month 0
#    Then I should find the movement with amount 1500.00
#    When I search for movements in month 2
#    Then I should not find any movement
#    When I search for movements in month 3
#    Then I should not find any movement