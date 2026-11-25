@journey @credit-card @installments
Feature: A month in the life of a credit card

  Scenario: Buy in installments, pay the invoice, cancel the remaining installments and revert the payment
    Given today is "2026-03-01"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Nubank" with limit 5000.00, closing day 5 and due day 12, paid from wallet "Checking"

    When I make a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    And I make a credit card purchase "Shoes" of 300.00 on "2026-03-10" on card "Nubank" under category "Electronics"
    Then the available limit of card "Nubank" is 3500.00
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | -700.00 | no   |
      | 2026-05 | -400.00 | no   |

    When it is now "2026-03-12"
    And I pay the invoice of card "Nubank" due in "2026-03" from wallet "Checking"
    Then the balance of wallet "Checking" is 2600.00
    And the available limit of card "Nubank" is 3900.00
    And installment 1 of "TV" is paid

    When I delete installment 2 of "TV" and all next
    Then the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | yes  |
      | 2026-04 | -300.00 | no   |
      | 2026-05 | 0.00    | no   |
    And the available limit of card "Nubank" is 4700.00

    When I revert the payment of the invoice of card "Nubank" due in "2026-03"
    Then the balance of wallet "Checking" is 3000.00
    And the available limit of card "Nubank" is 4300.00
    And the invoice of card "Nubank" due in "2026-03" is open
