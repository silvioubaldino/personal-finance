@credit-card @invoice
Feature: Pay an invoice
  Paying an invoice debits the wallet and gives the limit back; a partial payment carries
  the rest to the next invoice; reverting the payment undoes all of it.

  Background:
    Given today is "2026-03-12"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Home"
    And a credit card "Nubank" with limit 5000.00, closing day 5 and due day 12, paid from wallet "Checking"
    And a credit card purchase "Sofa" of 600.00 on "2026-03-03" on card "Nubank" under category "Home"

  Scenario: Paying the whole invoice debits the wallet and gives the limit back
    When I pay the invoice of card "Nubank" due in "2026-03" from wallet "Checking"
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" is paid
    And the balance of wallet "Checking" is 2400.00
    And the available limit of card "Nubank" is 5000.00
    And "Sofa" is paid

  Scenario: The payment date defaults to today
    When I pay the invoice of card "Nubank" due in "2026-03" from wallet "Checking"
    Then the invoice of card "Nubank" due in "2026-03" was paid on "2026-03-12"

  Scenario: Paying part of the invoice carries the rest to the next invoice
    When I pay the invoice of card "Nubank" due in "2026-03" from wallet "Checking" paying 200.00
    Then the operation succeeds
    And the balance of wallet "Checking" is 2800.00
    And the available limit of card "Nubank" is 4600.00
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -600.00 | yes  |
      | 2026-04 | -400.00 | no   |

  Scenario: Reverting the payment of an invoice undoes it
    Given I paid the invoice of card "Nubank" due in "2026-03" from wallet "Checking"
    When I revert the payment of the invoice of card "Nubank" due in "2026-03"
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" is open
    And the balance of wallet "Checking" is 3000.00
    And the available limit of card "Nubank" is 4400.00
    And "Sofa" is pending

  Scenario: Reverting a partial payment removes the carried amount from the next invoice
    Given I paid the invoice of card "Nubank" due in "2026-03" from wallet "Checking" paying 200.00
    When I revert the payment of the invoice of card "Nubank" due in "2026-03"
    Then the operation succeeds
    And the balance of wallet "Checking" is 3000.00
    And the available limit of card "Nubank" is 4400.00
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -600.00 | no   |
      | 2026-04 | 0.00    | no   |

  Scenario: An invoice larger than the wallet balance cannot be paid
    Given a wallet "Savings" with balance 100.00
    When I pay the invoice of card "Nubank" due in "2026-03" from wallet "Savings"
    Then the operation is rejected as "insufficient balance"
    And the invoice of card "Nubank" due in "2026-03" is open
    And the balance of wallet "Savings" is 100.00
    And the available limit of card "Nubank" is 4400.00

  Scenario: Recalculating a consistent invoice changes nothing
    When I recalculate the invoice of card "Nubank" due in "2026-03"
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" has amount -600.00
    And the available limit of card "Nubank" is 4400.00

  @known-bug
  Scenario: Paying an invoice that is already paid is rejected
    Given I paid the invoice of card "Nubank" due in "2026-03" from wallet "Checking"
    When I pay the invoice of card "Nubank" due in "2026-03" from wallet "Checking"
    Then the operation is rejected as "conflict"
    And the balance of wallet "Checking" is 2400.00

  @known-bug
  Scenario: Reverting the payment of an open invoice is rejected
    When I revert the payment of the invoice of card "Nubank" due in "2026-03"
    Then the operation is rejected as "conflict"
    And the balance of wallet "Checking" is 3000.00

  @known-bug
  Scenario Outline: A payment amount outside the invoice range is rejected
    When I pay the invoice of card "Nubank" due in "2026-03" from wallet "Checking" paying <amount>
    Then the operation is rejected as "invalid input"
    And the invoice of card "Nubank" due in "2026-03" is open
    And the balance of wallet "Checking" is 3000.00

    Examples:
      | amount  |
      | 0.00    |
      | 1000.00 |
