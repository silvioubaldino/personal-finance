package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"personal-finance/internal/domain/estimate"
	"personal-finance/internal/domain/estimate/service"
	"personal-finance/internal/model"
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
	r.POST("/estimate", handler.AddEstimate())
	r.POST("/sub-estimate", handler.AddSubEstimate())
	r.PUT("/estimate/:id", handler.UpdateEstimateAmount())
	r.PUT("/sub-estimate/:id", handler.UpdateSubEstimateAmount())
}

func (h handler) FindEstimateByMonth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
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

func (h handler) AddEstimate() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}

		var estimateCategories model.EstimateCategories
		if err := c.BindJSON(&estimateCategories); err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		addedEstimate, err := h.service.AddEstimate(c.Request.Context(), estimateCategories, userID)
		if err != nil {
			if errors.Is(err, estimate.ErrMonthCategoryEstimateExists) {
				c.JSON(http.StatusConflict, err.Error())
				return
			}
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, addedEstimate)
		return
	}

}

func (h handler) AddSubEstimate() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}

		var subEstimate model.EstimateSubCategories
		if err := c.BindJSON(&subEstimate); err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		addedSubEstimate, err := h.service.AddSubEstimate(c.Request.Context(), subEstimate, userID)
		if err != nil {
			if errors.Is(err, estimate.ErrMonthSubCategoryEstimateExists) {
				c.JSON(http.StatusConflict, err.Error())
				return
			}
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, addedSubEstimate)
		return
	}

}

func (h handler) UpdateEstimateAmount() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}

		idString := c.Param("id")
		id, err := uuid.Parse(idString)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		var input model.EstimateCategories
		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedEstimate, err := h.service.UpdateEstimateAmount(c.Request.Context(), &id, input.Amount, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, updatedEstimate)
		return
	}
}

func (h handler) UpdateSubEstimateAmount() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}

		idString := c.Param("id")
		id, err := uuid.Parse(idString)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		var input model.EstimateSubCategories
		if err := c.BindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedSubEstimate, err := h.service.UpdateSubEstimateAmount(c.Request.Context(), &id, input.Amount, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, updatedSubEstimate)
		return
	}
}
