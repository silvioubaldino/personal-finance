
# Changelog - personal-finance

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

- Added invoice partial payment [PR#131](https://github.com/silvioubaldino/personal-finance/pull/131)
- Updated creditcard limit [PR#130](https://github.com/silvioubaldino/personal-finance/pull/130)
- Added creditcard movement delete all next [PR#129](https://github.com/silvioubaldino/personal-finance/pull/129)
- Added creditcard movement delete one [PR#128](https://github.com/silvioubaldino/personal-finance/pull/128)
- Added creditcard movement update one [PR#127](https://github.com/silvioubaldino/personal-finance/pull/127)
- Refactored movement handler UpdateOne [PR#126](https://github.com/silvioubaldino/personal-finance/pull/126)
- Refactored movement usecase UpdateOne [PR#125](https://github.com/silvioubaldino/personal-finance/pull/125)
- Added credit installment usecase [PR#124](https://github.com/silvioubaldino/personal-finance/pull/124)
- Fixed revert invoice payment to delete only invoice payment type - [PR#123](https://github.com/silvioubaldino/personal-finance/pull/123)
- Added credit installment model and migrations - [PR#122](https://github.com/silvioubaldino/personal-finance/pull/122)
- Added credit card pay validation - [PR#121](https://github.com/silvioubaldino/personal-finance/pull/121)
- Added revert payment endpoint - [PR#120](https://github.com/silvioubaldino/personal-finance/pull/120)
- Fixed Pay invoice - [PR#119](https://github.com/silvioubaldino/personal-finance/pull/119)
- Added get credit card with invoices - [PR#118](https://github.com/silvioubaldino/personal-finance/pull/118)
- Added get detailed invoices by period - [PR#117](https://github.com/silvioubaldino/personal-finance/pull/117)

## Release - v1.4.0 - 19-08-2025

- Fixed update movement to update subcategory - [PR#114](https://github.com/silvioubaldino/personal-finance/pull/114)
- Fixed add movement to save type payment - [PR#113](https://github.com/silvioubaldino/personal-finance/pull/113)
- Added credit card and invoice handlers - [PR#112](https://github.com/silvioubaldino/personal-finance/pull/112)
- Added credit card and invoice usecase - [PR#111](https://github.com/silvioubaldino/personal-finance/pull/111)
- Added credit card and invoice repository - [PR#110](https://github.com/silvioubaldino/personal-finance/pull/110)
- Added credit card model and migrations - [PR#109](https://github.com/silvioubaldino/personal-finance/pull/109)

## Release - v1.3.2 - 09-08-2025

- Blocked write operations for default categories [PR#106](https://github.com/silvioubaldino/personal-finance/pull/106)
- Removed auto request review [PR#105](https://github.com/silvioubaldino/personal-finance/pull/105)
- Fixed recurrent type payment [PR#104](https://github.com/silvioubaldino/personal-finance/pull/104)

## Release - v1.3.1 - 07-08-2025

- Fixed update type_payment [PR#101](https://github.com/silvioubaldino/personal-finance/pull/101)

## Release - v1.3.0 - 07-08-2025

- Added return category is income [PR#98](https://github.com/silvioubaldino/personal-finance/pull/98)
- Changed type_payment type to string [PR#97](https://github.com/silvioubaldino/personal-finance/pull/97)

## Release - v1.2.0 - 12-06-2025

- Fixed and improved dockerfile [PR#95](https://github.com/silvioubaldino/personal-finance/pull/95)
- Refactored repository movement update one [PR#92](https://github.com/silvioubaldino/personal-finance/pull/92)
- Refactored handler movement revertPay [PR#91](https://github.com/silvioubaldino/personal-finance/pull/91)
- Refactored usecase repository revert pay [PR#90](https://github.com/silvioubaldino/personal-finance/pull/90)
- Refactored movement repository pay [PR#89](https://github.com/silvioubaldino/personal-finance/pull/89)
- Refactored movement usecase pay [PR#88](https://github.com/silvioubaldino/personal-finance/pull/88)
- Refactored movement handler pay [PR#87](https://github.com/silvioubaldino/personal-finance/pull/87)
- Fixed and improved findByPeriod flow [PR#86](https://github.com/silvioubaldino/personal-finance/pull/86)
- Refactored movement handler findByPeriod [PR#85](https://github.com/silvioubaldino/personal-finance/pull/85)
- Refactored movement usecase findByPeriod [PR#84](https://github.com/silvioubaldino/personal-finance/pull/84)
- Migrated recurrent repository find queries [PR#83](https://github.com/silvioubaldino/personal-finance/pull/83)
- Updated libs [PR#82](https://github.com/silvioubaldino/personal-finance/pull/82)
- Refactored movement repository find queries [PR#81](https://github.com/silvioubaldino/personal-finance/pull/81)
- Created error handling structure [PR#80](https://github.com/silvioubaldino/personal-finance/pull/80)
- Created logs structure [PR#79](https://github.com/silvioubaldino/personal-finance/pull/79)
- Refactored type payment and status. [PR#78](https://github.com/silvioubaldino/personal-finance/pull/78)
- Refactored db bootstrap [PR#77](https://github.com/silvioubaldino/personal-finance/pull/77)
- Created new bootstrap structure [PR#76](https://github.com/silvioubaldino/personal-finance/pull/76)
- Refactored add movement handler [PR#75]()https://github.com/silvioubaldino/personal-finance/pull/75
- Created UpdateAmount wallet query [PR#74](https://github.com/silvioubaldino/personal-finance/pull/74)
- Created IsSubCategoryBelongsToCategory query [PR#73](https://github.com/silvioubaldino/personal-finance/pull/73)
- Refactored Add recurrent repository [PR#72](https://github.com/silvioubaldino/personal-finance/pull/72)
- Refactored Add movement repository [PR#71](https://github.com/silvioubaldino/personal-finance/pull/71)
- Improved go context to transport userID [PR#70](https://github.com/silvioubaldino/personal-finance/pull/70)
- Refactored Add movement usecase [PR#69](https://github.com/silvioubaldino/personal-finance/pull/69)
- Removed deprecated fields [PR#66](https://github.com/silvioubaldino/personal-finance/pull/66)

## Release - v1.1.0 - 26-11-2024

- Corrected update all next logic [PR#64](https://github.com/silvioubaldino/personal-finance/pull/64) 
- Created db migrations [PR#63](https://github.com/silvioubaldino/personal-finance/pull/63)
- Corrected success returning 500 and improve error handling [PR#62](https://github.com/silvioubaldino/personal-finance/pull/62)
- Added delete one recurrent [PR#61](https://github.com/silvioubaldino/personal-finance/pull/61)
- Added delete all next endpoint [PR#60](https://github.com/silvioubaldino/personal-finance/pull/60)
- Added update all next endpoint [PR#59](https://github.com/silvioubaldino/personal-finance/pull/59)
- Added create movement when update recurrent [PR#58](https://github.com/silvioubaldino/personal-finance/pull/58)
- Added create movement when paying recurrent [PR#57](https://github.com/silvioubaldino/personal-finance/pull/57)
- Created Get recurrent movement [PR#56](https://github.com/silvioubaldino/personal-finance/pull/56)
- Created model and Add recurrent movement [PR#55](https://github.com/silvioubaldino/personal-finance/pull/55)
- Added update wallet when deleting movement [PR#54](https://github.com/silvioubaldino/personal-finance/pull/54)

## Release - v1.0.1 - 09-11-2024

- Corrected wallet id comparison [PR#51](https://github.com/silvioubaldino/personal-finance/pull/51)

## Release - v1.0.0 - 09-11-2024

- Enriched and ordered get movements, and corrected update movement [PR#48](https://github.com/silvioubaldino/personal-finance/pull/48)
- Corrected wallets recalculate and add automatic recalculate [PR#47](https://github.com/silvioubaldino/personal-finance/pull/47)

## Release 05-11-2024

- Changed wallets ui from integer to uuid [PR#44](https://github.com/silvioubaldino/personal-finance/pull/44)
- Corrected session control concurrency bug [PR#43](https://github.com/silvioubaldino/personal-finance/pull/43)
- Corrected update initial balance wallets [PR#42](https://github.com/silvioubaldino/personal-finance/pull/42)
- Corrected signal of add and update estimates [PR#41](https://github.com/silvioubaldino/personal-finance/pull/41)
- Added is income field at categories [PR#40](https://github.com/silvioubaldino/personal-finance/pull/40)
- Ordered get estimates by description [PR#39](https://github.com/silvioubaldino/personal-finance/pull/39)
- Corrected concurrency session control error [PR#38](https://github.com/silvioubaldino/personal-finance/pull/38)
- Ordered get wallets by description [PR#37](https://github.com/silvioubaldino/personal-finance/pull/37)
- Refactored balance by period [PR#36](https://github.com/silvioubaldino/personal-finance/pull/36)
- Created estimate and sub estimate update [PR#35](https://github.com/silvioubaldino/personal-finance/pull/35)
- Created estimate and sub estimate add [PR#34](https://github.com/silvioubaldino/personal-finance/pull/34)
- Created model and get estimate categories [PR#33](https://github.com/silvioubaldino/personal-finance/pull/33)
- Changed category id type [PR#32](https://github.com/silvioubaldino/personal-finance/pull/32)
- Allowed optional sub categories at movement creation [PR#31](https://github.com/silvioubaldino/personal-finance/pull/31)
- Ordered get categories [PR#30](https://github.com/silvioubaldino/personal-finance/pull/30)
- Added get default categories and subcategories [PR#29](https://github.com/silvioubaldino/personal-finance/pull/29)
- Added revert pay endpoint [PR#28](https://github.com/silvioubaldino/personal-finance/pull/28)
- Added pay endpoint [PR#27](https://github.com/silvioubaldino/personal-finance/pull/27)
- Added new endpoint to simply add movement [PR#26](https://github.com/silvioubaldino/personal-finance/pull/26)

## Release 09-12-2023

- Added recalculate balance endpoint [PR#11](https://github.com/silvioubaldino/personal-finance/pull/11)
- Added initial balance and initial date [PR#10](https://github.com/silvioubaldino/personal-finance/pull/10)
- Added sub category entity [PR#09](https://github.com/silvioubaldino/personal-finance/pull/9)
- Added rule to update wallet balance when update a movement [PR#08](https://github.com/silvioubaldino/personal-finance/pull/8)
- Started changelog [PR#00]()

## Example 06-08-2021
