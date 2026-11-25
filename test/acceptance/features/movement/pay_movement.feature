@movement @paid
Feature: Pay and revert the payment of a movement
  Paying moves the money in the wallet; reverting gives it back.

  Background:
    Given today is "2026-03-10"
    And a wallet "Checking" with balance 1000.00
    And an expense category "Home"
    And a pending expense "Rent" of 400.00 on "2026-03-05" in wallet "Checking" under category "Home"

  Scenario: Paying a pending expense debits the wallet
    When I pay "Rent"
    Then the operation succeeds
    And "Rent" is paid
    And the balance of wallet "Checking" is 600.00

  Scenario: Paying a movement that is already paid is rejected
    Given I paid "Rent"
    When I pay "Rent"
    Then the operation is rejected as "conflict"
    And the balance of wallet "Checking" is 600.00

  Scenario: Reverting a payment gives the money back
    Given I paid "Rent"
    When I revert the payment of "Rent"
    Then the operation succeeds
    And "Rent" is pending
    And the balance of wallet "Checking" is 1000.00

  Scenario: Reverting a movement that is pending is rejected
    When I revert the payment of "Rent"
    Then the operation is rejected as "conflict"
    And the balance of wallet "Checking" is 1000.00

  Scenario: Paying an expense above the wallet balance is rejected
    Given an expense category "Car"
    And a pending expense "Car" of 1500.00 on "2026-03-06" in wallet "Checking" under category "Car"
    When I pay "Car"
    Then the operation is rejected as "insufficient balance"
    And "Car" is pending
    And the balance of wallet "Checking" is 1000.00
