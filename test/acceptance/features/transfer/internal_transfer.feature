@transfer
Feature: Internal transfer between wallets
  A transfer is two linked movements, one out and one in; it is created, paid, reverted,
  updated and deleted as a pair, never one leg at a time.

  Background:
    Given today is "2026-03-10"
    And a wallet "Checking" with balance 1000.00
    And a wallet "Savings" with balance 500.00

  Scenario: A paid transfer moves the money between the wallets
    When I add a paid internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    Then the operation succeeds
    And the balance of wallet "Checking" is 700.00
    And the balance of wallet "Savings" is 800.00

  Scenario: A pending transfer moves nothing until it is paid
    When I add a pending internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    Then the balance of wallet "Checking" is 1000.00
    And the balance of wallet "Savings" is 500.00

  Scenario: Paying a pending transfer moves the money
    Given a pending internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    When I pay the internal transfer "Reserve"
    Then the operation succeeds
    And the balance of wallet "Checking" is 700.00
    And the balance of wallet "Savings" is 800.00

  Scenario: Reverting the payment of a transfer gives the money back
    Given a paid internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    When I revert the payment of the internal transfer "Reserve"
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And the balance of wallet "Savings" is 500.00

  Scenario: Deleting a paid transfer gives the money back and removes both legs
    Given a paid internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    When I delete the internal transfer "Reserve"
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And the balance of wallet "Savings" is 500.00
    And the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |
    And the movements of wallet "Savings" in "2026-03" are:
      | description | amount | paid |

  Scenario: Deleting a pending transfer leaves the balances alone
    Given a pending internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    When I delete the internal transfer "Reserve"
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00
    And the balance of wallet "Savings" is 500.00

  Scenario: Updating the amount of a paid transfer adjusts both wallets
    Given a paid internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    When I update the internal transfer "Reserve" with:
      | amount | 400.00 |
    Then the operation succeeds
    And the balance of wallet "Checking" is 600.00
    And the balance of wallet "Savings" is 900.00

  Scenario: A transfer to the same wallet is rejected
    When I add a paid internal transfer "Loop" of 100.00 from "Checking" to "Checking" on "2026-03-10"
    Then the operation is rejected as "invalid input"
    And the balance of wallet "Checking" is 1000.00

  Scenario: A paid transfer above the origin balance is rejected
    When I add a paid internal transfer "Big" of 5000.00 from "Checking" to "Savings" on "2026-03-10"
    Then the operation is rejected as "insufficient balance"
    And the balance of wallet "Checking" is 1000.00
    And the balance of wallet "Savings" is 500.00

  Scenario Outline: A leg of a transfer cannot be changed through the movement routes
    Given a paid internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    When I <action>
    Then the operation is rejected as "invalid input"
    And the balance of wallet "Checking" is 700.00
    And the balance of wallet "Savings" is 800.00

    Examples:
      | action                                                  |
      | update only the outgoing leg of "Reserve" setting amount to -50.00 |
      | delete only the outgoing leg of "Reserve"               |
      | pay the outgoing leg of "Reserve"                       |
      | revert the payment of the outgoing leg of "Reserve"     |

  @known-bug
  Scenario Outline: A leg of a transfer cannot be changed through the "all next" movement routes
    Given a paid internal transfer "Reserve" of 300.00 from "Checking" to "Savings" on "2026-03-10"
    When I <action>
    Then the operation is rejected as "invalid input"
    And the balance of wallet "Checking" is 700.00
    And the balance of wallet "Savings" is 800.00

    Examples:
      | action                                                              |
      | update the outgoing leg of "Reserve" and all next setting amount to -50.00 |
      | delete the outgoing leg of "Reserve" and all next                   |
