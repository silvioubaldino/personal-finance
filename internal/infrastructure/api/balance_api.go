package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
)

type (
	BalanceUsecase interface {
		CalculateBalance(ctx context.Context, period domain.Period) (domain.Balance, error)
	}

	BalanceHandler struct {
		usecase BalanceUsecase
	}
)

func NewBalanceV2Handlers(r *gin.Engine, srv BalanceUsecase) {
	handler := BalanceHandler{usecase: srv}
	r.GET("/v2/balance/estimate/period", handler.FindEstimateByPeriod())
}

func (h BalanceHandler) FindEstimateByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var period domain.Period
		var err error

		if fromString := c.Query("from"); fromString != "" {
			period.From, err = time.Parse("2006-01-02", fromString)
			if err != nil {
				HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid from date format"))
				return
			}
		}

		if toString := c.Query("to"); toString != "" {
			period.To, err = time.Parse("2006-01-02", toString)
			if err != nil {
				HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid to date format"))
				return
			}
		}

		balance, err := h.usecase.CalculateBalance(ctx, period)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToBalanceOutput(balance))
	}
}
