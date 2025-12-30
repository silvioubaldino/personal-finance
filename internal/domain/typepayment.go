package domain

type TypePayment string

var (
	TypePaymentPix              TypePayment = "pix"
	TypePaymentDebit            TypePayment = "debit_card"
	TypePaymentMoney            TypePayment = "money"
	TypePaymentCreditCard       TypePayment = "credit_card"
	TypePaymentInvoicePayment   TypePayment = "invoice_payment"
	TypePaymentInvoiceRemainder TypePayment = "invoice_remainder"
	TypePaymentInternalTransfer TypePayment = "internal_transfer"
)

const (
	InternalTransferOutCategoryID = "c1a2b3c4-d5e6-f7a8-b9c0-d1e2f3a4b5c6"
	InternalTransferInCategoryID  = "c2b3c4d5-e6f7-a8b9-c0d1-e2f3a4b5c6d7"
)
