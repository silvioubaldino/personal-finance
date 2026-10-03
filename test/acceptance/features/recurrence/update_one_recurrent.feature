@recurrence @update-one
Feature: Update one occurrence of a recurrent movement
  Updating only one occurrence changes that month and leaves the rest of the series alone.

  Background:
    Given today is "2026-01-15"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Sports"
    And a pending monthly recurrent expense "Gym" of 100.00 starting "2026-01-10" in wallet "Checking" under category "Sports"

  @virtual-occurrence
  Scenario: Updating a future occurrence changes only that month
    When I update only the occurrence of "Gym" in "2026-03" setting amount to -130.00
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -100.00 | no   |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -130.00 | no   |
      | 2026-04 | -100.00 | no   |
      | 2026-12 | -100.00 | no   |
    And the balance of wallet "Checking" is 1000.00

  Scenario: Updating the first occurrence changes only that month
    When I update only the occurrence of "Gym" in "2026-01" setting amount to -130.00
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-01 | -130.00 | no   |
      | 2026-02 | -100.00 | no   |
      | 2026-06 | -100.00 | no   |

  Scenario: Updating the description of one occurrence does not rename the others
    When I update only the occurrence of "Gym" in "2026-03" with:
      | description | Gym (personal trainer) |
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | description            | paid |
      | 2026-02 | Gym                    | no   |
      | 2026-03 | Gym (personal trainer) | no   |
      | 2026-04 | Gym                    | no   |

  Scenario: Updating the same occurrence twice keeps a single occurrence per month
    Given I update only the occurrence of "Gym" in "2026-03" setting amount to -130.00
    When I update only the occurrence of "Gym" in "2026-03" setting amount to -140.00
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-03 | -140.00 | no   |
      | 2026-04 | -100.00 | no   |
      | 2026-05 | -100.00 | no   |

  Scenario: Updating two different occurrences keeps the months in between
    Given I update only the occurrence of "Gym" in "2026-03" setting amount to -130.00
    When I update only the occurrence of "Gym" in "2026-05" setting amount to -150.00
    Then the operation succeeds
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -130.00 | no   |
      | 2026-04 | -100.00 | no   |
      | 2026-05 | -150.00 | no   |
      | 2026-06 | -100.00 | no   |

  @paid
  Scenario: Updating a paid occurrence adjusts the wallet by the difference
    Given I paid the occurrence of "Gym" in "2026-03"
    When I update only the occurrence of "Gym" in "2026-03" setting amount to -130.00
    Then the operation succeeds
    And the balance of wallet "Checking" is 870.00
    And the occurrences of "Gym" are:
      | month   | amount  | paid |
      | 2026-02 | -100.00 | no   |
      | 2026-03 | -130.00 | yes  |
      | 2026-04 | -100.00 | no   |
