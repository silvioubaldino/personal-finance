@import @statement
Feature: Statement import
  Importing a bank statement registers every line as a paid movement of the wallet, whatever
  the AI understood of its category or name.

  Background:
    Given today is "2026-09-01"
    And a wallet "Checking" with balance 3000.00
    And an income category "Salary"
    And an expense category "Utilities"
    And an expense category "Food"

  Scenario: Every line enters the wallet as paid and moves its balance
    When I import a statement into wallet "Checking" read as:
      | date       | description                          | amount   |
      | 2026-08-01 | PIX RECEBIDO SALARIO ACME LTDA       | 6500.00  |
      | 2026-08-02 | PAGAMENTO BOLETO ALUGUEL IMOBILIARIA | -2100.00 |
      | 2026-08-12 | COMPRA DEBITO PADARIA CENTRAL        | -23.80   |
    Then the import created 3 and skipped 0
    And every item of the document is in the app
    And the balance of wallet "Checking" is 7376.20

  Scenario: Lines the AI could not categorize go to the fallback by sign
    When I import a statement into wallet "Checking" read as:
      | date       | description                   | amount |
      | 2026-08-15 | PIX RECEBIDO REEMBOLSO MARIA  | 120.00 |
      | 2026-08-12 | COMPRA DEBITO PADARIA CENTRAL | -23.80 |
    Then "PIX RECEBIDO REEMBOLSO MARIA" is uncategorized income
    And "COMPRA DEBITO PADARIA CENTRAL" is uncategorized

  Scenario: A description categorized in the past keeps its category
    Given a paid expense "COMPRA DEBITO PADARIA CENTRAL" of 10.00 on "2026-07-01" in wallet "Checking" under category "Food"
    When I import a statement into wallet "Checking" read as:
      | date       | description                   | amount |
      | 2026-08-12 | COMPRA DEBITO PADARIA CENTRAL | -23.80 |
    Then "COMPRA DEBITO PADARIA CENTRAL" is in category "Food"

  # R2 (SPEC-009, decisão D2): o extrato tem dois PIX idênticos no mesmo dia.
  @known-bug
  Scenario: The whole document enters the wallet
    When I import the statement "extrato-2026-08" into wallet "Checking"
    Then every item of the document is in the app
    And the balance of wallet "Checking" is 4247.98

  # R2 (SPEC-009, decisão D2).
  @known-bug
  Scenario: Two identical lines on the same day are both imported
    When I import a statement into wallet "Checking" read as:
      | date       | description               | amount |
      | 2026-08-05 | PIX ENVIADO JOAO FICTICIO | -50.00 |
      | 2026-08-05 | PIX ENVIADO JOAO FICTICIO | -50.00 |
    Then the import created 2 and skipped 0
    And the balance of wallet "Checking" is 2900.00
