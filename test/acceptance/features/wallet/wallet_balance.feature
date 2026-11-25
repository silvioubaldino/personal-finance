@wallet
Feature: Wallet balance
  A wallet starts with the balance it was created with.

  Background:
    Given today is "2026-01-15"

  Scenario: A new wallet starts with its initial balance
    Given a wallet "Checking" with balance 1000.00
    Then the operation succeeds
    And the balance of wallet "Checking" is 1000.00

  Scenario: Creating a category does not change a wallet balance
    Given a wallet "Checking" with balance 250.50
    And an expense category "Home"
    Then the balance of wallet "Checking" is 250.50
