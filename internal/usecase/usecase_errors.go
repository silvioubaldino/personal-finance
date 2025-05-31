package usecase

import "errors"

var (
	ErrDateRequired        = errors.New("date must be informed")
	ErrMovementAlreadyPaid = errors.New("movement already paid")
	ErrMovementNotPaid     = errors.New("movement is not paid")
	ErrInsufficientBalance = errors.New("wallet has insufficient balance")
)
