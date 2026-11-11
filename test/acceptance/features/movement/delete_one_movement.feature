@movement @delete-one
Feature: Delete a single movement
  Deleting a movement removes it; when it was paid the wallet gets the money back.

  Background:
    Given today is "2026-03-10"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Home"

  Scenario: Deleting a pending movement leaves the wallet balance alone
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I delete only "Rent"
    Then the operation succeeds
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |
    And the balance of wallet "Checking" is 1000.00

  @paid
  Scenario: Deleting a paid movement refunds the wallet
    Given a paid expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I delete only "Rent"
    Then the operation succeeds
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |
    And the balance of wallet "Checking" is 1000.00

  Scenario: Deleting one movement keeps the others
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    And a pending expense "Water" of 80.00 on "2026-03-07" in wallet "Checking" under category "Home"
    When I delete only "Rent"
    Then the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |
      | Water       | -80.00 | no   |
