@credit-card @installments
Feature: Installment purchases
  An installment purchase puts one installment in each following invoice and uses up the
  limit for the whole purchase at once.

  Background:
    Given today is "2026-03-01"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Nubank" with limit 5000.00, closing day 5 and due day 12, paid from wallet "Checking"

  Scenario: Installments are spread over consecutive invoices
    When I make a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | -400.00 | no   |
      | 2026-05 | -400.00 | no   |

  Scenario: Each invoice item carries its installment number
    When I make a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    Then the invoice of card "Nubank" due in "2026-04" contains:
      | description | amount  | installment |
      | TV          | -400.00 | 2/3         |

  Scenario: The whole purchase uses up the limit at once
    When I make a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    Then the available limit of card "Nubank" is 3800.00
    And the balance of wallet "Checking" is 3000.00

  Scenario: Installments after the closing day start in the next invoice
    When I make a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-10" on card "Nubank" under category "Electronics"
    Then the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-04 | -400.00 | no   |
      | 2026-05 | -400.00 | no   |
      | 2026-06 | -400.00 | no   |

  Scenario: Installments share the invoice with other purchases
    Given a credit card purchase "Shoes" of 300.00 on "2026-03-10" on card "Nubank" under category "Electronics"
    When I make a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    Then the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | -700.00 | no   |
      | 2026-05 | -400.00 | no   |

  @known-bug
  Scenario: An installment purchase above the available limit is rejected as a whole
    When I make a credit card purchase "TV" in 3 installments of 2000.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    Then the operation is rejected as "insufficient limit"
    And the available limit of card "Nubank" is 5000.00
    And the invoice of card "Nubank" due in "2026-03" does not exist
