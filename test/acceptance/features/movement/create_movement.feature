@movement
Feature: Create a movement
  A pending movement is only planned; a paid one already moved the wallet.

  Background:
    Given today is "2026-03-10"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Home"
    And an income category "Salary"

  Scenario: A pending expense does not change the wallet balance
    When I add a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    Then the operation succeeds
    And "Rent" is pending
    And the balance of wallet "Checking" is 1000.00

  Scenario: A paid expense debits the wallet
    When I add a paid expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    Then the operation succeeds
    And "Rent" is paid
    And the balance of wallet "Checking" is 600.00

  Scenario: A paid income credits the wallet
    When I add a paid income "Bonus" of 250.00 on "2026-03-05" in wallet "Checking" under category "Salary"
    Then the operation succeeds
    And the balance of wallet "Checking" is 1250.00

  Scenario: A paid expense above the wallet balance is rejected
    When I add a paid expense "Car" of 1500.00 on "2026-03-05" in wallet "Checking" under category "Home"
    Then the operation is rejected as "insufficient balance"
    And the balance of wallet "Checking" is 1000.00
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |

  Scenario: A created movement is listed in its month only
    When I add a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"
    Then the movements of wallet "Checking" in "2026-03" are:
      | description | amount  | paid |
      | Rent        | -400.00 | no   |
    And the movements of wallet "Checking" in "2026-04" are:
      | description | amount | paid |
