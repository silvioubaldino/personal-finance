package usecase

import "errors"

var (
	ErrDateRequired        = errors.New("date must be informed")
	ErrMovementAlreadyPaid = errors.New("movement already paid")
	ErrMovementNotPaid     = errors.New("movement is not paid")
	ErrInsufficientBalance = errors.New("wallet has insufficient balance")
	ErrInvoiceAlreadyPaid  = errors.New("invoice already paid")
	ErrInvoiceCannotModify = errors.New("cannot modify movements for paid invoice")
	ErrInvoiceNotFound     = errors.New("invoice not found")
	ErrInvalidClosingDay   = errors.New("closing day must be between 1 and 31")
	ErrInvalidDueDay       = errors.New("due day must be between 1 and 31")
	ErrInvalidCreditLimit  = errors.New("credit limit must be positive")
)
