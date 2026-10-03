@movement @update-all-next
Feature: Update a standalone movement and all next
  A standalone movement has no "next" occurrences, so updating it and all next behaves
  exactly like updating only it.

  Background:
    Given today is "2026-03-10"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Home"

  Scenario: Updating a pending standalone movement and all next leaves the wallet balance alone
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update "Rent" and all next setting amount to -450.00
    Then the operation succeeds
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount  | paid |
      | Rent        | -450.00 | no   |
    And the balance of wallet "Checking" is 1000.00

  @paid
  Scenario: Updating a paid standalone movement and all next adjusts the wallet by the difference
    Given a paid expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update "Rent" and all next setting amount to -450.00
    Then the operation succeeds
    And the balance of wallet "Checking" is 550.00
    And "Rent" is paid

  Scenario: Other standalone movements are not touched
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    And a pending expense "Water" of 80.00 on "2026-04-05" in wallet "Checking" under category "Home"
    When I update "Rent" and all next setting amount to -450.00
    Then the movements of wallet "Checking" in "2026-04" are:
      | description | amount | paid |
      | Water       | -80.00 | no   |
