package domain

type TypePayment string

var (
	TypePaymentPix            TypePayment = "pix"
	TypePaymentDebit          TypePayment = "debit_card"
	TypePaymentMoney          TypePayment = "money"
	TypePaymentCreditCard     TypePayment = "credit_card"
	TypePaymentInvoicePayment TypePayment = "invoice_payment"
)
