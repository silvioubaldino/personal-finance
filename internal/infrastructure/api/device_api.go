package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type (
	DeviceUseCase interface {
		Upsert(ctx context.Context, input usecase.DeviceInput) (domain.Device, error)
		List(ctx context.Context) ([]domain.Device, error)
		Delete(ctx context.Context, token string) error
	}

	DeviceHandler struct {
		usecase DeviceUseCase
	}

	DeviceRequest struct {
		ExpoPushToken string `json:"expo_push_token" binding:"required"`
		Platform      string `json:"platform" binding:"required"`
	}

	DeviceResponse struct {
		ID            string  `json:"id"`
		ExpoPushToken string  `json:"expo_push_token"`
		Platform      string  `json:"platform"`
		DateCreate    string  `json:"date_create"`
		DateUpdate    string  `json:"date_update"`
		LastSeenAt    *string `json:"last_seen_at,omitempty"`
	}
)

func NewDeviceHandlers(r *gin.Engine, srv DeviceUseCase) {
	handler := DeviceHandler{
		usecase: srv,
	}

	devicesGroup := r.Group("/devices")

	devicesGroup.POST("", handler.Upsert())
	devicesGroup.GET("", handler.List())
	devicesGroup.DELETE("/:token", handler.Delete())
}

func (h DeviceHandler) Upsert() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req DeviceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		input := usecase.DeviceInput{
			ExpoPushToken: req.ExpoPushToken,
			Platform:      req.Platform,
		}

		device, err := h.usecase.Upsert(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, toDeviceResponse(device))
	}
}

func (h DeviceHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		devices, err := h.usecase.List(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		responses := make([]DeviceResponse, len(devices))
		for i, device := range devices {
			responses[i] = toDeviceResponse(device)
		}

		c.JSON(http.StatusOK, responses)
	}
}

func (h DeviceHandler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		token := c.Param("token")

		if err := h.usecase.Delete(ctx, token); err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func toDeviceResponse(device domain.Device) DeviceResponse {
	resp := DeviceResponse{
		ID:            device.ID.String(),
		ExpoPushToken: device.ExpoPushToken,
		Platform:      string(device.Platform),
		DateCreate:    device.DateCreate.Format("2006-01-02T15:04:05Z07:00"),
		DateUpdate:    device.DateUpdate.Format("2006-01-02T15:04:05Z07:00"),
	}

	if device.LastSeenAt != nil {
		lastSeen := device.LastSeenAt.Format("2006-01-02T15:04:05Z07:00")
		resp.LastSeenAt = &lastSeen
	}

	return resp
}
