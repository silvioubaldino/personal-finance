
# Changelog - personal-finance

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

- Improved logs and metrics [PR#205](https://github.com/silvioubaldino/personal-finance/pull/205)

## Release - v1.18.0 - 21-06-2026

- Added metrics and traces [PR#203](https://github.com/silvioubaldino/personal-finance/pull/203)

## Release - v1.17.0 - 11-06-2026

- Added import statement pdf password validation [PR#201](https://github.com/silvioubaldino/personal-finance/pull/201)

## Release - v1.16.0 - 03-06-2026

- Migrated subscriptions to Stripe [PR#199](https://github.com/silvioubaldino/personal-finance/pull/199)

## Release - v1.15.4 - 29-05-2026

- Fixed MP subscription missmatch [PR#197](https://github.com/silvioubaldino/personal-finance/pull/197)

## Release - v1.15.3 - 26-05-2026

- Fix plan id for RC webhooks [PR#194](https://github.com/silvioubaldino/personal-finance/pull/194)

## Release - v1.15.2 - 26-05-2026

- Rolledback remove MP optional payer email [PR#192](https://github.com/silvioubaldino/personal-finance/pull/192)
- Allowed retry coupon checkout when coupon is pending [PR#191](https://github.com/silvioubaldino/personal-finance/pull/191)

## Release - v1.15.1 - 26-05-2026

- Removed MP optional payer email [PR#190](https://github.com/silvioubaldino/personal-finance/pull/190)

## Release - v1.15.0 - 26-05-2026
- Fixed 500 on credit card movement when default wallet is nil [PR#188](https://github.com/silvioubaldino/personal-finance/pull/188)
- Created coupon entity [PR#187](https://github.com/silvioubaldino/personal-finance/pull/187)
- Created subscription entity [PR#186](https://github.com/silvioubaldino/personal-finance/pull/186)
- Created user entity [PR#185](https://github.com/silvioubaldino/personal-finance/pull/185)
- Created subscription plan CRUD [PR#184](https://github.com/silvioubaldino/personal-finance/pull/184)

## Release - v1.14.0 - 02/05/2026

- Added type payment at statement and fix rc Auth key validation [PR#182](https://github.com/silvioubaldino/personal-finance/pull/182)

## Release - v1.13.0 - 13-04-2026

- Fixed edge cases [PR#181](https://github.com/silvioubaldino/personal-finance/pull/181)
- Finished v2 entities migration [PR#180](https://github.com/silvioubaldino/personal-finance/pull/180)
- Finished v2 movements migration [PR#179](https://github.com/silvioubaldino/personal-finance/pull/179)
- Added automatic classification [PR#178](https://github.com/silvioubaldino/personal-finance/pull/178)
- Added agent AI tools [PR#177](https://github.com/silvioubaldino/personal-finance/pull/177)
- Added import statement receive recurrent match [PR#176](https://github.com/silvioubaldino/personal-finance/pull/176)
- Added RevenueCat support [PR#175](https://github.com/silvioubaldino/personal-finance/pull/175)

## Release - v1.12.0 - 30-03-2026

- Added wallet plan validation [PR#173](https://github.com/silvioubaldino/personal-finance/pull/173)
- Fixed dedup issue [PR#172](https://github.com/silvioubaldino/personal-finance/pull/172)
- Created statement import structure [PR#170](https://github.com/silvioubaldino/personal-finance/pull/170)
- Fixed cancel subscription [PR#169](https://github.com/silvioubaldino/personal-finance/pull/169)
- Fixed chat title [PR#168](https://github.com/silvioubaldino/personal-finance/pull/168)
- Added ADK and model integration [PR#167](https://github.com/silvioubaldino/personal-finance/pull/167)
- Added AI agent structure [PR#166](https://github.com/silvioubaldino/personal-finance/pull/166)
- Added payment gateway structure [PR#164](https://github.com/silvioubaldino/personal-finance/pull/164)
- Created authorization structure [PR#163](https://github.com/silvioubaldino/personal-finance/pull/163)
- Configured gin logger [PR#162](https://github.com/silvioubaldino/personal-finance/pull/162)

## Release - v1.11.0 - 26-01-2026

- Added gorm loglevel from env[PR#160](https://github.com/silvioubaldino/personal-finance/pull/160)

## Release - v1.11.0 - 26-01-2026

- Added gorm loglevel from env[160](https://github.com/silvioubaldino/personal-finance/pull/160)

## Release - v1.10.0 - 21-01-2026

- Added device CRUD and push notifications [PR#156](https://github.com/silvioubaldino/personal-finance/pull/156)

## Release - v1.9.0 - 17-01-2026

- Added credit remainder subcategory [PR#155](https://github.com/silvioubaldino/personal-finance/pull/155)

## Release - v1.8.0 - 12-01-2026

- Added credit card color [PR#154](https://github.com/silvioubaldino/personal-finance/pull/154)
- Returned real invoice payment movement as a source of truth [PR#153](https://github.com/silvioubaldino/personal-finance/pull/153)
- Added export and delete all data [PR#152](https://github.com/silvioubaldino/personal-finance/pull/152)
- Added user consents CRUD [PR#151](https://github.com/silvioubaldino/personal-finance/pull/151)
- Added category is_income at recurrent find [PR#150](https://github.com/silvioubaldino/personal-finance/pull/150)
- Added invoice recalculation endpoint [PR#149](https://github.com/silvioubaldino/personal-finance/pull/149)
- Created category color field [PR#148](https://github.com/silvioubaldino/personal-finance/pull/148)

## Release - v1.7.0 - 30-12-2025

- Improved user preferences flow [PR#145](https://github.com/silvioubaldino/personal-finance/pull/145)
- Added user preferences [PR#144](https://github.com/silvioubaldino/personal-finance/pull/144)
- Added internal transfer [PR#143](https://github.com/silvioubaldino/personal-finance/pull/143)
- Fixed update wallet in revert invoice payment [PR#142](https://github.com/silvioubaldino/personal-finance/pull/142)

## Release - v1.6.0 - 10-12-2025 

- Added pay credit movements when paying invoice [PR#139](https://github.com/silvioubaldino/personal-finance/pull/139)

## Release - v1.5.1 - 10-12-2025

- Fixed revert pay invoice [PR#136](https://github.com/silvioubaldino/personal-finance/pull/136)

## Release - v1.5.0 - 08-12-2025

- Added detailed invoice at movements find by period endpoint [PR#133](https://github.com/silvioubaldino/personal-finance/pull/133)
- Added credit card limit validation[PR#132](https://github.com/silvioubaldino/personal-finance/pull/132)
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
