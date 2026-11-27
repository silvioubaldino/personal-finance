package domain

type TypePayment string

var (
	TypePaymentPix                   TypePayment = "pix"
	TypePaymentDebit                 TypePayment = "debit_card"
	TypePaymentMoney                 TypePayment = "money"
	TypePaymentCreditCard            TypePayment = "credit_card"
	TypePaymentTED                   TypePayment = "ted"
	TypePaymentDOC                   TypePayment = "doc"
	TypePaymentInvoicePayment        TypePayment = "invoice_payment"
	TypePaymentInvoiceAdvancePayment TypePayment = "invoice_advance_payment"
	TypePaymentInvoiceRemainder      TypePayment = "invoice_remainder"
	TypePaymentInternalTransfer      TypePayment = "internal_transfer"
)

const (
	InternalTransferOutCategoryID = "c1a2b3c4-d5e6-f7a8-b9c0-d1e2f3a4b5c6"
	InternalTransferInCategoryID  = "c2b3c4d5-e6f7-a8b9-c0d1-e2f3a4b5c6d7"
)

func (t TypePayment) IsInvoicePayment() bool {
	return t == TypePaymentInvoicePayment || t == TypePaymentInvoiceAdvancePayment
}

func InvoicePaymentTypes() []TypePayment {
	return []TypePayment{TypePaymentInvoicePayment, TypePaymentInvoiceAdvancePayment}
}

func NonRealizedTypePayments() []string {
	types := InvoicePaymentTypes()
	result := make([]string, 0, len(types)+1)
	for _, t := range types {
		result = append(result, string(t))
	}
	return append(result, string(TypePaymentInternalTransfer))
}
