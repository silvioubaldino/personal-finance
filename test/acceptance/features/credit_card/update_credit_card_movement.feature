@credit-card @invoice
Feature: Update a credit card purchase
  Changing a purchase of an open invoice changes the invoice and the card limit by the
  difference; the wallet is never touched.

  Background:
    Given today is "2026-03-01"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Nubank" with limit 5000.00, closing day 5 and due day 12, paid from wallet "Checking"

  @update-one
  Scenario: Updating the amount of a purchase adjusts its invoice and the limit by the difference
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    When I update only "Lamp" setting amount to -250.00
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" has amount -250.00
    And the available limit of card "Nubank" is 4750.00
    And the balance of wallet "Checking" is 3000.00

  @update-one
  Scenario: Updating a purchase leaves the other purchases of the invoice alone
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    And a credit card purchase "Chair" of 300.00 on "2026-03-04" on card "Nubank" under category "Electronics"
    When I update only "Lamp" setting amount to -250.00
    Then the invoice of card "Nubank" due in "2026-03" contains:
      | description | amount  | installment |
      | Lamp        | -250.00 |             |
      | Chair       | -300.00 |             |
    And the available limit of card "Nubank" is 4450.00

  @update-one
  Scenario: Updating one installment changes only the invoice of that installment
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    When I update only installment 2 of "TV" setting amount to -500.00
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | -500.00 | no   |
      | 2026-05 | -400.00 | no   |
    And the available limit of card "Nubank" is 3700.00

  @update-one @installments
  Scenario: Updating an installment is allowed while only its own invoice is open
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    And I paid the invoice of card "Nubank" due in "2026-03" from wallet "Checking"
    When I update only installment 2 of "TV" setting amount to -450.00
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-04" has amount -450.00

  @update-one @known-bug
  Scenario: Raising a purchase above the available limit is rejected
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    When I update only "Lamp" setting amount to -6000.00
    Then the operation is rejected as "insufficient limit"
    And the invoice of card "Nubank" due in "2026-03" has amount -200.00
    And the available limit of card "Nubank" is 4800.00

  @update-one @known-bug
  Scenario: Moving a purchase past the closing day moves it to the next invoice
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    When I update only "Lamp" with:
      | date | 2026-03-10 |
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | 0.00    | no   |
      | 2026-04 | -200.00 | no   |
    And the available limit of card "Nubank" is 4800.00

  @update-all-next @known-bug
  Scenario: Updating a standalone purchase and all next behaves like updating only it
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Electronics"
    When I update "Lamp" and all next setting amount to -250.00
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" has amount -250.00
    And the available limit of card "Nubank" is 4750.00
    And the balance of wallet "Checking" is 3000.00

  @update-all-next @installments @known-bug
  Scenario: Updating an installment and all next rewrites the remaining installments
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    When I update installment 2 of "TV" and all next setting amount to -500.00
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | -500.00 | no   |
      | 2026-05 | -500.00 | no   |
    And the available limit of card "Nubank" is 3600.00

  @update-all-next @installments @known-bug
  Scenario: Updating an installment and all next is rejected when a later invoice is already paid
    Given a credit card purchase "TV" in 3 installments of 400.00 starting "2026-03-03" on card "Nubank" under category "Electronics"
    And I paid the invoice of card "Nubank" due in "2026-04" from wallet "Checking"
    When I update installment 1 of "TV" and all next setting amount to -500.00
    Then the operation is rejected as "conflict"
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -400.00 | no   |
      | 2026-04 | -400.00 | yes  |
      | 2026-05 | -400.00 | no   |
