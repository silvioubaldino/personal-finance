@credit-card @invoice
Feature: Credit card purchases go to the invoice of their billing period
  The invoice a purchase belongs to depends on the card's closing day; a purchase uses
  up the card limit but does not touch any wallet until the invoice is paid.

  Background:
    Given today is "2026-03-01"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Home"
    And a credit card "Nubank" with limit 5000.00, closing day 5 and due day 12, paid from wallet "Checking"

  Scenario: A purchase up to the closing day goes to that month's invoice
    When I make a credit card purchase "Lamp" of 200.00 on "2026-03-05" on card "Nubank" under category "Home"
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -200.00 | no   |

  Scenario: A purchase after the closing day goes to the next month's invoice
    When I make a credit card purchase "Lamp" of 200.00 on "2026-03-06" on card "Nubank" under category "Home"
    Then the operation succeeds
    And the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-04 | -200.00 | no   |

  Scenario: Purchases of the same billing period share one invoice
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-01" on card "Nubank" under category "Home"
    When I make a credit card purchase "Chair" of 300.00 on "2026-03-04" on card "Nubank" under category "Home"
    Then the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -500.00 | no   |
    And the invoice of card "Nubank" due in "2026-03" contains:
      | description | amount  | installment |
      | Lamp        | -200.00 |             |
      | Chair       | -300.00 |             |

  Scenario: Purchases of different billing periods go to different invoices
    Given a credit card purchase "Lamp" of 200.00 on "2026-03-05" on card "Nubank" under category "Home"
    When I make a credit card purchase "Chair" of 300.00 on "2026-03-06" on card "Nubank" under category "Home"
    Then the invoices of card "Nubank" are:
      | due     | amount  | paid |
      | 2026-03 | -200.00 | no   |
      | 2026-04 | -300.00 | no   |

  Scenario: A purchase uses up the card limit and leaves the wallet alone
    When I make a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Home"
    Then the available limit of card "Nubank" is 4800.00
    And the balance of wallet "Checking" is 3000.00

  Scenario: A purchase is an item of the invoice, not a wallet movement
    When I make a credit card purchase "Lamp" of 200.00 on "2026-03-03" on card "Nubank" under category "Home"
    Then the movements of wallet "Checking" in "2026-03" are:
      | description | amount | paid |
    And the invoice of card "Nubank" due in "2026-03" contains:
      | description | amount  | installment |
      | Lamp        | -200.00 |             |

  Scenario: A purchase near the end of the year goes to the invoice of the next year
    Given a credit card "Inter" with limit 2000.00, closing day 25 and due day 5, paid from wallet "Checking"
    When I make a credit card purchase "Gift" of 150.00 on "2026-12-20" on card "Inter" under category "Home"
    Then the invoices of card "Inter" are:
      | due     | amount  | paid |
      | 2027-01 | -150.00 | no   |

  @known-bug
  Scenario: A purchase above the available limit is rejected
    When I make a credit card purchase "Boat" of 6000.00 on "2026-03-03" on card "Nubank" under category "Home"
    Then the operation is rejected as "insufficient limit"
    And the available limit of card "Nubank" is 5000.00
    And the invoice of card "Nubank" due in "2026-03" does not exist
