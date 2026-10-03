@recurrence @delete-all-next
Feature: Delete a recurrent movement from an occurrence onwards
  Deleting "and all next" ends the series just before that month.

  Background:
    Given today is "2026-01-15"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Sports"
    And a pending monthly recurrent expense "Gym" of 100.00 starting "2026-01-10" in wallet "Checking" under category "Sports"

  @virtual-occurrence
  Scenario: Deleting a future occurrence and all next ends the series before it
    When I delete the occurrence of "Gym" in "2026-03" and all next
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |
      | 2026-02 | -100.00 | no   |
    And there is no occurrence of "Gym" in "2026-03"
    And there is no occurrence of "Gym" in "2026-04"
    And there is no occurrence of "Gym" in "2027-01"

  Scenario: Deleting the first occurrence and all next removes the whole series
    When I delete the occurrence of "Gym" in "2026-01" and all next
    Then the operation succeeds
    And there is no occurrence of "Gym" in "2026-01"
    And there is no occurrence of "Gym" in "2026-02"
    And there is no occurrence of "Gym" in "2026-12"

  @paid
  Scenario: Deleting a paid occurrence and all next refunds the wallet
    Given I paid the occurrence of "Gym" in "2026-02"
    When I delete the occurrence of "Gym" in "2026-02" and all next
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And there is no occurrence of "Gym" in "2026-02"
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |

  @paid @known-bug
  Scenario: Deleting all next also removes a later occurrence that was already paid
    Given I paid the occurrence of "Gym" in "2026-04"
    When I delete the occurrence of "Gym" in "2026-02" and all next
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And there is no occurrence of "Gym" in "2026-04"

  Scenario: Deleting all next keeps the earlier months, even when edited before
    Given I update only the occurrence of "Gym" in "2026-02" setting amount to -90.00
    When I delete the occurrence of "Gym" in "2026-04" and all next
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |
      | 2026-02 | -90.00  | no   |
      | 2026-03 | -100.00 | no   |
    And there is no occurrence of "Gym" in "2026-04"

  @paid @known-bug
  Scenario: Deleting the first occurrence and all next refunds a later occurrence that was already paid
    Given I paid the occurrence of "Gym" in "2026-03"
    When I delete the occurrence of "Gym" in "2026-01" and all next
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And there is no occurrence of "Gym" in "2026-03"
