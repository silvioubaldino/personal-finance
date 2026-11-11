@recurrence @update-all-next
Feature: Update a recurrent movement from an occurrence onwards
  Updating an occurrence "and all next" rewrites the series from that month on,
  leaving the earlier months untouched.

  Background:
    Given today is "2026-01-15"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Sports"
    And a pending monthly recurrent expense "Gym" of 100.00 starting "2026-01-10" in wallet "Checking" under category "Sports"

  @virtual-occurrence
  Scenario: Updating a future occurrence and all next
    When I update the occurrence of "Gym" in "2026-03" and all next with:
      | amount | -120.00 |
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -120.00 | no   |
      | 2026-12 | -120.00 | no   |
    And the balance of wallet "Checking" is 1000.00

  Scenario: Updating the first occurrence and all next rewrites the whole series
    When I update the occurrence of "Gym" in "2026-01" and all next with:
      | amount      | -120.00     |
      | description | Gym premium |
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | description | paid |
      | 2026-01 | -120.00 | Gym premium | no   |
      | 2026-06 | -120.00 | Gym premium | no   |

  @paid
  Scenario: Updating a paid occurrence and all next adjusts the wallet by the difference
    Given I paid the occurrence of "Gym" in "2026-01"
    When I update the occurrence of "Gym" in "2026-01" and all next with:
      | amount | -120.00 |
    Then the operation succeeds
    And the balance of wallet "Checking" is 880.00
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -120.00 | yes  |
      | 2026-02 | -120.00 | no   |

  Scenario: Updating all next in the middle keeps the earlier months, even when edited before
    Given I update only the occurrence of "Gym" in "2026-02" setting amount to -90.00
    When I update the occurrence of "Gym" in "2026-04" and all next with:
      | amount | -120.00 |
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |
      | 2026-02 | -90.00  | no   |
      | 2026-03 | -100.00 | no   |
      | 2026-04 | -120.00 | no   |
      | 2026-09 | -120.00 | no   |

  Scenario: Updating all next from a month moves the series to another wallet
    Given a wallet "Savings" with balance 500.00
    When I update the occurrence of "Gym" in "2026-03" and all next with:
      | wallet | Savings |
    Then the operation succeeds
    And the movements of wallet "Savings" in "2026-03" are:
      | description | amount  | paid |
      | Gym         | -100.00 | no   |
    And the movements of wallet "Checking" in "2026-02" are:
      | description | amount  | paid |
      | Gym         | -100.00 | no   |

  @paid
  Scenario: Updating all next keeps the paid months before it as they were
    Given I paid the occurrence of "Gym" in "2026-01"
    When I update the occurrence of "Gym" in "2026-03" and all next with:
      | amount | -120.00 |
    Then the operation succeeds
    And the balance of wallet "Checking" is 900.00
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | yes  |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -120.00 | no   |

  @paid @known-bug
  Scenario: Updating all next from a month before an occurrence that was already paid does not duplicate it
    Given I paid the occurrence of "Gym" in "2026-03"
    When I update the occurrence of "Gym" in "2026-02" and all next with:
      | amount | -120.00 |
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -120.00 | no   |
      | 2026-03 | -100.00 | yes  |
      | 2026-04 | -120.00 | no   |

  @paid @known-bug
  Scenario: Updating the first occurrence and all next keeps a later occurrence that was already paid
    Given I paid the occurrence of "Gym" in "2026-03"
    When I update the occurrence of "Gym" in "2026-01" and all next with:
      | amount | -120.00 |
    Then the operation succeeds
    And the balance of wallet "Checking" is 900.00
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -120.00 | no   |
      | 2026-02 | -120.00 | no   |
      | 2026-03 | -100.00 | yes  |
      | 2026-04 | -120.00 | no   |
