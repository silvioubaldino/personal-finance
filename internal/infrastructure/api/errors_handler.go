package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/usecase"
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
	case domain.Is(err, domain.ErrNotFound),
		domain.Is(err, repository.ErrMovementNotFound),
		domain.Is(err, repository.ErrRecurrentMovementNotFound),
		domain.Is(err, repository.ErrWalletNotFound),
		domain.Is(err, repository.ErrCategoryNotFound),
		domain.Is(err, repository.ErrSubCategoryNotFound),
		domain.Is(err, repository.ErrDeviceNotFound),
		domain.Is(err, usecase.ErrSubscriptionPlanNotFound):
		return newErrorResponse(http.StatusNotFound, "Resource not found")

	case domain.Is(err, domain.ErrInvalidInput),
		domain.Is(err, repository.ErrInvalidMovementData),
		domain.Is(err, repository.ErrInvalidRecurrentMovementData),
		domain.Is(err, repository.ErrInvalidWalletData),
		domain.Is(err, usecase.ErrEmptyToken),
		domain.Is(err, usecase.ErrInvalidPlatform),
		domain.Is(err, usecase.ErrInvalidPlan),
		domain.Is(err, usecase.ErrInvalidRole),
		domain.Is(err, usecase.ErrInvalidWebhookSignature),
		domain.Is(err, usecase.ErrRevenueCatWebhook):
		return newErrorResponse(http.StatusBadRequest, "Invalid data provided")

	case domain.Is(err, usecase.ErrInvalidFrequencyType):
		return newErrorResponse(http.StatusBadRequest, err.Error())

	case domain.Is(err, domain.ErrUnauthorized),
		domain.Is(err, usecase.ErrUnauthorized):
		return newErrorResponse(http.StatusUnauthorized, "Authentication required")

	case domain.Is(err, usecase.ErrForbidden),
		domain.Is(err, usecase.ErrWalletLimitReached),
		domain.Is(err, usecase.ErrCreditCardLimitReached),
		domain.Is(err, usecase.ErrMovementLimitReached),
		domain.Is(err, usecase.ErrRecurrenceLimitReached):
		return newErrorResponse(http.StatusForbidden, err.Error())

	case domain.Is(err, domain.ErrWalletInsufficient):
		return newErrorResponse(http.StatusUnprocessableEntity, "Insufficient wallet balance")

	case domain.Is(err, domain.ErrConflict),
		domain.Is(err, repository.ErrDuplicateMovement),
		domain.Is(err, repository.ErrDuplicateRecurrentMovement),
		domain.Is(err, repository.ErrDuplicateWallet):
		return newErrorResponse(http.StatusConflict, "Resource conflict")

	case domain.Is(err, domain.ErrAgentMemoryCapExceeded):
		return newErrorResponse(http.StatusUnprocessableEntity, "Memory limit reached. Delete stale memories first.")

	case domain.Is(err, domain.ErrAgentPIIDetected):
		return newErrorResponse(http.StatusBadRequest, "Content contains personally identifiable information")

	case domain.Is(err, domain.ErrAgentInvalidMemoryType):
		return newErrorResponse(http.StatusBadRequest, "Invalid memory type")

	case domain.Is(err, domain.ErrAgentMemoryNotFound):
		return newErrorResponse(http.StatusNotFound, "Agent memory not found")

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
