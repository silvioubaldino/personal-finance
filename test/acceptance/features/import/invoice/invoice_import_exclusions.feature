@import @invoice @exclusions
Feature: Items that do not belong to the imported invoice
  The previous invoice's payment and installments of later invoices are marked and never
  enter the invoice, even when the client sends them back.

  Background:
    Given today is "2026-08-05"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Inter" with limit 10000.00, closing day 3 and due day 10, paid from wallet "Checking"

  Scenario: The previous invoice payment is marked and not imported
    When I import an invoice into card "Inter" read as:
      | date       | description         | amount  |
      | 2026-07-10 | PAGAMENTO ON LINE   | 2310.45 |
      | 2026-07-12 | MERCADO PIRATININGA | -187.36 |
    Then the extraction marks:
      | description         | as              |
      | PAGAMENTO ON LINE   | invoice payment |
      | MERCADO PIRATININGA | new             |
    And the import created 1 and skipped 1
    And the invoice of card "Inter" due in "2026-08" has amount -187.36
    And the available limit of card "Inter" is 9812.64

  Scenario: The server refuses the payment even if the client re-includes it
    When I extract an invoice for card "Inter" read as:
      | date       | description         | amount  |
      | 2026-07-10 | PAGAMENTO ON LINE   | 2310.45 |
      | 2026-07-12 | MERCADO PIRATININGA | -187.36 |
    And I re-include "PAGAMENTO ON LINE"
    And I confirm the import
    Then the import created 1 and skipped 1
    And the invoice of card "Inter" due in "2026-08" has amount -187.36

  Scenario: Merchants whose name starts like a payment are imported
    When I import an invoice into card "Inter" read as:
      | date       | description      | amount |
      | 2026-07-10 | PAGUE MENOS 1234 | -45.00 |
      | 2026-07-11 | PAGSEGURO *LOJA  | -30.00 |
      | 2026-07-12 | PAGBANK SERVICOS | -20.00 |
    Then the extraction marks:
      | description      | as  |
      | PAGUE MENOS 1234 | new |
      | PAGSEGURO *LOJA  | new |
      | PAGBANK SERVICOS | new |
    And the import created 3 and skipped 0

  Scenario: An installment of a later invoice is excluded; an ordinary purchase on the same date is not
    When I import an invoice into card "Inter" read as:
      | date       | description                     | amount  | installment |
      | 2026-07-05 | UBER *TRIP HELP.U               | -18.90  |             |
      | 2026-07-12 | MERCADO PIRATININGA             | -187.36 |             |
      | 2026-07-15 | NETFLIX ENTRETENIMENTO          | -59.90  |             |
      | 2026-08-20 | DROGARIA SP (Parcela 03 de 03)  | -198.67 | 3/3         |
      | 2026-08-20 | POSTO SHELL AV BRASIL           | -220.00 |             |
    Then the extraction marks:
      | description                    | as                 |
      | DROGARIA SP (Parcela 03 de 03) | future installment |
      | POSTO SHELL AV BRASIL          | new                |
    And the import created 4 and skipped 1
    And the invoice of card "Inter" due in "2026-08" has amount -266.16
    And the invoice of card "Inter" due in "2026-09" has amount -220.00
