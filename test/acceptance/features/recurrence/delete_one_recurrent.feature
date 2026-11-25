@recurrence @delete-one
Feature: Delete one occurrence of a recurrent movement
  Deleting one occurrence leaves a gap in that month; the series goes on.

  Background:
    Given today is "2026-01-15"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Sports"
    And a pending monthly recurrent expense "Gym" of 100.00 starting "2026-01-10" in wallet "Checking" under category "Sports"

  @virtual-occurrence
  Scenario: Deleting a future occurrence leaves a gap in that month only
    When I delete only the occurrence of "Gym" in "2026-03"
    Then the operation succeeds
    And there is no occurrence of "Gym" in "2026-03"
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |
      | 2026-02 | -100.00 | no   |
      | 2026-04 | -100.00 | no   |
      | 2026-12 | -100.00 | no   |

  Scenario: Deleting the first occurrence removes that month and keeps the series going
    When I delete only the occurrence of "Gym" in "2026-01"
    Then the operation succeeds
    And there is no occurrence of "Gym" in "2026-01"
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-06 | -100.00 | no   |

  Scenario: Deleting two different occurrences leaves two gaps
    Given I delete only the occurrence of "Gym" in "2026-03"
    When I delete only the occurrence of "Gym" in "2026-05"
    Then the operation succeeds
    And there is no occurrence of "Gym" in "2026-03"
    And there is no occurrence of "Gym" in "2026-05"
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-04 | -100.00 | no   |
      | 2026-06 | -100.00 | no   |

  @paid
  Scenario: Deleting a paid occurrence refunds the wallet and keeps the series going
    Given I paid the occurrence of "Gym" in "2026-03"
    When I delete only the occurrence of "Gym" in "2026-03"
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And there is no occurrence of "Gym" in "2026-03"
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-04 | -100.00 | no   |

  @paid @known-bug
  Scenario: Deleting the first occurrence keeps an occurrence that was edited in a later month
    Given I update only the occurrence of "Gym" in "2026-03" setting amount to -130.00
    When I delete only the occurrence of "Gym" in "2026-01"
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -130.00 | no   |
      | 2026-04 | -100.00 | no   |

  @paid @known-bug
  Scenario: Deleting the first occurrence keeps a later occurrence that was already paid
    Given I paid the occurrence of "Gym" in "2026-03"
    When I delete only the occurrence of "Gym" in "2026-01"
    Then the operation succeeds
    And the balance of wallet "Checking" is 900.00
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -100.00 | yes  |
