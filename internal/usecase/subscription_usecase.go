package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/plataform/authentication"
)

type MercadoPagoSubscriptionGateway interface {
	CreateSubscriptionURL(ctx context.Context, payerEmail, externalID string) (string, error)
	GetSubscription(ctx context.Context, id string) (gateway.MPSubscription, error)
	CancelSubscription(ctx context.Context, id string) error
}

type (
	FirebaseSubscriptionGateway interface {
		SetUserSubscription(ctx context.Context, userID string, plan authentication.Plan, mpSubscriptionID string, expiresAt int64) error
	}

	SubscriptionUseCase interface {
		CreateCheckout(ctx context.Context) (CheckoutResponse, error)
		CancelSubscription(ctx context.Context) error
		HandleWebhook(ctx context.Context, xSignature, xRequestId string, body []byte) error
	}
)

type Subscription struct {
	mpGateway       MercadoPagoSubscriptionGateway
	firebaseGateway FirebaseSubscriptionGateway
	webhookSecret   string
}

func NewSubscription(mpGateway MercadoPagoSubscriptionGateway, firebaseGateway FirebaseSubscriptionGateway) *Subscription {
	return &Subscription{
		mpGateway:       mpGateway,
		firebaseGateway: firebaseGateway,
		webhookSecret:   os.Getenv("MERCADOPAGO_WEBHOOK_SECRET"),
	}
}

type CheckoutResponse struct {
	URL string `json:"checkout_url"`
}

func (s *Subscription) CreateCheckout(ctx context.Context) (string, error) {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return "", ErrUnauthorized
	}

	payerEmail := auth.Email
	if overrideEmail := os.Getenv("MERCADOPAGO_PAYER_EMAIL_OVERRIDE"); overrideEmail != "" {
		payerEmail = overrideEmail
	}

	checkoutURL, err := s.mpGateway.CreateSubscriptionURL(ctx, payerEmail, auth.UserID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMercadoPagoGateway, err)
	}

	return checkoutURL, nil
}

type WebhookEvent struct {
	Action string `json:"action"`
	Type   string `json:"type"`
	Data   struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (s *Subscription) CancelSubscription(ctx context.Context) error {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return ErrUnauthorized
	}

	if auth.MPSubscriptionID == "" {
		return fmt.Errorf("user does not have an active mercado pago subscription")
	}

	// Request MP to cancel
	err := s.mpGateway.CancelSubscription(ctx, auth.MPSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription in mercado pago: %w", err)
	}

	// After MP accepts the cancel, we fetch the updated subscription details to know when it ends
	subscription, err := s.mpGateway.GetSubscription(ctx, auth.MPSubscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get updated subscription: %w", err)
	}

	// Calculate graceful downgrade
	var expiresAt int64
	if subscription.NextPaymentDate != "" {
		parsedDate, err := time.Parse("2006-01-02T15:04:05.000-07:00", subscription.NextPaymentDate)
		if err == nil {
			expiresAt = parsedDate.Unix()
		}
	}

	// We set `plan = Plus` but with an expiration date.
	// Empty MP Subscription ID means they won't be charged anymore and prevents repeat cancels.
	err = s.firebaseGateway.SetUserSubscription(ctx, auth.UserID, authentication.PlanPlus, "", expiresAt)
	if err != nil {
		return fmt.Errorf("error updating firebase subscription data on cancel: %w", err)
	}

	return nil
}

func (s *Subscription) HandleWebhook(ctx context.Context, xSignature, xRequestId string, body []byte) error {
	// Secret validation
	if s.webhookSecret != "" {
		if err := s.validateSignature(xSignature, xRequestId, body); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidWebhookSignature, err)
		}
	}

	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("error unmarshaling webhook: %w", err)
	}

	// We only process subscription pre-approval events
	if event.Type != "subscription_preapproval" && event.Type != "preapproval" {
		return nil
	}

	// Anti-fraud: Call MP API to get the real status from the source of truth
	subscription, err := s.mpGateway.GetSubscription(ctx, event.Data.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMercadoPagoGateway, err)
	}

	uid := subscription.ExternalReference
	if uid == "" {
		return fmt.Errorf("no external_reference (UID) found in subscription %s", event.Data.ID)
	}

	var mpSubID string
	var expiresAt int64
	var plan authentication.Plan

	switch subscription.Status {
	case "authorized":
		plan = authentication.PlanPlus
		mpSubID = subscription.ID
		// For authorized we do not set expiration directly via webhook
	case "cancelled", "paused":
		plan = authentication.PlanFree
		// When webhook pushes a cancelled event asynchronously
		// If `NextPaymentDate` is present and in the future, we still give them Plus until that date
		if subscription.NextPaymentDate != "" {
			parsedDate, err := time.Parse("2006-01-02T15:04:05.000-07:00", subscription.NextPaymentDate)
			if err == nil {
				if parsedDate.After(time.Now()) {
					plan = authentication.PlanPlus
					expiresAt = parsedDate.Unix()
				}
			}
		}
	default:
		// Other statuses (pending, etc.) - we don't change anything
		return nil
	}

	err = s.firebaseGateway.SetUserSubscription(ctx, uid, plan, mpSubID, expiresAt)
	if err != nil {
		return fmt.Errorf("error updating firebase subscription data: %w", err)
	}

	return nil
}

func (s *Subscription) validateSignature(xSignature, xRequestId string, body []byte) error {
	// Mercado Pago signature format: ts=...,v1=...
	parts := strings.Split(xSignature, ",")
	var ts, v1 string
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "ts" {
			ts = kv[1]
		} else if kv[0] == "v1" {
			v1 = kv[1]
		}
	}

	if ts == "" || v1 == "" {
		return fmt.Errorf("missing ts or v1 in signature")
	}

	// Data to sign: id:[data.id];request-id:[x-request-id];ts:[ts];
	// According to MP docs, for Preapprovals (Subscriptions) the format might vary,
	// but usually it's x-request-id and the payload for V2.
	// Let's use the standard V2 validation: template string = "id:DATA_ID;request-id:X_REQUEST_ID;ts:TS;"
	// However, HandleWebhook doesn't have DATA_ID yet until unmarshal.
	// MP V2 actually signs: "id:" + dataID + ";request-id:" + xRequestId + ";ts:" + ts + ";"

	var event struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &event)

	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", event.Data.ID, xRequestId, ts)

	h := hmac.New(sha256.New, []byte(s.webhookSecret))
	h.Write([]byte(manifest))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if expectedSignature != v1 {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
