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

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/plataform/authentication"
)

type MercadoPagoSubscriptionGateway interface {
	CreateSubscriptionURL(ctx context.Context, payerEmail, externalID, backURL string, plan gateway.SubscriptionPlanConfig) (string, error)
	GetSubscription(ctx context.Context, id string) (gateway.MPSubscription, error)
	CancelSubscription(ctx context.Context, id string) error
}

type (
	FirebaseSubscriptionGateway interface {
		SetUserSubscription(ctx context.Context, userID string, plan authentication.Plan, mpSubscriptionID string, subscriptionSource authentication.SubscriptionSource, expiresAt int64) error
	}

	SubscriptionPlanRepository interface {
		Create(ctx context.Context, plan domain.SubscriptionPlan) error
		FindActive(ctx context.Context) ([]domain.SubscriptionPlan, error)
		FindActiveByID(ctx context.Context, id string) (domain.SubscriptionPlan, error)
	}

	SubscriptionRepository interface {
		Upsert(ctx context.Context, sub domain.Subscription) (domain.Subscription, error)
		List(ctx context.Context, filter repository.SubscriptionListFilter) ([]domain.Subscription, error)
	}

	SubscriptionUseCase interface {
		CreateCheckout(ctx context.Context, planID, backURL string) (string, error)
		CancelSubscription(ctx context.Context) error
		HandleWebhook(ctx context.Context, xSignature, xRequestId string, body []byte) error
		HandleRevenueCatWebhook(ctx context.Context, authHeader string, body []byte) error
		GetActivePlans(ctx context.Context) ([]domain.SubscriptionPlan, error)
	}
)

type Subscription struct {
	mpGateway        MercadoPagoSubscriptionGateway
	firebaseGateway  FirebaseSubscriptionGateway
	planRepo         SubscriptionPlanRepository
	subRepo          SubscriptionRepository
	webhookSecret    string
	rcWebhookAuthKey string
}

func NewSubscription(
	mpGateway MercadoPagoSubscriptionGateway,
	firebaseGateway FirebaseSubscriptionGateway,
	planRepo SubscriptionPlanRepository,
	subRepo SubscriptionRepository,
) *Subscription {
	return &Subscription{
		mpGateway:        mpGateway,
		firebaseGateway:  firebaseGateway,
		planRepo:         planRepo,
		subRepo:          subRepo,
		webhookSecret:    os.Getenv("MERCADOPAGO_WEBHOOK_SECRET"),
		rcWebhookAuthKey: os.Getenv("REVENUECAT_WEBHOOK_AUTH_KEY"),
	}
}

type CheckoutResponse struct {
	URL string `json:"checkout_url"`
}

func (s *Subscription) CreateCheckout(ctx context.Context, planID, backURL string) (string, error) {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return "", ErrUnauthorized
	}

	plan, err := s.planRepo.FindActiveByID(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSubscriptionPlanNotFound, err)
	}

	payerEmail := auth.Email
	if overrideEmail := os.Getenv("MERCADOPAGO_PAYER_EMAIL_OVERRIDE"); overrideEmail != "" {
		payerEmail = overrideEmail
	}

	planConfig := gateway.SubscriptionPlanConfig{
		Price:         plan.Price,
		Currency:      plan.Currency,
		Frequency:     plan.Frequency,
		FrequencyType: plan.FrequencyType,
	}

	checkoutURL, err := s.mpGateway.CreateSubscriptionURL(ctx, payerEmail, auth.UserID, backURL, planConfig)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMercadoPagoGateway, err)
	}

	return checkoutURL, nil
}

func (s *Subscription) ListSubscriptions(ctx context.Context, filter repository.SubscriptionListFilter) ([]domain.Subscription, error) {
	if s.subRepo == nil {
		return []domain.Subscription{}, nil
	}
	return s.subRepo.List(ctx, filter)
}

func (s *Subscription) GetActivePlans(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	plans, err := s.planRepo.FindActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing active plans: %w", err)
	}
	return plans, nil
}

var validFrequencyTypes = map[string]bool{"months": true, "days": true}

func (s *Subscription) CreatePlan(ctx context.Context, plan domain.SubscriptionPlan) error {
	if plan.ID == "" {
		return domain.WrapInvalidInput(domain.New("id is required"), "create plan")
	}
	if plan.Name == "" {
		return domain.WrapInvalidInput(domain.New("name is required"), "create plan")
	}
	if plan.Price <= 0 {
		return domain.WrapInvalidInput(domain.New("price must be positive"), "create plan")
	}
	if plan.Frequency <= 0 {
		return domain.WrapInvalidInput(domain.New("frequency must be positive"), "create plan")
	}
	if !validFrequencyTypes[plan.FrequencyType] {
		return ErrInvalidFrequencyType
	}
	if plan.Currency == "" {
		plan.Currency = "BRL"
	}
	return s.planRepo.Create(ctx, plan)
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

	if err := s.upsertMPSubscription(ctx, auth.UserID, subscription); err != nil {
		return fmt.Errorf("error mirroring cancelled subscription to db: %w", err)
	}

	// We set `plan = Plus` but with an expiration date.
	// Empty MP Subscription ID means they won't be charged anymore and prevents repeat cancels.
	err = s.firebaseGateway.SetUserSubscription(ctx, auth.UserID, authentication.PlanPlus, "", authentication.SubscriptionSourceMP, expiresAt)
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

	// DB mirror runs before Firebase so a retry from MP refreshes both sources idempotently.
	if err := s.upsertMPSubscription(ctx, uid, subscription); err != nil {
		return fmt.Errorf("error mirroring subscription to db: %w", err)
	}

	err = s.firebaseGateway.SetUserSubscription(ctx, uid, plan, mpSubID, authentication.SubscriptionSourceMP, expiresAt)
	if err != nil {
		return fmt.Errorf("error updating firebase subscription data: %w", err)
	}

	return nil
}

// upsertMPSubscription persists the MP subscription into the local mirror.
// Idempotent via UNIQUE(source, external_id). plan_id is left empty when not derivable
// from the MP payload — backfill can enrich later.
func (s *Subscription) upsertMPSubscription(ctx context.Context, userID string, mp gateway.MPSubscription) error {
	if s.subRepo == nil {
		return nil
	}

	status := mapMPStatusToDomain(mp.Status)
	if status == "" {
		return nil
	}

	startedAt := parseMPDate(mp.DateCreated)
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	var currentPeriodEnd *time.Time
	if parsed := parseMPDate(mp.NextPaymentDate); !parsed.IsZero() {
		currentPeriodEnd = &parsed
	}

	var cancelledAt *time.Time
	if status == domain.SubscriptionStatusCancelled {
		now := time.Now()
		cancelledAt = &now
	}

	sub := domain.Subscription{
		UserID:           userID,
		Source:           domain.SubscriptionSourceMercadoPago,
		ExternalID:       mp.ID,
		Status:           status,
		CurrentPrice:     mp.AutoRecurring.TransactionAmount,
		Currency:         mp.AutoRecurring.CurrencyID,
		StartedAt:        startedAt,
		CurrentPeriodEnd: currentPeriodEnd,
		CancelledAt:      cancelledAt,
	}

	_, err := s.subRepo.Upsert(ctx, sub)
	return err
}

func mapMPStatusToDomain(mpStatus string) domain.SubscriptionStatus {
	switch mpStatus {
	case "authorized":
		return domain.SubscriptionStatusActive
	case "cancelled":
		return domain.SubscriptionStatusCancelled
	case "paused":
		return domain.SubscriptionStatusPaused
	default:
		return ""
	}
}

func parseMPDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05.000Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// RevenueCat webhook types
type RevenueCatWebhookEvent struct {
	APIVersion string              `json:"api_version"`
	Event      RevenueCatEventData `json:"event"`
}

type RevenueCatEventData struct {
	Type            string `json:"type"`
	AppUserID       string `json:"app_user_id"`
	EntitlementIDs  []string `json:"entitlement_ids"`
	ExpirationAtMs  int64  `json:"expiration_at_ms"`
	PeriodType      string `json:"period_type"`
	ProductID       string `json:"product_id"`
}

func (s *Subscription) HandleRevenueCatWebhook(ctx context.Context, authHeader string, body []byte) error {
	// Validate authorization - always required
	if s.rcWebhookAuthKey == "" {
		return fmt.Errorf("%w: REVENUECAT_WEBHOOK_AUTH_KEY not configured", ErrRevenueCatWebhook)
	}
	expectedHeader := "Bearer " + s.rcWebhookAuthKey
	if authHeader != expectedHeader {
		return fmt.Errorf("%w: invalid authorization header", ErrRevenueCatWebhook)
	}

	var webhook RevenueCatWebhookEvent
	if err := json.Unmarshal(body, &webhook); err != nil {
		return fmt.Errorf("%w: error unmarshaling webhook: %v", ErrRevenueCatWebhook, err)
	}

	event := webhook.Event
	uid := event.AppUserID
	if uid == "" {
		return fmt.Errorf("%w: missing app_user_id", ErrRevenueCatWebhook)
	}

	var plan authentication.Plan
	var expiresAt int64

	switch event.Type {
	case "INITIAL_PURCHASE", "RENEWAL", "UNCANCELLATION":
		plan = authentication.PlanPlus
	case "CANCELLATION", "PRODUCT_CHANGE":
		// User cancelled but still has access until expiration
		plan = authentication.PlanPlus
		if event.ExpirationAtMs > 0 {
			expiresAt = event.ExpirationAtMs / 1000 // Convert ms to seconds
		}
	case "EXPIRATION", "BILLING_ISSUE":
		plan = authentication.PlanFree
	default:
		// Other event types (e.g., TEST, TRANSFER) - no action needed
		return nil
	}

	err := s.firebaseGateway.SetUserSubscription(ctx, uid, plan, "", authentication.SubscriptionSourceIAP, expiresAt)
	if err != nil {
		return fmt.Errorf("%w: error updating firebase: %v", ErrRevenueCatWebhook, err)
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
			ID json.RawMessage `json:"id"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &event)
	if err != nil {
		fmt.Printf("Unmarshal error in validateSignature: %v\n", err)
	}

	// The id could be a string or a number in json, let's keep it exactly as it appeared (without quotes if we marshal it)
	// Actually, Mercado Pago docs say id:[data.id]
	idStr := strings.Trim(string(event.Data.ID), "\"")

	manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", idStr, xRequestId, ts)

	h := hmac.New(sha256.New, []byte(s.webhookSecret))
	h.Write([]byte(manifest))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if expectedSignature != v1 {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

