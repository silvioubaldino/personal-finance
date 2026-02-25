package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	SubscriptionUseCase interface {
		CreateCheckout(ctx context.Context) (string, error)
		CancelSubscription(ctx context.Context) error
		HandleWebhook(ctx context.Context, xSignature, xRequestId string, body []byte) error
	}

	SubscriptionHandler struct {
		usecase SubscriptionUseCase
	}
)

func NewSubscriptionHandlers(r *gin.Engine, srv SubscriptionUseCase, auth gin.HandlerFunc) {
	handler := SubscriptionHandler{
		usecase: srv,
	}

	// Authenticated group
	meGroup := r.Group("/me")
	meGroup.Use(auth)
	meGroup.POST("/subscription/checkout", handler.CreateCheckout())
	meGroup.POST("/subscription/cancel", handler.CancelSubscription())

	// Public Webhooks group
	webhooksGroup := r.Group("/webhooks")
	webhooksGroup.POST("/mercadopago", handler.HandleWebhook())
}

func (h SubscriptionHandler) CreateCheckout() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		resp, err := h.usecase.CreateCheckout(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func (h SubscriptionHandler) CancelSubscription() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		err := h.usecase.CancelSubscription(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusOK)
	}
}

func (h SubscriptionHandler) HandleWebhook() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}
		defer c.Request.Body.Close()

		xSignature := c.GetHeader("x-signature")
		xRequestId := c.GetHeader("x-request-id")

		err = h.usecase.HandleWebhook(ctx, xSignature, xRequestId, body)
		if err != nil {
			// Even if it fails, we log it but usually return 200 OK to Mercado Pago
			// to avoid infinite retries unless it's a transient error.
			// But for now, we'll use HandleErr to see what's happening.
			HandleErr(c, ctx, err)
			return
		}

		// Mercado Pago requires a 200/201 to confirm receipt
		c.Status(http.StatusOK)
	}
}
