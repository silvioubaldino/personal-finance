@movement @delete-all-next
Feature: Delete a standalone movement and all next
  A standalone movement has no "next" occurrences, so deleting it and all next behaves
  exactly like deleting only it.

  Background:
    Given today is "2026-03-10"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Home"

  Scenario: Deleting a pending standalone movement and all next leaves the wallet balance alone
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I delete "Rent" and all next
    Then the operation succeeds
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |
    And the balance of wallet "Checking" is 1000.00

  @paid
  Scenario: Deleting a paid standalone movement and all next refunds the wallet
    Given a paid expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I delete "Rent" and all next
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00

  Scenario: Other standalone movements are not touched
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    And a pending expense "Water" of 80.00 on "2026-04-05" in wallet "Checking" under category "Home"
    When I delete "Rent" and all next
    Then the movements of wallet "Checking" in "2026-04" are:
      | description | amount | paid |
      | Water       | -80.00 | no   |
