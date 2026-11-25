@import @invoice
Feature: Invoice import
  Importing a credit card invoice registers every item that belongs to it in the card's
  invoice, whatever the AI understood of its category or name.

  Background:
    Given today is "2026-08-05"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Transport"
    And a credit card "Inter" with limit 10000.00, closing day 3 and due day 10, paid from wallet "Checking"

  Scenario: Every item enters the invoice, which closes with the document total
    When I import an invoice into card "Inter" declaring total -323.15 read as:
      | date       | description            | amount  |
      | 2026-07-05 | UBER *TRIP HELP.U      | -18.90  |
      | 2026-07-12 | MERCADO PIRATININGA    | -187.36 |
      | 2026-07-15 | NETFLIX ENTRETENIMENTO | -59.90  |
      | 2026-07-28 | DL*GOOGLE YouTube      | -24.90  |
      | 2026-08-02 | X9#K*@@ 88812          | -32.09  |
    Then the import created 5 and skipped 0
    And every item of the document is in the app
    And the invoice of card "Inter" due in "2026-08" matches the document total
    And the available limit of card "Inter" is 9676.85
    And the balance of wallet "Checking" is 3000.00

  Scenario: An item the AI could not categorize goes to the expense fallback
    When I import an invoice into card "Inter" read as:
      | date       | description       | amount |
      | 2026-07-05 | UBER *TRIP HELP.U | -18.90 |
    Then "UBER *TRIP HELP.U" is uncategorized

  Scenario: A category suggested by the AI is used
    Given the AI suggests category "Transport" for "UBER *TRIP HELP.U"
    When I import an invoice into card "Inter" read as:
      | date       | description       | amount |
      | 2026-07-05 | UBER *TRIP HELP.U | -18.90 |
    Then "UBER *TRIP HELP.U" is in category "Transport"

  Scenario: A description categorized in the past keeps its category
    Given a credit card purchase "UBER *TRIP HELP.U" of 20.00 on "2026-07-01" on card "Inter" under category "Transport"
    When I import an invoice into card "Inter" read as:
      | date       | description       | amount |
      | 2026-07-05 | UBER *TRIP HELP.U | -18.90 |
    Then the import created 1 and skipped 0
    And "UBER *TRIP HELP.U" is in category "Transport"

  Scenario: A refund reduces the invoice and gives the limit back
    When I import an invoice into card "Inter" read as:
      | date       | description         | amount  |
      | 2026-07-12 | MERCADO PIRATININGA | -187.36 |
      | 2026-07-24 | ESTORNO DROGASIL    | 32.40   |
    Then the import created 2 and skipped 0
    And the invoice of card "Inter" due in "2026-08" has amount -154.96
    And the available limit of card "Inter" is 9845.04

  # Achado 1 do AYD-004, sem decisão: hoje o estorno vira "Uncategorized Income".
  @wip
  Scenario: A refund is not classified as income
    When I import an invoice into card "Inter" read as:
      | date       | description      | amount |
      | 2026-07-24 | ESTORNO DROGASIL | 32.40  |
    Then "ESTORNO DROGASIL" is uncategorized

  Scenario: The whole document enters and the invoice closes with its total
    When I import the invoice "inter-2026-08" into card "Inter"
    Then every item of the document is in the app
    And the invoice of card "Inter" due in "2026-08" matches the document total

  Scenario: A password-protected document is read after the right password
    When I extract the protected invoice "inter-2026-08" for card "Inter" with password "1234"
    Then the extraction marks:
      | description                     | as              |
      | PAGAMENTO ON LINE               | invoice payment |
      | SAMSUNG*LOJA (Parcela 01 de 10) | new             |

  Scenario: A password-protected document without the password is rejected
    When I extract the protected invoice "inter-2026-08" for card "Inter"
    Then the operation is rejected as "password required"

  Scenario: A password-protected document with a wrong password is rejected
    When I extract the protected invoice "inter-2026-08" for card "Inter" with password "0000"
    Then the operation is rejected as "wrong password"
