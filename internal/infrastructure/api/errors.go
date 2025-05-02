package api

import (
	"context"
	"net/http"
	"personal-finance/internal/infrastructure/repository"

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

	apiErr := toAPIError(err)
	c.JSON(apiErr.Code, newErrorResponse(apiErr))
}

func toAPIError(err error) apiError {
	switch {
	case domain.Is(err, domain.ErrNotFound),
		domain.Is(err, repository.ErrMovementNotFound),
		domain.Is(err, repository.ErrRecurrentMovementNotFound),
		domain.Is(err, repository.ErrWalletNotFound),
		domain.Is(err, repository.ErrCategoryNotFound),
		domain.Is(err, repository.ErrSubCategoryNotFound):
		return newAPIError(http.StatusNotFound, "Resource not found")

	case domain.Is(err, domain.ErrInvalidInput),
		domain.Is(err, repository.ErrInvalidMovementData),
		domain.Is(err, repository.ErrInvalidRecurrentMovementData),
		domain.Is(err, repository.ErrInvalidWalletData):
		return newAPIError(http.StatusBadRequest, "Invalid data provided")

	case domain.Is(err, domain.ErrUnauthorized):
		return newAPIError(http.StatusUnauthorized, "Authentication required")

	case domain.Is(err, domain.ErrWalletInsufficient):
		return newAPIError(http.StatusUnprocessableEntity, "Insufficient wallet balance")

	case domain.Is(err, domain.ErrConflict),
		domain.Is(err, repository.ErrDuplicateMovement),
		domain.Is(err, repository.ErrDuplicateRecurrentMovement),
		domain.Is(err, repository.ErrDuplicateWallet):
		return newAPIError(http.StatusConflict, "Resource conflict")

	default:
		return newAPIError(http.StatusInternalServerError, "Internal server error")
	}
}

func newAPIError(code int, message string) apiError {
	return apiError{
		Code:    code,
		Message: message,
	}
}

func newErrorResponse(apiErr apiError) errorResponse {
	return errorResponse{
		Error: apiErr,
	}
}
