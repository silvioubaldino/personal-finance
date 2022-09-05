package model

import (
	"errors"
	"net/http"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("resource not found")
)

type BusinessError struct {
	Msg      string `json:"msg,omitempty"`
	HTTPCode int    `json:"httpCode,omitempty"`
	Cause    error  `json:"cause,omitempty"`
}

func (e BusinessError) Error() string {
	return e.Msg
}

func (e BusinessError) Unwrap() error {
	return e.Cause
}

func BuildBusinessError(msg string, httpCode int, cause error) error {
	return BusinessError{
		Msg:      msg,
		HTTPCode: httpCode,
		Cause:    cause,
	}
}

func BuildErrNotfound(msg string) error {
	return BusinessError{
		Msg:      msg,
		HTTPCode: http.StatusNotFound,
		Cause:    ErrNotFound,
	}
}

func BuildErrValidation(msg string) error {
	return BusinessError{
		Msg:      msg,
		HTTPCode: http.StatusBadRequest,
		Cause:    ErrValidation,
	}
}
