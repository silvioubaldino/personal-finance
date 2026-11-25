@journey @recurrence @transfer @paid
Feature: A few months of salary and bills

  Scenario: Salary and rent are paid, savings are topped up, the rent goes up and then ends
    Given today is "2026-03-01"
    And a wallet "Checking" with balance 500.00
    And a wallet "Savings" with balance 0.00
    And an income category "Salary"
    And an expense category "Home"
    And a pending monthly recurrent income "Pay" of 3000.00 starting "2026-03-05" in wallet "Checking" under category "Salary"
    And a pending monthly recurrent expense "Rent" of 1200.00 starting "2026-03-10" in wallet "Checking" under category "Home"

    When it is now "2026-03-10"
    And I pay the occurrence of "Pay" in "2026-03"
    And I pay the occurrence of "Rent" in "2026-03"
    And I add a paid internal transfer "Reserve" of 500.00 from "Checking" to "Savings" on "2026-03-10"
    Then the balance of wallet "Checking" is 1800.00
    And the balance of wallet "Savings" is 500.00

    When I update the occurrence of "Rent" in "2026-04" and all next with:
      | amount | -1300.00 |
    Then the occurrences of "Rent" are:
      | month   | amount   | paid |
      | 2026-03 | -1200.00 | yes  |
      | 2026-04 | -1300.00 | no   |
      | 2026-08 | -1300.00 | no   |

    When it is now "2026-04-10"
    And I pay the occurrence of "Pay" in "2026-04"
    And I pay the occurrence of "Rent" in "2026-04"
    Then the balance of wallet "Checking" is 3500.00

    When I delete the internal transfer "Reserve"
    Then the balance of wallet "Checking" is 4000.00
    And the balance of wallet "Savings" is 0.00

    When I delete the occurrence of "Rent" in "2026-05" and all next
    Then the occurrences of "Rent" are:
      | month   | amount   | paid |
      | 2026-03 | -1200.00 | yes  |
      | 2026-04 | -1300.00 | yes  |
    And there is no occurrence of "Rent" in "2026-05"
    And the balance of wallet "Checking" is 4000.00
