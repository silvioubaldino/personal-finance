package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"

	"github.com/gin-gonic/gin"
)

type (
	DeleteAccountUseCase interface {
		DeleteUserAccount(ctx context.Context) error
	}

	DeleteAccountHandler struct {
		usecase DeleteAccountUseCase
	}

	DeleteAccountConfirmRequest struct {
		Confirm bool `json:"confirm" binding:"required"`
	}
)

func NewDeleteAccountHandlers(r *gin.Engine, srv DeleteAccountUseCase) {
	handler := DeleteAccountHandler{
		usecase: srv,
	}

	meGroup := r.Group("/me")

	meGroup.DELETE("/account", handler.DeleteAccount())
}

func (h DeleteAccountHandler) DeleteAccount() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req DeleteAccountConfirmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		if !req.Confirm {
			HandleErr(c, ctx, domain.WrapInvalidInput(domain.New("confirm must be true"), "you must explicitly confirm account deletion"))
			return
		}

		err := h.usecase.DeleteUserAccount(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Account and all associated data have been permanently deleted",
		})
	}
}
