@import @statement @idempotency
Feature: Reimporting a statement
  Importing the same lines again — the same file or an overlapping one — never duplicates them.

  Background:
    Given today is "2026-09-01"
    And a wallet "Checking" with balance 3000.00
    And an income category "Salary"
    And an expense category "Utilities"
    And an expense category "Food"

  Scenario: Reimporting the same statement creates nothing
    Given I import a statement into wallet "Checking" read as:
      | date       | description                    | amount  |
      | 2026-08-01 | PIX RECEBIDO SALARIO ACME LTDA | 6500.00 |
      | 2026-08-12 | COMPRA DEBITO PADARIA CENTRAL  | -23.80  |
    When I import a statement into wallet "Checking" read as:
      | date       | description                    | amount  |
      | 2026-08-01 | PIX RECEBIDO SALARIO ACME LTDA | 6500.00 |
      | 2026-08-12 | COMPRA DEBITO PADARIA CENTRAL  | -23.80  |
    Then the import created 0 and skipped 2
    And the balance of wallet "Checking" is 9476.20

  Scenario: Overlapping statements do not duplicate the shared lines
    Given I import the statement "extrato-2026-08" into wallet "Checking"
    When I import the statement "extrato-2026-08-segunda-quinzena" into wallet "Checking"
    Then the import created 2 and skipped 7
    And every item of the document is in the app
