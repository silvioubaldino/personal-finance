@credit-card @invoice
Feature: Delete a credit card purchase
  Deleting a purchase of an open invoice takes it out of the invoice and gives its limit
  back; the wallet is never touched.

  Background:
    Given today is "2026-03-01"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Nubank" with limit 5000.00, closing day 5 and due day 12, paid from wallet "Checking"

  @delete-one
  Scenario: Deleting a purchase restores the invoice and the limit
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    When I delete only "Lamp"
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" has amount 0.00
    And the available limit of card "Nubank" is 5000.00
    And the balance of wallet "Checking" is 3000.00

  @delete-one
  Scenario: Deleting a purchase leaves the other purchases of the invoice alone
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    And a credit card purchase "Chair" of 300.00 on "2026-03-04" on card "Nubank" under category "Electronics"
    When I delete only "Lamp"
    Then the invoice of card "Nubank" due in "2026-03" contains:
      | description | amount  | installment |
      | Chair       | -300.00 |             |
    And the available limit of card "Nubank" is 4700.00

  @delete-all-next
  Scenario: Deleting a standalone purchase and all next behaves like deleting only it
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    When I delete "Lamp" and all next
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" has amount 0.00
    And the available limit of card "Nubank" is 5000.00

  @delete-one @installments
  Scenario: Deleting one installment leaves the others
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    When I delete only installment 2 of "TV"
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | 0.00    | no   |
      | 2026-05 | -400.00 | no   |
    And the available limit of card "Nubank" is 4200.00

  @delete-all-next @installments
  Scenario: Deleting an installment and all next removes it and the following ones
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    When I delete installment 2 of "TV" and all next
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | 0.00    | no   |
      | 2026-05 | 0.00    | no   |
    And the available limit of card "Nubank" is 4600.00

  @delete-all-next @installments
  Scenario: Deleting the first installment and all next removes the whole purchase
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    When I delete installment 1 of "TV" and all next
    Then the operation succeeds
    And the available limit of card "Nubank" is 5000.00
    And the invoices of card "Nubank" are:
      | due     | amount | paid |
      | 2026-03 | 0.00   | no   |
      | 2026-04 | 0.00   | no   |
      | 2026-05 | 0.00   | no   |

  @delete-one @installments
  Scenario: Deleting an installment is allowed while only a later invoice is paid
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    And I paid the invoice of card "Nubank" due in "2026-04" from wallet "Checking"
    When I delete only installment 1 of "TV"
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" has amount 0.00

  @delete-all-next @installments @known-bug
  Scenario: Deleting an installment and all next is rejected as a whole when a later invoice is paid
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    And I paid the invoice of card "Nubank" due in "2026-04" from wallet "Checking"
    When I delete installment 1 of "TV" and all next
    Then the operation is rejected as "conflict"
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | -400.00 | yes  |
      | 2026-05 | -400.00 | no   |
    And the available limit of card "Nubank" is 4200.00
