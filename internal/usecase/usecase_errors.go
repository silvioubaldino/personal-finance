package usecase

import "errors"

var (
	ErrDateRequired                  = errors.New("date must be informed")
	ErrMovementAlreadyPaid           = errors.New("movement already paid")
	ErrMovementNotPaid               = errors.New("movement is not paid")
	ErrCreditMovementShouldNotBePaid = errors.New("credit movement should not be paid")
	ErrInsufficientBalance           = errors.New("wallet has insufficient balance")
	ErrInvoiceAlreadyPaid            = errors.New("invoice already paid")
	ErrInvoiceNotPaid                = errors.New("invoice is not paid")
	ErrInvoiceCannotModify           = errors.New("cannot modify movements for paid invoice")
	ErrInvoiceNotFound               = errors.New("invoice not found")
	ErrInvalidClosingDay             = errors.New("closing day must be between 1 and 31")
	ErrInvalidDueDay                 = errors.New("due day must be between 1 and 31")
	ErrInvalidCreditLimit            = errors.New("credit limit must be positive")
	ErrCreditCardPay                 = errors.New("credit card should`n be paid")
	ErrUnsupportedMovementTypeV2     = errors.New("unsupported movement type for V2 endpoint")
)
