package repository

import "errors"

var (
	// movement

	ErrMovementNotFound    = errors.New("movement not found in repository")
	ErrDuplicateMovement   = errors.New("movement already exists")
	ErrInvalidMovementData = errors.New("invalid movement data")

	// recurrent movement

	ErrRecurrentMovementNotFound    = errors.New("recurrent movement not found in repository")
	ErrDuplicateRecurrentMovement   = errors.New("recurrent movement already exists")
	ErrInvalidRecurrentMovementData = errors.New("invalid recurrent movement data")

	// wallet

	ErrWalletNotFound    = errors.New("wallet not found in repository")
	ErrDuplicateWallet   = errors.New("wallet already exists")
	ErrInvalidWalletData = errors.New("invalid wallet data")

	// category

	ErrCategoryNotFound    = errors.New("category not found in repository")
	ErrSubCategoryNotFound = errors.New("subcategory not found in repository")

	ErrDatabaseError = errors.New("database error")
)
