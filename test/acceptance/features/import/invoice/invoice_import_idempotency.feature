@import @invoice @idempotency
Feature: Reimporting an invoice
  Importing the same invoice again never duplicates anything.

  Background:
    Given today is "2026-08-05"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Inter" with limit 10000.00, closing day 3 and due day 10, paid from wallet "Checking"

  Scenario: Reimporting the same invoice creates nothing
    Given I import an invoice into card "Inter" read as:
      | date       | description                           | amount  | installment |
      | 2026-07-12 | MERCADO PIRATININGA                   | -187.36 |             |
      | 2026-07-09 | SHOPEE *UTILIDADES (Parcela 01 de 03) | -83.65  | 1/3         |
    When I import an invoice into card "Inter" read as:
      | date       | description                           | amount  | installment |
      | 2026-07-12 | MERCADO PIRATININGA                   | -187.36 |             |
      | 2026-07-09 | SHOPEE *UTILIDADES (Parcela 01 de 03) | -83.65  | 1/3         |
    Then the import created 0 and skipped 4
    And the invoice of card "Inter" due in "2026-08" has amount -271.01
    And the available limit of card "Inter" is 9561.69

  Scenario: Reimporting after a missed line creates only that line
    Given I import an invoice into card "Inter" read as:
      | date       | description         | amount  |
      | 2026-07-12 | MERCADO PIRATININGA | -187.36 |
    When I import an invoice into card "Inter" read as:
      | date       | description         | amount  |
      | 2026-07-12 | MERCADO PIRATININGA | -187.36 |
      | 2026-07-15 | NETFLIX ENTRETENIMENTO | -59.90 |
    Then the import created 1 and skipped 1
    And the invoice of card "Inter" due in "2026-08" has amount -247.26

  # R5 (SPEC-009, decisão D5): a IA real varia a pontuação entre leituras.
  @known-bug
  Scenario: Reimporting with a reading that differs only in punctuation creates nothing
    Given I import an invoice into card "Inter" read as:
      | date       | description   | amount |
      | 2026-07-19 | MP *CONECTCAR | -4.37  |
    When I import an invoice into card "Inter" read as:
      | date       | description  | amount |
      | 2026-07-19 | MP*CONECTCAR | -4.37  |
    Then the import created 0 and skipped 1
    And the invoice of card "Inter" due in "2026-08" has amount -4.37

  # R2 (SPEC-009, decisão D2).
  @known-bug
  Scenario: Two identical items on the same day are both imported
    When I import an invoice into card "Inter" read as:
      | date       | description   | amount |
      | 2026-07-19 | MP *CONECTCAR | -4.37  |
      | 2026-07-19 | MP *CONECTCAR | -4.37  |
    Then the import created 2 and skipped 0
    And every item of the document is in the app
    And the invoice of card "Inter" due in "2026-08" has amount -8.74

  Scenario: Reimporting two identical items does not duplicate them
    Given I import an invoice into card "Inter" read as:
      | date       | description   | amount |
      | 2026-07-19 | MP *CONECTCAR | -4.37  |
      | 2026-07-19 | MP *CONECTCAR | -4.37  |
    When I import an invoice into card "Inter" read as:
      | date       | description   | amount |
      | 2026-07-19 | MP *CONECTCAR | -4.37  |
      | 2026-07-19 | MP *CONECTCAR | -4.37  |
    Then the import created 0 and skipped 2
