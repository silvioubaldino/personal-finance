package usecase

import "errors"

var (
	ErrDateRequired                    = errors.New("date must be informed")
	ErrMovementAlreadyPaid             = errors.New("movement already paid")
	ErrMovementNotPaid                 = errors.New("movement is not paid")
	ErrCreditMovementShouldNotBePaid   = errors.New("credit movement should not be paid")
	ErrInsufficientBalance             = errors.New("wallet has insufficient balance")
	ErrInsufficientCreditLimit         = errors.New("credit card has insufficient credit limit")
	ErrInvoiceAlreadyPaid              = errors.New("invoice already paid")
	ErrInvoiceNotPaid                  = errors.New("invoice is not paid")
	ErrInvoiceCannotModify             = errors.New("cannot modify movements for paid invoice")
	ErrInvoiceNotFound                 = errors.New("invoice not found")
	ErrInvalidClosingDay               = errors.New("closing day must be between 1 and 31")
	ErrInvalidDueDay                   = errors.New("due day must be between 1 and 31")
	ErrInvalidCreditLimit              = errors.New("credit limit must be positive")
	ErrCreditCardPay                   = errors.New("credit card should`n be paid")
	ErrUnsupportedMovementTypeV2       = errors.New("unsupported movement type for V2 endpoint")
	ErrInvalidPaymentAmount            = errors.New("payment amount must be between invoice amount and zero")
	ErrSameWalletTransfer              = errors.New("origin and destination wallets must be different")
	ErrInvalidTransferAmount           = errors.New("transfer amount must be positive")
	ErrRecurrentCreditCardNotSupported = errors.New("recurrent movements with credit card are not supported")
	ErrMovementNotFound                = errors.New("movement not found")
	ErrRecurrentNotFound               = errors.New("recurrent movement not found")
)
