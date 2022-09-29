package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
)

type handler struct {
	srv service.Service
}

func NewWalletHandlers(r *gin.Engine, srv service.Service) {
	handler := handler{srv: srv}

	r.GET("/wallets", handler.FindAll())
	r.GET("/wallets/:id", handler.FindByID())
	r.POST("/wallets", handler.Add())
	r.PUT("/wallets/:id", handler.Update())
	r.DELETE("/wallets/:id", handler.Delete())
}

// FindAll godoc
// @Summary List wallets
// @Tags Wallet
// @Description list all wallets
// @Accept json
// @Produce json
// @Success 200 {object} []model.Wallet
// @Failure 404 {object} string
// @Router /wallets [get]
func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		wallets, err := h.srv.FindAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, wallets)
	}
}

// FindByID godoc
// @Summary wallet by ID
// @Tags Wallet
// @Description wallet by ID
// @Accept json
// @Produce json
// @Param id path string true "Wallet ID"
// @Success 200 {object} model.Wallet
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /wallets/:id [get]
func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		bitSize := 64
		id, err := strconv.ParseInt(idString, base, bitSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		wallet, err := h.srv.FindByID(c.Request.Context(), int(id))
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, wallet)
	}
}

// Add godoc
// @Summary Creates new wallet
// @Tags Wallet
// @Description Creates new wallet
// @Accept json
// @Produce json
// @Success 201 {object} model.Wallet
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /wallets [post]
func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		var wallet model.Wallet
		err := c.ShouldBindJSON(&wallet)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedCateg, err := h.srv.Add(context.Background(), wallet)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, savedCateg)
	}
}

// Update godoc
// @Summary Updates wallet
// @Tags Wallet
// @Description Updates existing wallet
// @Accept json
// @Produce json
// @Param id path string true "Wallet ID"
// @Success 200 {object} model.Wallet
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /wallets/:id [put]
func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		bitSize := 64
		id, err := strconv.ParseInt(idString, base, bitSize)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		var wallet model.Wallet
		err = c.ShouldBindJSON(&wallet)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedCateg, err := h.srv.Update(context.Background(), int(id), wallet)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

// Delete godoc
// @Summary Delete wallet
// @Tags Wallet
// @Description Delete wallet
// @Accept json
// @Produce json
// @Param id path string true "Wallet ID"
// @Success 204 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /wallets/:id [delete]
func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		base := 10
		bitSize := 64
		id, err := strconv.ParseInt(idString, base, bitSize)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
		err = h.srv.Delete(context.Background(), int(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}
