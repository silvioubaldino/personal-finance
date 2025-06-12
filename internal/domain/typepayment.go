package domain

type TypePayment string

var (
	TypePaymentPix   TypePayment = "pix"
	TypePaymentDebit TypePayment = "debit_card"
	TypePaymentMoney TypePayment = "money"
)
