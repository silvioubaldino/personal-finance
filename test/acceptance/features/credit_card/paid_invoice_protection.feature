@credit-card @invoice-paid
Feature: Purchases in a paid invoice are frozen
  Once an invoice is paid its purchases cannot be changed or removed;
  the user has to revert the invoice payment first.

  Background:
    Given today is "2026-03-20"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Home"
    And a credit card "Nubank" with limit 5000.00, closing day 5 and due day 12, paid from wallet "Checking"
    And a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Home"
    And I paid the invoice of card "Nubank" due in "2026-03" from wallet "Checking"

  Scenario Outline: Changing a purchase of a paid invoice is rejected
    When I <action>
    Then the operation is rejected as "conflict"
    And the invoice of card "Nubank" due in "2026-03" has amount -200.00
    And the available limit of card "Nubank" is 5000.00
    And the balance of wallet "Checking" is 2800.00

    # The answer is a 500 today: the business errors are not mapped in HandleErr (K1).
    @known-bug
    Examples: Update one, delete one and delete all next
      | action                                       |
      | update only "Lamp" setting amount to -250.00 |
      | delete only "Lamp"                           |
      | delete "Lamp" and all next                   |

    # Update all next goes through the single-update path, which ignores the paid invoice (K4).
    @known-bug
    Examples: Update all next
      | action                                               |
      | update "Lamp" and all next setting amount to -250.00 |

  Scenario: The purchase can be changed again after the invoice payment is reverted
    Given I revert the payment of the invoice of card "Nubank" due in "2026-03"
    When I update only "Lamp" setting amount to -250.00
    Then the operation succeeds
    And the invoice of card "Nubank" due in "2026-03" has amount -250.00
    And the available limit of card "Nubank" is 4750.00
    And the balance of wallet "Checking" is 3000.00

  Scenario: A purchase of a paid invoice cannot be paid or reverted on its own
    When I pay "Lamp"
    Then the operation is rejected as "conflict"
