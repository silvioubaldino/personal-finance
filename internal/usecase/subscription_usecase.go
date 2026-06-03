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

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v85"
)

type MercadoPagoSubscriptionGateway interface {
	CreateSubscriptionURL(ctx context.Context, payerEmail, externalReference, backURL string, plan gateway.SubscriptionPlanConfig) (string, error)
	GetSubscription(ctx context.Context, id string) (gateway.MPSubscription, error)
	CancelSubscription(ctx context.Context, id string) error
}

type StripeSubscriptionGateway interface {
	CreateCheckoutSession(ctx context.Context, p gateway.StripeCheckoutParams) (string, error)
	ValidatePromotionCode(ctx context.Context, code string) (gateway.StripePromotionCode, bool, error)
	CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error
	ConstructWebhookEvent(payload []byte, sigHeader string) (stripe.Event, error)
}

const externalReferenceSeparator = "|"

func parseExternalReference(ref string) (userID, planID string, redemptionID uuid.UUID) {
	parts := strings.SplitN(ref, externalReferenceSeparator, 3)
	switch len(parts) {
	case 3:
		redemptionID, _ = uuid.Parse(parts[2])
		return parts[0], parts[1], redemptionID
	case 2:
		return parts[0], parts[1], uuid.Nil
	default:
		return ref, "", uuid.Nil
	}
}

type (
	FirebaseSubscriptionGateway interface {
		SetUserSubscription(ctx context.Context, userID string, plan authentication.Plan, mpSubscriptionID string, subscriptionSource authentication.SubscriptionSource, expiresAt int64) error
	}

	SubscriptionPlanRepository interface {
		Create(ctx context.Context, plan domain.SubscriptionPlan) error
		FindActive(ctx context.Context) ([]domain.SubscriptionPlan, error)
		FindActiveByID(ctx context.Context, id string) (domain.SubscriptionPlan, error)
		FindIDByStoreProduct(ctx context.Context, store, productID string) (string, error)
	}

	SubscriptionRepository interface {
		Upsert(ctx context.Context, sub domain.Subscription) (domain.Subscription, error)
		List(ctx context.Context, filter repository.SubscriptionListFilter) ([]domain.Subscription, error)
		FindActiveByUserAndSource(ctx context.Context, userID string, source domain.SubscriptionSource) (domain.Subscription, error)
	}

	CouponCheckoutUseCase interface {
		ApplyWebCheckout(ctx context.Context, userID string, plan domain.SubscriptionPlan, code string) (redemptionID uuid.UUID, err error)
		Confirm(ctx context.Context, redemptionID, subscriptionID uuid.UUID) error
		MarkCancelledBySubscription(ctx context.Context, subscriptionID uuid.UUID) error
	}

	SubscriptionUseCase interface {
		CreateCheckout(ctx context.Context, planID, successURL, cancelURL, couponCode string) (string, error)
		CancelSubscription(ctx context.Context) error
		HandleWebhook(ctx context.Context, xSignature, xRequestId string, body []byte) error
		HandleStripeWebhook(ctx context.Context, payload []byte, sigHeader string) error
		HandleRevenueCatWebhook(ctx context.Context, authHeader string, body []byte) error
		GetActivePlans(ctx context.Context) ([]domain.SubscriptionPlan, error)
	}
)

type Subscription struct {
	mpGateway        MercadoPagoSubscriptionGateway
	stripeGateway    StripeSubscriptionGateway
	firebaseGateway  FirebaseSubscriptionGateway
	planRepo         SubscriptionPlanRepository
	subRepo          SubscriptionRepository
	couponUseCase    CouponCheckoutUseCase
	webhookSecret    string
	rcWebhookAuthKey string
}

func NewSubscription(
	mpGateway MercadoPagoSubscriptionGateway,
	stripeGateway StripeSubscriptionGateway,
	firebaseGateway FirebaseSubscriptionGateway,
	planRepo SubscriptionPlanRepository,
	subRepo SubscriptionRepository,
	couponUseCase CouponCheckoutUseCase,
) *Subscription {
	return &Subscription{
		mpGateway:        mpGateway,
		stripeGateway:    stripeGateway,
		firebaseGateway:  firebaseGateway,
		planRepo:         planRepo,
		subRepo:          subRepo,
		couponUseCase:    couponUseCase,
		webhookSecret:    os.Getenv("MERCADOPAGO_WEBHOOK_SECRET"),
		rcWebhookAuthKey: os.Getenv("REVENUECAT_WEBHOOK_AUTH_KEY"),
	}
}

type CheckoutResponse struct {
	URL string `json:"checkout_url"`
}

// CreateCheckout opens a Stripe Checkout Session for the web subscription flow and returns
// its URL. (MercadoPago checkout is retired; only its webhook/cancel remain for legacy subs.)
func (s *Subscription) CreateCheckout(ctx context.Context, planID, successURL, cancelURL, couponCode string) (string, error) {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return "", ErrUnauthorized
	}

	plan, err := s.planRepo.FindActiveByID(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSubscriptionPlanNotFound, err)
	}
	if plan.StripePriceID == "" {
		return "", ErrStripePriceNotConfigured
	}

	params := gateway.StripeCheckoutParams{
		PriceID:    plan.StripePriceID,
		UserID:     auth.UserID,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
	}

	if couponCode != "" {
		redemptionID, err := s.couponUseCase.ApplyWebCheckout(ctx, auth.UserID, plan, couponCode)
		if err != nil {
			return "", err
		}

		// The coupon code is the same string configured as the Stripe promotion_code.
		pc, found, err := s.stripeGateway.ValidatePromotionCode(ctx, couponCode)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrStripeGateway, err)
		}
		if !found {
			return "", domain.ErrCouponNotOnStripe
		}

		params.PromotionCodeID = pc.ID
		if redemptionID != uuid.Nil {
			params.RedemptionID = redemptionID.String()
		}
	}

	url, err := s.stripeGateway.CreateCheckoutSession(ctx, params)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStripeGateway, err)
	}
	return url, nil
}

type SubscriptionsSummary struct {
	TotalSubscriptions      int                `json:"total_subscriptions"`
	ActiveSubscriptions     int                `json:"active_subscriptions"`
	BySource                map[string]int     `json:"by_source"`
	ByStatus                map[string]int     `json:"by_status"`
	ActiveRevenueByCurrency map[string]float64 `json:"active_revenue_by_currency"`
}

func (s *Subscription) SummarizeSubscriptions(ctx context.Context, filter repository.SubscriptionListFilter) (SubscriptionsSummary, error) {
	summary := SubscriptionsSummary{
		BySource:                map[string]int{},
		ByStatus:                map[string]int{},
		ActiveRevenueByCurrency: map[string]float64{},
	}

	if s.subRepo == nil {
		return summary, nil
	}

	subs, err := s.subRepo.List(ctx, filter)
	if err != nil {
		return SubscriptionsSummary{}, err
	}

	summary.TotalSubscriptions = len(subs)
	for _, sub := range subs {
		summary.BySource[string(sub.Source)]++
		summary.ByStatus[string(sub.Status)]++
		if sub.Status == domain.SubscriptionStatusActive {
			summary.ActiveSubscriptions++
			if sub.Currency != "" {
				summary.ActiveRevenueByCurrency[sub.Currency] += sub.CurrentPrice
			}
		}
	}

	return summary, nil
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

	switch auth.SubscriptionSource {
	case authentication.SubscriptionSourceStripe:
		return s.cancelStripeSubscription(ctx, auth.UserID)
	case authentication.SubscriptionSourceMP:
		return s.cancelMPSubscription(ctx, auth)
	default:
		// Older claims may still carry an MP subscription id without an explicit source.
		if auth.MPSubscriptionID != "" {
			return s.cancelMPSubscription(ctx, auth)
		}
		return fmt.Errorf("user does not have an active subscription to cancel")
	}
}

func (s *Subscription) cancelStripeSubscription(ctx context.Context, userID string) error {
	sub, err := s.subRepo.FindActiveByUserAndSource(ctx, userID, domain.SubscriptionSourceStripe)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSubscriptionNotFound, err)
	}
	if err := s.stripeGateway.CancelSubscription(ctx, sub.ExternalID, true); err != nil {
		return fmt.Errorf("%w: %v", ErrStripeGateway, err)
	}
	return nil
}

func (s *Subscription) cancelMPSubscription(ctx context.Context, auth authentication.AuthContext) error {
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

	plan := authentication.PlanFree
	var expiresAt int64
	if parsed := parseMPDate(subscription.NextPaymentDate); !parsed.IsZero() && parsed.After(time.Now()) {
		plan = authentication.PlanPlus
		expiresAt = parsed.Unix()
	}

	_, planID, _ := parseExternalReference(subscription.ExternalReference)
	upserted, err := s.upsertMPSubscription(ctx, auth.UserID, planID, subscription)
	if err != nil {
		return fmt.Errorf("error mirroring cancelled subscription to db: %w", err)
	}

	if upserted.ID != uuid.Nil {
		_ = s.couponUseCase.MarkCancelledBySubscription(ctx, upserted.ID)
	}

	// Empty MP Subscription ID means they won't be charged anymore and prevents repeat cancels.
	err = s.firebaseGateway.SetUserSubscription(ctx, auth.UserID, plan, "", authentication.SubscriptionSourceMP, expiresAt)
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

	uid, planID, redemptionID := parseExternalReference(subscription.ExternalReference)
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
	case "pending":
		_, err := s.upsertMPSubscription(ctx, uid, planID, subscription)
		if err != nil {
			return fmt.Errorf("error mirroring pending subscription to db: %w", err)
		}
		return nil
	case "cancelled", "paused":
		plan = authentication.PlanFree
		if parsed := parseMPDate(subscription.NextPaymentDate); !parsed.IsZero() && parsed.After(time.Now()) {
			plan = authentication.PlanPlus
			expiresAt = parsed.Unix()
		}
	default:
		return nil
	}

	// DB mirror runs before Firebase so a retry from MP refreshes both sources idempotently.
	upserted, err := s.upsertMPSubscription(ctx, uid, planID, subscription)
	if err != nil {
		return fmt.Errorf("error mirroring subscription to db: %w", err)
	}

	if redemptionID != uuid.Nil && upserted.ID != uuid.Nil {
		switch subscription.Status {
		case "authorized":
			_ = s.couponUseCase.Confirm(ctx, redemptionID, upserted.ID)
		case "cancelled", "paused":
			_ = s.couponUseCase.MarkCancelledBySubscription(ctx, upserted.ID)
		}
	}

	err = s.firebaseGateway.SetUserSubscription(ctx, uid, plan, mpSubID, authentication.SubscriptionSourceMP, expiresAt)
	if err != nil {
		return fmt.Errorf("error updating firebase subscription data: %w", err)
	}

	return nil
}

func (s *Subscription) upsertMPSubscription(ctx context.Context, userID, planID string, mp gateway.MPSubscription) (domain.Subscription, error) {
	if s.subRepo == nil {
		return domain.Subscription{}, nil
	}

	status := mapMPStatusToDomain(mp.Status)
	if status == "" {
		return domain.Subscription{}, nil
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
		PlanID:           planID,
		Status:           status,
		CurrentPrice:     mp.AutoRecurring.TransactionAmount,
		Currency:         mp.AutoRecurring.CurrencyID,
		StartedAt:        startedAt,
		CurrentPeriodEnd: currentPeriodEnd,
		CancelledAt:      cancelledAt,
	}

	return s.subRepo.Upsert(ctx, sub)
}

func mapMPStatusToDomain(mpStatus string) domain.SubscriptionStatus {
	switch mpStatus {
	case "pending":
		return domain.SubscriptionStatusPending
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

func (s *Subscription) HandleStripeWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	event, err := s.stripeGateway.ConstructWebhookEvent(payload, sigHeader)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidWebhookSignature, err)
	}

	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		return s.handleStripeSubscriptionUpsert(ctx, event)
	case "customer.subscription.deleted":
		return s.handleStripeSubscriptionDeleted(ctx, event)
	default:
		return nil
	}
}

func (s *Subscription) handleStripeSubscriptionUpsert(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("%w: error unmarshaling stripe subscription: %v", ErrStripeGateway, err)
	}

	userID := sub.Metadata["app_user_id"]
	if userID == "" {
		return fmt.Errorf("%w: no app_user_id in stripe subscription %s", ErrStripeGateway, sub.ID)
	}
	redemptionID, _ := uuid.Parse(sub.Metadata["coupon_redemption_id"])

	upserted, err := s.upsertStripeSubscription(ctx, userID, sub)
	if err != nil {
		return fmt.Errorf("error mirroring stripe subscription to db: %w", err)
	}

	active := sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing
	if !active {
		// past_due / unpaid / incomplete: mirror to DB but don't change the claim here.
		return nil
	}

	var expiresAt int64
	if sub.CancelAtPeriodEnd {
		// Cancellation scheduled: keep Plus until the period ends, then auto-downgrade.
		expiresAt = stripeCurrentPeriodEnd(sub)
	}
	if err := s.firebaseGateway.SetUserSubscription(ctx, userID, authentication.PlanPlus, sub.ID, authentication.SubscriptionSourceStripe, expiresAt); err != nil {
		return fmt.Errorf("error updating firebase subscription data: %w", err)
	}

	if redemptionID != uuid.Nil && upserted.ID != uuid.Nil {
		_ = s.couponUseCase.Confirm(ctx, redemptionID, upserted.ID)
	}
	return nil
}

func (s *Subscription) handleStripeSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("%w: error unmarshaling stripe subscription: %v", ErrStripeGateway, err)
	}

	userID := sub.Metadata["app_user_id"]
	if userID == "" {
		return fmt.Errorf("%w: no app_user_id in stripe subscription %s", ErrStripeGateway, sub.ID)
	}

	upserted, err := s.upsertStripeSubscription(ctx, userID, sub)
	if err != nil {
		return fmt.Errorf("error mirroring stripe subscription to db: %w", err)
	}

	// Grace until the paid period ends (if still in the future), then Free.
	plan := authentication.PlanFree
	var expiresAt int64
	if end := stripeCurrentPeriodEnd(sub); end > time.Now().Unix() {
		plan = authentication.PlanPlus
		expiresAt = end
	}
	if err := s.firebaseGateway.SetUserSubscription(ctx, userID, plan, "", authentication.SubscriptionSourceStripe, expiresAt); err != nil {
		return fmt.Errorf("error updating firebase subscription data: %w", err)
	}

	if upserted.ID != uuid.Nil {
		_ = s.couponUseCase.MarkCancelledBySubscription(ctx, upserted.ID)
	}
	return nil
}

func (s *Subscription) upsertStripeSubscription(ctx context.Context, userID string, sub stripe.Subscription) (domain.Subscription, error) {
	if s.subRepo == nil {
		return domain.Subscription{}, nil
	}

	status := mapStripeStatusToDomain(sub.Status)
	if status == "" {
		return domain.Subscription{}, nil
	}

	var priceID string
	var currentPrice float64
	currency := strings.ToUpper(string(sub.Currency))
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.Price != nil {
			priceID = item.Price.ID
			currentPrice = float64(item.Price.UnitAmount) / 100.0
			if currency == "" {
				currency = strings.ToUpper(string(item.Price.Currency))
			}
		}
	}

	planID, _ := s.planRepo.FindIDByStoreProduct(ctx, "STRIPE", priceID)

	startedAt := time.Unix(sub.Created, 0)
	var currentPeriodEnd *time.Time
	if end := stripeCurrentPeriodEnd(sub); end > 0 {
		t := time.Unix(end, 0)
		currentPeriodEnd = &t
	}
	var cancelledAt *time.Time
	if status == domain.SubscriptionStatusCancelled {
		now := time.Now()
		cancelledAt = &now
	}

	return s.subRepo.Upsert(ctx, domain.Subscription{
		UserID:           userID,
		Source:           domain.SubscriptionSourceStripe,
		ExternalID:       sub.ID,
		PlanID:           planID,
		Status:           status,
		CurrentPrice:     currentPrice,
		Currency:         currency,
		StartedAt:        startedAt,
		CurrentPeriodEnd: currentPeriodEnd,
		CancelledAt:      cancelledAt,
	})
}

func mapStripeStatusToDomain(st stripe.SubscriptionStatus) domain.SubscriptionStatus {
	switch st {
	case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusTrialing:
		return domain.SubscriptionStatusActive
	case stripe.SubscriptionStatusCanceled:
		return domain.SubscriptionStatusCancelled
	case stripe.SubscriptionStatusPastDue, stripe.SubscriptionStatusUnpaid:
		return domain.SubscriptionStatusPastDue
	case stripe.SubscriptionStatusPaused:
		return domain.SubscriptionStatusPaused
	default:
		return ""
	}
}

// stripeCurrentPeriodEnd reads the current period end. In recent Stripe API versions this lives
// on the subscription item rather than the subscription itself.
func stripeCurrentPeriodEnd(sub stripe.Subscription) int64 {
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		return sub.Items.Data[0].CurrentPeriodEnd
	}
	return 0
}

// RevenueCat webhook types
type RevenueCatWebhookEvent struct {
	APIVersion string              `json:"api_version"`
	Event      RevenueCatEventData `json:"event"`
}

type RevenueCatEventData struct {
	Type                     string   `json:"type"`
	AppUserID                string   `json:"app_user_id"`
	EntitlementIDs           []string `json:"entitlement_ids"`
	ExpirationAtMs           int64    `json:"expiration_at_ms"`
	PeriodType               string   `json:"period_type"`
	ProductID                string   `json:"product_id"`
	Store                    string   `json:"store"`
	OriginalTransactionID    string   `json:"original_transaction_id"`
	PurchasedAtMs            int64    `json:"purchased_at_ms"`
	PriceInPurchasedCurrency float64  `json:"price_in_purchased_currency"`
	Currency                 string   `json:"currency"`
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

	// Stripe (web) subscriptions are handled by our own /webhooks/stripe. RevenueCat receives
	// Stripe events only for metrics; ignore them here to avoid double-processing.
	if event.Store == "STRIPE" {
		return nil
	}

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

	if err := s.upsertRCSubscription(ctx, uid, event); err != nil {
		return fmt.Errorf("%w: error mirroring subscription to db: %v", ErrRevenueCatWebhook, err)
	}

	err := s.firebaseGateway.SetUserSubscription(ctx, uid, plan, "", authentication.SubscriptionSourceIAP, expiresAt)
	if err != nil {
		return fmt.Errorf("%w: error updating firebase: %v", ErrRevenueCatWebhook, err)
	}

	return nil
}

func (s *Subscription) upsertRCSubscription(ctx context.Context, userID string, event RevenueCatEventData) error {
	if s.subRepo == nil {
		return nil
	}

	status := mapRCStatusToDomain(event.Type)
	if status == "" {
		return nil
	}

	source := mapRCStoreToDomain(event.Store)
	if source == "" {
		return nil
	}

	startedAt := time.Now()
	if event.PurchasedAtMs > 0 {
		startedAt = time.UnixMilli(event.PurchasedAtMs)
	}

	var currentPeriodEnd *time.Time
	if event.ExpirationAtMs > 0 {
		t := time.UnixMilli(event.ExpirationAtMs)
		currentPeriodEnd = &t
	}

	var cancelledAt *time.Time
	if status == domain.SubscriptionStatusCancelled {
		now := time.Now()
		cancelledAt = &now
	}

	planID, err := s.planRepo.FindIDByStoreProduct(ctx, event.Store, event.ProductID)
	if err != nil {
		return fmt.Errorf("error resolving plan id for store product: %w", err)
	}

	sub := domain.Subscription{
		UserID:            userID,
		Source:            source,
		ExternalID:        event.OriginalTransactionID,
		ExternalProductID: event.ProductID,
		PlanID:            planID,
		Status:            status,
		CurrentPrice:      event.PriceInPurchasedCurrency,
		Currency:          event.Currency,
		StartedAt:         startedAt,
		CurrentPeriodEnd:  currentPeriodEnd,
		CancelledAt:       cancelledAt,
	}

	_, err = s.subRepo.Upsert(ctx, sub)
	return err
}

func mapRCStatusToDomain(eventType string) domain.SubscriptionStatus {
	switch eventType {
	case "INITIAL_PURCHASE", "RENEWAL", "UNCANCELLATION":
		return domain.SubscriptionStatusActive
	case "CANCELLATION", "PRODUCT_CHANGE":
		return domain.SubscriptionStatusCancelled
	case "EXPIRATION":
		return domain.SubscriptionStatusExpired
	case "BILLING_ISSUE":
		return domain.SubscriptionStatusPastDue
	default:
		return ""
	}
}

func mapRCStoreToDomain(store string) domain.SubscriptionSource {
	switch store {
	case "APP_STORE":
		return domain.SubscriptionSourceApple
	case "PLAY_STORE":
		return domain.SubscriptionSourceGoogle
	default:
		return ""
	}
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
