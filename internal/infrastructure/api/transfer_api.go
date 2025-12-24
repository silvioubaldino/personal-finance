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

	TransferHandler struct {
		usecase       TransferUseCase
		updateUsecase UpdateTransferUseCase
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
		PairID              uuid.UUID `json:"pair_id" binding:"required"`
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

func NewTransferHandlers(r *gin.Engine, srv TransferUseCase, updateSrv UpdateTransferUseCase) {
	handler := TransferHandler{
		usecase:       srv,
		updateUsecase: updateSrv,
	}

	transferGroup := r.Group("/v2/transfers")

	transferGroup.POST("/", handler.Add())
	transferGroup.PATCH("/:movement_id", handler.Update())
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

		response := TransferResponse{
			PairID:              result.PairID,
			OriginMovement:      *output.ToMovementOutput(result.OriginMovement),
			DestinationMovement: *output.ToMovementOutput(result.DestinationMovement),
		}

		c.JSON(http.StatusCreated, response)
	}
}

func (h TransferHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		movementIDStr := c.Param("movement_id")
		movementID, err := uuid.Parse(movementIDStr)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid movement_id format"))
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
			MovementID:          movementID,
			PairID:              req.PairID,
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

		response := TransferResponse{
			PairID:              result.PairID,
			OriginMovement:      *output.ToMovementOutput(result.OriginMovement),
			DestinationMovement: *output.ToMovementOutput(result.DestinationMovement),
		}

		c.JSON(http.StatusOK, response)
	}
}
