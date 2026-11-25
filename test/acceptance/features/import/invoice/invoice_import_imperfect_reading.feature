@import @invoice @imperfect-reading
Feature: Imperfect readings of the AI
  Whatever the AI gets wrong, the app never loses a value nor doubles one.

  Background:
    Given today is "2026-08-05"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Inter" with limit 10000.00, closing day 3 and due day 10, paid from wallet "Checking"

  Scenario: An item with an unreadable date is skipped and the others are imported
    When I import an invoice into card "Inter" read as:
      | date       | description         | amount  |
      | 2026-07-12 | MERCADO PIRATININGA | -187.36 |
      | 2026-13-45 | UBER *TRIP HELP.U   | -18.90  |
      | 2026-07-15 | NETFLIX ENTRETENIMENTO | -59.90 |
    Then the import created 2 and skipped 1

  Scenario: A declared total with the opposite sign is still matched
    When I import an invoice into card "Inter" declaring total 247.26 read as:
      | date       | description            | amount  |
      | 2026-07-12 | MERCADO PIRATININGA    | -187.36 |
      | 2026-07-15 | NETFLIX ENTRETENIMENTO | -59.90  |
    Then the invoice of card "Inter" due in "2026-08" matches the document total

  # R6 (SPEC-009, questão Q2 em aberto).
  @wip
  Scenario: An installment read without its installment fields does not double the series
    Given I import an invoice into card "Inter" read as:
      | date       | description                           | amount | installment |
      | 2026-07-09 | SHOPEE *UTILIDADES (Parcela 01 de 03) | -83.65 | 1/3         |
    And it is now "2026-09-05"
    When I import an invoice into card "Inter" read as:
      | date       | description                           | amount |
      | 2026-07-09 | SHOPEE *UTILIDADES (Parcela 02 de 03) | -83.65 |
      | 2026-08-05 | UBER *TRIP HELP.U                     | -22.40 |
    Then the installments of "SHOPEE *UTILIDADES" on card "Inter" are:
      | installment | due     | amount |
      | 1           | 2026-08 | -83.65 |
      | 2           | 2026-09 | -83.65 |
      | 3           | 2026-10 | -83.65 |
    And the invoice of card "Inter" due in "2026-08" has amount -83.65
    And the invoice of card "Inter" due in "2026-09" has amount -106.05
