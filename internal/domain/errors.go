package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrInternalError      = errors.New("internal system error")
	ErrConflict           = errors.New("resource conflict")
	ErrWalletInsufficient = errors.New("insufficient wallet balance")
)

func WrapInvalidInput(err error, context string) error {
	return fmt.Errorf("%s: %w: %s", context, ErrInvalidInput, err.Error())
}

func WrapNotFound(err error, resource string) error {
	return fmt.Errorf("%s: %w: %s", resource, ErrNotFound, err.Error())
}

func WrapUnauthorized(err error, context string) error {
	return fmt.Errorf("%s: %w: %s", context, ErrUnauthorized, err.Error())
}

func WrapInternalError(err error, context string) error {
	return fmt.Errorf("%s: %w: %s", context, ErrInternalError, err.Error())
}

func WrapConflict(err error, context string) error {
	return fmt.Errorf("%s: %w: %s", context, ErrConflict, err.Error())
}

func WrapWalletInsufficient(err error, context string) error {
	return fmt.Errorf("%s: %w: %s", context, ErrWalletInsufficient, err.Error())
}

func New(text string) error {
	return errors.New(text)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}
