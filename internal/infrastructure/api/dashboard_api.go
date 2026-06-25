package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"

	"github.com/gin-gonic/gin"
)

type (
	DashboardUsecase interface {
		CalculateSummary(ctx context.Context, period domain.Period) (domain.DashboardSummary, error)
	}

	DashboardHandler struct {
		usecase DashboardUsecase
	}
)

func NewDashboardV2Handlers(r *gin.Engine, srv DashboardUsecase) {
	handler := DashboardHandler{usecase: srv}
	r.GET("/v2/dashboard/summary", handler.GetSummary())
}

func (h DashboardHandler) GetSummary() gin.HandlerFunc {
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

		summary, err := h.usecase.CalculateSummary(ctx, period)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}
