@journey @import @invoice
Feature: Three months of imported invoices
  Importing consecutive invoices of the same card keeps every invoice equal to its document,
  links the installments that continue and never duplicates a series.

  # R1 (SPEC-009): as faturas reais trazem parcelas datadas pela compra original.
  @known-bug
  Scenario: Importing August, September and October
    Given today is "2026-08-05"
    And a wallet "Checking" with balance 5000.00
    And a credit card "Inter" with limit 10000.00, closing day 3 and due day 10, paid from wallet "Checking"
    When I import the invoice "inter-2026-08" into card "Inter"
    Then every item of the document is in the app
    And the invoice of card "Inter" due in "2026-08" matches the document total

    Given it is now "2026-09-05"
    When I import the invoice "inter-2026-09" into card "Inter"
    Then the extraction marks:
      | description                           | as              |
      | PAGAMENTO ON LINE                     | invoice payment |
      | DECATHLON (Parcela 01 de 04)          | new             |
    And every item of the document is in the app
    And the invoice of card "Inter" due in "2026-09" matches the document total

    Given I paid the invoice of card "Inter" due in "2026-08" from wallet "Checking"
    And it is now "2026-10-05"
    When I import the invoice "inter-2026-10" into card "Inter"
    Then every item of the document is in the app
    And the invoice of card "Inter" due in "2026-10" matches the document total
    And the installments of "SHOPEE *UTILIDADES" on card "Inter" are:
      | installment | due     | amount |
      | 1           | 2026-08 | -83.65 |
      | 2           | 2026-09 | -83.65 |
      | 3           | 2026-10 | -83.64 |
    And the installments of "AUTOCENTER PRIMO" on card "Inter" are:
      | installment | due     | amount  |
      | 4           | 2026-08 | -137.33 |
      | 5           | 2026-09 | -137.33 |
      | 6           | 2026-10 | -137.33 |
