@import @invoice @rejections
Feature: Invoice imports the app refuses

  Background:
    Given today is "2026-08-05"
    And a wallet "Checking" with balance 3000.00
    And an expense category "Electronics"
    And a credit card "Inter" with limit 10000.00, closing day 3 and due day 10, paid from wallet "Checking"

  Scenario: An import above the available limit is rejected and nothing of it is created
    Given a credit card "Small" with limit 100.00, closing day 3 and due day 10, paid from wallet "Checking"
    When I import an invoice into card "Small" read as:
      | date       | description         | amount  |
      | 2026-07-12 | MERCADO PIRATININGA | -500.00 |
    Then the operation is rejected as "forbidden"
    And the available limit of card "Small" is 100.00

  Scenario: Importing into a paid invoice is rejected and nothing is created
    Given a credit card purchase "Shoes" of 50.00 on "2026-07-10" on card "Inter" under category "Electronics"
    And I paid the invoice of card "Inter" due in "2026-08" from wallet "Checking"
    When I import an invoice into card "Inter" read as:
      | date       | description         | amount  |
      | 2026-08-10 | POSTO SHELL         | -220.00 |
      | 2026-07-12 | MERCADO PIRATININGA | -187.36 |
    Then the operation is rejected as "invoice already paid"
    And the invoice of card "Inter" due in "2026-08" has amount -50.00
    And the available limit of card "Inter" is 10000.00
