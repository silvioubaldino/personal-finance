@movement @update-one
Feature: Update a single movement
  Updating one movement changes only that movement; when it is paid the wallet follows.

  Background:
    Given today is "2026-03-10"
    And a wallet "Checking" with balance 1000.00
    And a wallet "Savings" with balance 500.00
    And an expense category "Home"
    And an expense category "Leisure"
    And a subcategory "Cinema" of "Leisure"

  Scenario: Updating a pending movement leaves the wallet balance alone
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update only "Rent" with:
      | amount      | -450.00      |
      | description | Rent (March) |
    Then the operation succeeds
    And the movements of wallet "Checking" in "2026-03" are:
      | description  | amount  | paid |
      | Rent (March) | -450.00 | no   |
    And the balance of wallet "Checking" is 1000.00

  @paid
  Scenario: Updating the amount of a paid movement adjusts the wallet by the difference
    Given a paid expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update only "Rent" setting amount to -450.00
    Then the operation succeeds
    And "Rent" is paid
    And the balance of wallet "Checking" is 550.00

  @paid
  Scenario: Moving a paid movement to another wallet refunds the old wallet and debits the new one
    Given a paid expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update only "Rent" with:
      | wallet | Savings |
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And the balance of wallet "Savings" is 100.00

  @paid
  Scenario: An update that would overdraw the wallet is rejected and changes nothing
    Given a paid expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update only "Rent" setting amount to -1500.00
    Then the operation is rejected as "insufficient balance"
    And the balance of wallet "Checking" is 600.00
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount  | paid |
      | Rent        | -400.00 | yes  |

  Scenario: Changing the date moves the movement to another month
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update only "Rent" with:
      | date | 2026-04-05 |
    Then the operation succeeds
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |
    And the movements of wallet "Checking" in "2026-04" are:
      | description | amount  | paid |
      | Rent        | -400.00 | no   |

  Scenario: A subcategory that does not belong to the category is rejected
    Given a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    When I update only "Rent" with:
      | subcategory | Cinema |
    Then the operation is rejected as "invalid input"
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount  | paid |
      | Rent        | -400.00 | no   |

  Scenario: A subcategory of the category is accepted
    Given a pending expense "Movie" of 40.00 on "2026-03-05" in wallet "Checking" under category "Leisure"
    When I update only "Movie" with:
      | subcategory | Cinema |
    Then the operation succeeds
