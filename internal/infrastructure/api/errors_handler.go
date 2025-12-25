package api

import (
	"context"
	"errors"
	"net/http"

	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/usecase"

	"personal-finance/internal/domain"
	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
)

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

func HandleErr(c *gin.Context, ctx context.Context, err error) {
	log.ErrorContext(ctx, "error handled", log.Err(err))

	responseErr := toAPIError(err)
	c.JSON(responseErr.Error.Code, responseErr)
}

func toAPIError(err error) errorResponse {
	switch {
	// Not Found errors
	case domain.Is(err, domain.ErrNotFound),
		domain.Is(err, repository.ErrMovementNotFound),
		domain.Is(err, repository.ErrRecurrentMovementNotFound),
		domain.Is(err, repository.ErrWalletNotFound),
		domain.Is(err, repository.ErrCategoryNotFound),
		domain.Is(err, repository.ErrSubCategoryNotFound),
		errors.Is(err, usecase.ErrMovementNotFound),
		errors.Is(err, usecase.ErrRecurrentNotFound),
		errors.Is(err, usecase.ErrInvoiceNotFound):
		return newErrorResponse(http.StatusNotFound, "Resource not found")

	// Bad Request errors (invalid input)
	case domain.Is(err, domain.ErrInvalidInput),
		domain.Is(err, repository.ErrInvalidMovementData),
		domain.Is(err, repository.ErrInvalidRecurrentMovementData),
		domain.Is(err, repository.ErrInvalidWalletData),
		errors.Is(err, usecase.ErrDateRequired),
		errors.Is(err, usecase.ErrInvalidClosingDay),
		errors.Is(err, usecase.ErrInvalidDueDay),
		errors.Is(err, usecase.ErrInvalidCreditLimit),
		errors.Is(err, usecase.ErrInvalidPaymentAmount),
		errors.Is(err, usecase.ErrInvalidTransferAmount),
		errors.Is(err, usecase.ErrSameWalletTransfer),
		errors.Is(err, usecase.ErrUnsupportedMovementTypeV2),
		errors.Is(err, usecase.ErrRecurrentCreditCardNotSupported):
		return newErrorResponse(http.StatusBadRequest, "Invalid data provided")

	// Unauthorized
	case domain.Is(err, domain.ErrUnauthorized):
		return newErrorResponse(http.StatusUnauthorized, "Authentication required")

	// Unprocessable Entity (business rule violations)
	case domain.Is(err, domain.ErrWalletInsufficient),
		errors.Is(err, usecase.ErrInsufficientBalance),
		errors.Is(err, usecase.ErrInsufficientCreditLimit):
		return newErrorResponse(http.StatusUnprocessableEntity, "Insufficient balance or credit limit")

	// Conflict errors (state conflicts)
	case domain.Is(err, domain.ErrConflict),
		domain.Is(err, repository.ErrDuplicateMovement),
		domain.Is(err, repository.ErrDuplicateRecurrentMovement),
		domain.Is(err, repository.ErrDuplicateWallet),
		errors.Is(err, usecase.ErrMovementAlreadyPaid),
		errors.Is(err, usecase.ErrMovementNotPaid),
		errors.Is(err, usecase.ErrCreditMovementShouldNotBePaid),
		errors.Is(err, usecase.ErrInvoiceAlreadyPaid),
		errors.Is(err, usecase.ErrInvoiceNotPaid),
		errors.Is(err, usecase.ErrInvoiceCannotModify),
		errors.Is(err, usecase.ErrCreditCardPay):
		return newErrorResponse(http.StatusConflict, "Operation not allowed due to resource state")

	default:
		return newErrorResponse(http.StatusInternalServerError, "Internal server error")
	}
}

func newErrorResponse(code int, message string) errorResponse {
	return errorResponse{
		Error: apiError{
			Code:    code,
			Message: message,
		},
	}
}
