@recurrence
Feature: Recurrent movements
  A monthly recurrent movement is materialized only for its first month; the following
  months are projected from the series until someone pays, edits or deletes them.

  Background:
    Given today is "2026-01-15"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Sports"
    And a pending monthly recurrent expense "Gym" of 100.00 starting "2026-01-10" in wallet "Checking" under category "Sports"

  Scenario: The series shows up in every month from its start
    Then the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |
      | 2026-02 | -100.00 | no   |
      | 2026-07 | -100.00 | no   |
      | 2027-03 | -100.00 | no   |
    And there is no occurrence of "Gym" in "2025-12"
    And the balance of wallet "Checking" is 1000.00

  @virtual-occurrence @paid
  Scenario: Paying a future occurrence pays only that month
    When I pay the occurrence of "Gym" in "2026-03"
    Then the operation succeeds
    And the balance of wallet "Checking" is 900.00
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -100.00 | yes  |
      | 2026-04 | -100.00 | no   |

  @paid
  Scenario: A paid occurrence is listed once
    Given I paid the occurrence of "Gym" in "2026-03"
    Then the movements of wallet "Checking" in "2026-03" are:
      | description | amount  | paid |
      | Gym         | -100.00 | yes  |

  @paid
  Scenario: Reverting the payment of an occurrence gives the money back
    Given I paid the occurrence of "Gym" in "2026-03"
    When I revert the payment of the occurrence of "Gym" in "2026-03"
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-03 | -100.00 | no   |

  Scenario: A recurrent income is projected with a positive amount
    Given an income category "Salary"
    And a pending monthly recurrent income "Pay" of 3000.00 starting "2026-01-05" in wallet "Checking" under category "Salary"
    Then the occurrences of "Pay" are:
      | month   | amount  | paid |
      | 2026-01 | 3000.00 | no   |
      | 2026-05 | 3000.00 | no   |
