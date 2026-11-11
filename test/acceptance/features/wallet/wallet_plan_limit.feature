@wallet
Feature: Wallet limit of the free plan
  Free users can have a limited number of wallets; plus users have no limit.

  Background:
    Given today is "2026-01-15"

  Scenario: A free user cannot exceed the wallet limit
    Given I am on the free plan
    And a wallet "First" with balance 0.00
    And a wallet "Second" with balance 0.00
    When I add a wallet "Third" with balance 0.00
    Then the operation is rejected as "forbidden"

  Scenario: A plus user has no wallet limit
    Given a wallet "First" with balance 0.00
    And a wallet "Second" with balance 0.00
    When I add a wallet "Third" with balance 0.00
    Then the operation succeeds
