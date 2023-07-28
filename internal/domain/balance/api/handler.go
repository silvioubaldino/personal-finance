package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/balance/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

type handler struct {
	service service.Balance
}

func NewBalanceHandlers(r *gin.Engine, service service.Balance) {
	handler := handler{
		service: service,
	}

	r.GET("/balance/estimate/period", handler.FindEstimateByPeriod())
}

func (h handler) FindEstimateByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		var period model.Period
		if fromString := c.Query("from"); fromString != "" {
			period.From, err = time.Parse("2006-01-02", fromString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if toString := c.Query("to"); toString != "" {
			period.To, err = time.Parse("2006-01-02", toString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}

		err = period.Validate()
		if err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("period invalid: %s", err.Error()))
			return
		}

		balance, err := h.service.FindByPeriod(period, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToBalanceOutput(balance))
	}
}
