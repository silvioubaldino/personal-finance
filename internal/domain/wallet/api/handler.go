package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
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

func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		wallets, err := h.srv.FindAll(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		outputWallet := make([]model.WalletOutput, len(wallets))
		for i, wallet := range wallets {
			outputWallet[i] = model.ToWalletOutput(wallet)
		}
		c.JSON(http.StatusOK, outputWallet)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		wallet, err := h.srv.FindByID(c.Request.Context(), int(id), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToWalletOutput(wallet))
	}
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		var wallet model.Wallet
		err = c.ShouldBindJSON(&wallet)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedWallet, err := h.srv.Add(context.Background(), wallet, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, model.ToWalletOutput(savedWallet))
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
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

		updatedCateg, err := h.srv.Update(context.Background(), int(id), wallet, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, model.ToWalletOutput(updatedCateg))
	}
}

func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
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
