@import @statement @recurrence
Feature: Statement lines linked to a recurrent movement
  A line linked to a recurrent movement becomes that month's occurrence instead of a new
  movement.

  Background:
    Given today is "2026-09-01"
    And a wallet "Checking" with balance 3000.00
    And an income category "Salary"
    And an expense category "Utilities"
    And an expense category "Food"
    And a pending monthly recurrent expense "Electricity" of 180.00 starting "2026-06-08" in wallet "Checking" under category "Utilities"

  Scenario: A line linked to a recurrence becomes the occurrence of the month
    When I extract a statement for wallet "Checking" read as:
      | date       | description       | amount  |
      | 2026-08-08 | CONTA DE LUZ ENEL | -187.42 |
    And I link the line "CONTA DE LUZ ENEL" to the recurrent expense "Electricity"
    And I confirm the import
    Then the import created 1 and skipped 0
    And the movements of wallet "Checking" in "2026-08" are:
      | description       | amount  | paid |
      | CONTA DE LUZ ENEL | -187.42 | yes  |
    And the balance of wallet "Checking" is 2812.58

  Scenario: Reimporting the statement without linking again does not duplicate the occurrence
    Given I extract a statement for wallet "Checking" read as:
      | date       | description       | amount  |
      | 2026-08-08 | CONTA DE LUZ ENEL | -187.42 |
    And I link the line "CONTA DE LUZ ENEL" to the recurrent expense "Electricity"
    And I confirm the import
    When I import a statement into wallet "Checking" read as:
      | date       | description       | amount  |
      | 2026-08-08 | CONTA DE LUZ ENEL | -187.42 |
    Then the import created 0 and skipped 1
    And the balance of wallet "Checking" is 2812.58
