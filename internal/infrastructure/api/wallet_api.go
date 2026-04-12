package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	WalletUsecase interface {
		Add(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error)
		FindAll(ctx context.Context) ([]domain.Wallet, error)
		FindByID(ctx context.Context, id *uuid.UUID) (domain.Wallet, error)
		Update(ctx context.Context, wallet domain.Wallet) (domain.Wallet, error)
		Delete(ctx context.Context, id *uuid.UUID) error
		RecalculateBalance(ctx context.Context, walletID *uuid.UUID) error
	}

	WalletHandler struct {
		usecase WalletUsecase
	}
)

func NewWalletV2Handlers(r *gin.Engine, srv WalletUsecase) {
	handler := WalletHandler{usecase: srv}

	group := r.Group("/v2/wallets")
	group.POST("/", handler.Add())
	group.GET("/", handler.FindAll())
	group.GET("/:id", handler.FindByID())
	group.PUT("/:id", handler.Update())
	group.DELETE("/:id", handler.Delete())
	group.POST("/:id/recalculate", handler.RecalculateBalance())
}

func (h WalletHandler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var wallet domain.Wallet
		if err := c.ShouldBindJSON(&wallet); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		saved, err := h.usecase.Add(ctx, wallet)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, output.ToWalletOutput(saved))
	}
}

func (h WalletHandler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		wallets, err := h.usecase.FindAll(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		result := make([]output.WalletOutput, len(wallets))
		for i, w := range wallets {
			result[i] = output.ToWalletOutput(w)
		}

		c.JSON(http.StatusOK, result)
	}
}

func (h WalletHandler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		wallet, err := h.usecase.FindByID(ctx, &id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToWalletOutput(wallet))
	}
}

func (h WalletHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var wallet domain.Wallet
		if err := c.ShouldBindJSON(&wallet); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		wallet.ID = &id
		updated, err := h.usecase.Update(ctx, wallet)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToWalletOutput(updated))
	}
}

func (h WalletHandler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		if err := h.usecase.Delete(ctx, &id); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (h WalletHandler) RecalculateBalance() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		if err := h.usecase.RecalculateBalance(ctx, &id); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
