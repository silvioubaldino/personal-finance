@import @invoice @installments
Feature: Installments in an imported invoice
  A new installment purchase creates the rest of its series; an installment of a series the
  app already knows is linked to it — the value is corrected, nothing is duplicated.

  Background:
    Given today is "2026-08-05"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Inter" with limit 10000.00, closing day 3 and due day 10, paid from wallet "Checking"

  Scenario: A new installment purchase creates the remaining series
    When I import an invoice into card "Inter" read as:
      | date       | description                           | amount | installment |
      | 2026-07-09 | SHOPEE *UTILIDADES (Parcela 01 de 03) | -83.65 | 1/3         |
    Then the import created 3 and skipped 0
    And the installments of "SHOPEE *UTILIDADES" on card "Inter" are:
      | installment | due     | amount |
      | 1           | 2026-08 | -83.65 |
      | 2           | 2026-09 | -83.65 |
      | 3           | 2026-10 | -83.65 |

  Scenario: A purchase made before the app creates only its remaining installments
    When I import an invoice into card "Inter" read as:
      | date       | description                         | amount  | installment |
      | 2026-07-17 | AUTOCENTER PRIMO (Parcela 04 de 06) | -137.33 | 4/6         |
    Then the import created 3 and skipped 0
    And the installments of "AUTOCENTER PRIMO" on card "Inter" are:
      | installment | due     | amount  |
      | 4           | 2026-08 | -137.33 |
      | 5           | 2026-09 | -137.33 |
      | 6           | 2026-10 | -137.33 |

  Scenario: An installment dated by the original purchase lands in the imported invoice
    When I import an invoice into card "Inter" read as:
      | date       | description                         | amount  | installment |
      | 2026-07-05 | UBER *TRIP HELP.U                   | -18.90  |             |
      | 2026-07-12 | MERCADO PIRATININGA                 | -187.36 |             |
      | 2026-04-17 | AUTOCENTER PRIMO (Parcela 04 de 06) | -137.33 | 4/6         |
    Then the installments of "AUTOCENTER PRIMO" on card "Inter" are:
      | installment | due     | amount  |
      | 4           | 2026-08 | -137.33 |
      | 5           | 2026-09 | -137.33 |
      | 6           | 2026-10 | -137.33 |
    And the invoice of card "Inter" due in "2026-08" has amount -343.59

  Scenario: An installment registered by hand is linked and its value corrected
    Given it is now "2026-09-05"
    And a credit card purchase "SAMSUNG*LOJA" in 3 installments of 349.90 starting "2026-07-20" on card "Inter" under category "Electronics"
    When I import an invoice into card "Inter" read as:
      | date       | description             | amount  | installment |
      | 2026-07-20 | SAMSUNG*LOJA PARC 02/03 | -349.93 | 2/3         |
      | 2026-08-05 | UBER *TRIP HELP.U       | -22.40  |             |
    Then the extraction marks:
      | description             | as                       |
      | SAMSUNG*LOJA PARC 02/03 | linked to "SAMSUNG*LOJA" |
    And the import created 1 and skipped 2
    And the installments of "SAMSUNG*LOJA" on card "Inter" are:
      | installment | due     | amount  |
      | 1           | 2026-08 | -349.90 |
      | 2           | 2026-09 | -349.93 |
      | 3           | 2026-10 | -349.90 |
    And the invoice of card "Inter" due in "2026-09" has amount -372.33
    And the available limit of card "Inter" is 8927.87

  Scenario: An installment in the Inter format is linked to the purchase registered by hand
    Given it is now "2026-09-05"
    And a credit card purchase "SAMSUNG*LOJA" in 3 installments of 349.90 starting "2026-07-20" on card "Inter" under category "Electronics"
    When I import an invoice into card "Inter" read as:
      | date       | description                     | amount  | installment |
      | 2026-07-20 | SAMSUNG*LOJA (Parcela 02 de 03) | -349.93 | 2/3         |
      | 2026-08-05 | UBER *TRIP HELP.U               | -22.40  |             |
    Then the extraction marks:
      | description                     | as                       |
      | SAMSUNG*LOJA (Parcela 02 de 03) | linked to "SAMSUNG*LOJA" |
    And the installments of "SAMSUNG*LOJA" on card "Inter" are:
      | installment | due     | amount  |
      | 1           | 2026-08 | -349.90 |
      | 2           | 2026-09 | -349.93 |
      | 3           | 2026-10 | -349.90 |

  Scenario: A purchase registered under another name comes only as a suggestion
    Given it is now "2026-09-05"
    And a credit card purchase "Notebook" in 3 installments of 349.90 starting "2026-07-20" on card "Inter" under category "Electronics"
    When I extract an invoice for card "Inter" read as:
      | date       | description             | amount  | installment |
      | 2026-07-20 | SAMSUNG*LOJA PARC 02/03 | -349.93 | 2/3         |
      | 2026-08-05 | UBER *TRIP HELP.U       | -22.40  |             |
    Then the extraction marks:
      | description             | as                   |
      | SAMSUNG*LOJA PARC 02/03 | suggested "Notebook" |

  Scenario: Accepting the suggestion links the installment to the registered purchase
    Given it is now "2026-09-05"
    And a credit card purchase "Notebook" in 3 installments of 349.90 starting "2026-07-20" on card "Inter" under category "Electronics"
    When I extract an invoice for card "Inter" read as:
      | date       | description             | amount  | installment |
      | 2026-07-20 | SAMSUNG*LOJA PARC 02/03 | -349.93 | 2/3         |
      | 2026-08-05 | UBER *TRIP HELP.U       | -22.40  |             |
    And I accept the suggested link for "SAMSUNG*LOJA PARC 02/03"
    And I confirm the import
    Then the import created 1 and skipped 2
    And the installments of "Notebook" on card "Inter" are:
      | installment | due     | amount  |
      | 1           | 2026-08 | -349.90 |
      | 2           | 2026-09 | -349.93 |
      | 3           | 2026-10 | -349.90 |

  Scenario: A link to a series that does not fit falls back to creating the purchase
    Given a credit card purchase "Notebook" in 10 installments of 100.00 starting "2026-07-20" on card "Inter" under category "Electronics"
    When I extract an invoice for card "Inter" read as:
      | date       | description                   | amount | installment |
      | 2026-07-25 | COMPRA X (Parcela 01 de 03)   | -50.00 | 1/3         |
    And I link "COMPRA X (Parcela 01 de 03)" to the purchase "Notebook"
    And I confirm the import
    Then the import created 3 and skipped 0
    And the installments of "COMPRA X" on card "Inter" are:
      | installment | due     | amount |
      | 1           | 2026-08 | -50.00 |
      | 2           | 2026-09 | -50.00 |
      | 3           | 2026-10 | -50.00 |
