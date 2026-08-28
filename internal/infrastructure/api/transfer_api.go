package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	TransferUseCase interface {
		Execute(ctx context.Context, input usecase.TransferInput) (usecase.TransferOutput, error)
	}

	UpdateTransferUseCase interface {
		Execute(ctx context.Context, input usecase.UpdateTransferInput) (usecase.TransferOutput, error)
	}

	DeleteTransferUseCase interface {
		Execute(ctx context.Context, pairID uuid.UUID) error
	}

	PayTransferUseCase interface {
		Execute(ctx context.Context, pairID uuid.UUID) (usecase.TransferOutput, error)
		Revert(ctx context.Context, pairID uuid.UUID) (usecase.TransferOutput, error)
	}

	TransferHandler struct {
		usecase       TransferUseCase
		updateUsecase UpdateTransferUseCase
		deleteUsecase DeleteTransferUseCase
		payUsecase    PayTransferUseCase
	}

	TransferRequest struct {
		OriginWalletID      uuid.UUID `json:"origin_wallet_id" binding:"required"`
		DestinationWalletID uuid.UUID `json:"destination_wallet_id" binding:"required"`
		Amount              float64   `json:"amount" binding:"required,gt=0"`
		Date                string    `json:"date" binding:"required"`
		Description         string    `json:"description"`
		IsPaid              bool      `json:"is_paid"`
	}

	UpdateTransferRequest struct {
		OriginWalletID      uuid.UUID `json:"origin_wallet_id" binding:"required"`
		DestinationWalletID uuid.UUID `json:"destination_wallet_id" binding:"required"`
		Amount              float64   `json:"amount" binding:"required,gt=0"`
		Date                string    `json:"date" binding:"required"`
		Description         string    `json:"description"`
	}

	TransferResponse struct {
		PairID              uuid.UUID             `json:"pair_id"`
		OriginMovement      output.MovementOutput `json:"origin_movement"`
		DestinationMovement output.MovementOutput `json:"destination_movement"`
	}
)

func NewTransferHandlers(
	r *gin.Engine,
	srv TransferUseCase,
	updateSrv UpdateTransferUseCase,
	deleteSrv DeleteTransferUseCase,
	paySrv PayTransferUseCase,
) {
	handler := TransferHandler{
		usecase:       srv,
		updateUsecase: updateSrv,
		deleteUsecase: deleteSrv,
		payUsecase:    paySrv,
	}

	transferGroup := r.Group("/v2/transfers")

	transferGroup.POST("/", handler.Add())
	transferGroup.PUT("/:pair_id", handler.Update())
	transferGroup.DELETE("/:pair_id", handler.Delete())
	transferGroup.POST("/:pair_id/pay", handler.Pay())
	transferGroup.POST("/:pair_id/revert-pay", handler.RevertPay())
}

func toTransferResponse(result usecase.TransferOutput) TransferResponse {
	return TransferResponse{
		PairID:              result.PairID,
		OriginMovement:      *output.ToMovementOutput(result.OriginMovement),
		DestinationMovement: *output.ToMovementOutput(result.DestinationMovement),
	}
}

func (h TransferHandler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req TransferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid date format, expected YYYY-MM-DD"))
			return
		}

		input := usecase.TransferInput{
			OriginWalletID:      req.OriginWalletID,
			DestinationWalletID: req.DestinationWalletID,
			Amount:              req.Amount,
			Date:                date,
			Description:         req.Description,
			IsPaid:              req.IsPaid,
		}

		result, err := h.usecase.Execute(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, toTransferResponse(result))
	}
}

func (h TransferHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		pairID, err := uuid.Parse(c.Param("pair_id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "pair_id must be valid"))
			return
		}

		var req UpdateTransferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid date format, expected YYYY-MM-DD"))
			return
		}

		input := usecase.UpdateTransferInput{
			PairID:              pairID,
			OriginWalletID:      req.OriginWalletID,
			DestinationWalletID: req.DestinationWalletID,
			Amount:              req.Amount,
			Date:                date,
			Description:         req.Description,
		}

		result, err := h.updateUsecase.Execute(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toTransferResponse(result))
	}
}

func (h TransferHandler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		pairID, err := uuid.Parse(c.Param("pair_id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "pair_id must be valid"))
			return
		}

		if err := h.deleteUsecase.Execute(ctx, pairID); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (h TransferHandler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		pairID, err := uuid.Parse(c.Param("pair_id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "pair_id must be valid"))
			return
		}

		result, err := h.payUsecase.Execute(ctx, pairID)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toTransferResponse(result))
	}
}

func (h TransferHandler) RevertPay() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		pairID, err := uuid.Parse(c.Param("pair_id"))
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "pair_id must be valid"))
			return
		}

		result, err := h.payUsecase.Revert(ctx, pairID)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toTransferResponse(result))
	}
}
