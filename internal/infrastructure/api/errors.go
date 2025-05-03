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
		domain.Is(err, repository.ErrSubCategoryNotFound):
		return newErrorResponse(http.StatusNotFound, "Resource not found")

	case domain.Is(err, domain.ErrInvalidInput),
		domain.Is(err, repository.ErrInvalidMovementData),
		domain.Is(err, repository.ErrInvalidRecurrentMovementData),
		domain.Is(err, repository.ErrInvalidWalletData):
		return newErrorResponse(http.StatusBadRequest, "Invalid data provided")

	case domain.Is(err, domain.ErrUnauthorized):
		return newErrorResponse(http.StatusUnauthorized, "Authentication required")

	case domain.Is(err, domain.ErrWalletInsufficient):
		return newErrorResponse(http.StatusUnprocessableEntity, "Insufficient wallet balance")

	case domain.Is(err, domain.ErrConflict),
		domain.Is(err, repository.ErrDuplicateMovement),
		domain.Is(err, repository.ErrDuplicateRecurrentMovement),
		domain.Is(err, repository.ErrDuplicateWallet):
		return newErrorResponse(http.StatusConflict, "Resource conflict")

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
