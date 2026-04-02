package api

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type (
	SubscriptionUseCase interface {
		CreateCheckout(ctx context.Context, backURL string) (string, error)
		CancelSubscription(ctx context.Context) error
		HandleWebhook(ctx context.Context, xSignature, xRequestId string, body []byte) error
		HandleRevenueCatWebhook(ctx context.Context, authHeader string, body []byte) error
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
	webhooksGroup.POST("/revenuecat", handler.HandleRevenueCatWebhook())
}

// RegisterSubscriptionReturnRoute registers the public redirect endpoint that MP uses after checkout.
// Must be called before any global auth middleware is added to the engine.
func RegisterSubscriptionReturnRoute(r *gin.Engine) {
	deeplink := os.Getenv("MERCADOPAGO_APP_DEEPLINK")
	r.GET("/subscription/return", func(c *gin.Context) {
		if deeplink == "" {
			c.Status(http.StatusNotFound)
			return
		}
		c.Redirect(http.StatusFound, deeplink)
	})
}

type createCheckoutRequest struct {
	BackURL string `json:"back_url"`
}

func (h SubscriptionHandler) CreateCheckout() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req createCheckoutRequest
		_ = c.ShouldBindJSON(&req)

		resp, err := h.usecase.CreateCheckout(ctx, req.BackURL)
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

func (h SubscriptionHandler) HandleRevenueCatWebhook() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}
		defer c.Request.Body.Close()

		authHeader := c.GetHeader("Authorization")

		err = h.usecase.HandleRevenueCatWebhook(ctx, authHeader, body)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Status(http.StatusOK)
	}
}
