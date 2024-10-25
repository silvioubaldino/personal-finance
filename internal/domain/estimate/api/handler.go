package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/estimate/service"
	"personal-finance/internal/plataform/authentication"
)

type handler struct {
	service service.Service
}

func NewBalanceHandlers(r *gin.Engine, service service.Service) {
	handler := handler{
		service: service,
	}
	r.GET("/estimate", handler.FindEstimateByMonth())
}

func (h handler) FindEstimateByMonth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		var month, year int64
		if monthString := c.Query("month"); monthString != "" {
			month, err = strconv.ParseInt(monthString, 10, 64)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if yearString := c.Query("year"); yearString != "" {
			year, err = strconv.ParseInt(yearString, 10, 64)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}

		byMonth, err := h.service.FindByMonth(c.Request.Context(), int(month), int(year), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, byMonth)
		return
	}
}
