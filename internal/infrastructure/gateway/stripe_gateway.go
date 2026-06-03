package gateway

import (
	"context"
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/checkout/session"
	"github.com/stripe/stripe-go/v85/promotioncode"
	"github.com/stripe/stripe-go/v85/subscription"
	"github.com/stripe/stripe-go/v85/webhook"
)

const (
	envStripeSecretKey     = "STRIPE_SECRET_KEY"
	envStripeWebhookSecret = "STRIPE_WEBHOOK_SECRET"
	envStripeSuccessURL    = "STRIPE_SUCCESS_URL"
	envStripeCancelURL     = "STRIPE_CANCEL_URL"
)

type StripeGateway struct {
	webhookSecret string
	successURL    string
	cancelURL     string
}

func NewStripeGateway() *StripeGateway {
	// stripe.Key is a package-level global used by the stripe-go resource clients.
	stripe.Key = os.Getenv(envStripeSecretKey)
	return &StripeGateway{
		webhookSecret: os.Getenv(envStripeWebhookSecret),
		successURL:    os.Getenv(envStripeSuccessURL),
		cancelURL:     os.Getenv(envStripeCancelURL),
	}
}

// StripeCheckoutParams carries everything needed to open a subscription Checkout Session.
type StripeCheckoutParams struct {
	PriceID         string
	UserID          string
	SuccessURL      string // optional; falls back to STRIPE_SUCCESS_URL
	CancelURL       string // optional; falls back to STRIPE_CANCEL_URL
	PromotionCodeID string // optional; Stripe promotion_code id (promo_...) for a coupon
	RedemptionID    string // optional; our coupon redemption id, echoed back via metadata
}

// CreateCheckoutSession opens a subscription-mode Checkout Session and returns its URL.
// client_reference_id and subscription metadata carry our user id so RevenueCat (and our
// own webhook) can attribute the resulting subscription.
func (g *StripeGateway) CreateCheckoutSession(ctx context.Context, p StripeCheckoutParams) (string, error) {
	successURL := p.SuccessURL
	if successURL == "" {
		successURL = g.successURL
	}
	cancelURL := p.CancelURL
	if cancelURL == "" {
		cancelURL = g.cancelURL
	}

	metadata := map[string]string{"app_user_id": p.UserID}
	if p.RedemptionID != "" {
		metadata["coupon_redemption_id"] = p.RedemptionID
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(p.UserID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(p.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		},
	}
	params.Context = ctx

	if p.PromotionCodeID != "" {
		params.Discounts = []*stripe.CheckoutSessionDiscountParams{
			{PromotionCode: stripe.String(p.PromotionCodeID)},
		}
	}

	sess, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("error creating stripe checkout session: %w", err)
	}
	return sess.URL, nil
}

// StripePromotionCode is the validated subset of a Stripe promotion_code we care about.
type StripePromotionCode struct {
	ID       string
	Code     string
	Active   bool
	CouponID string
}

// ValidatePromotionCode looks up an active promotion_code by its customer-facing code.
// Returns found=false when no active code matches.
func (g *StripeGateway) ValidatePromotionCode(ctx context.Context, code string) (StripePromotionCode, bool, error) {
	params := &stripe.PromotionCodeListParams{
		Code:   stripe.String(code),
		Active: stripe.Bool(true),
	}
	params.Context = ctx
	params.Limit = stripe.Int64(1)

	iter := promotioncode.List(params)
	for iter.Next() {
		pc := iter.PromotionCode()
		couponID := ""
		if pc.Promotion != nil && pc.Promotion.Coupon != nil {
			couponID = pc.Promotion.Coupon.ID
		}
		return StripePromotionCode{ID: pc.ID, Code: pc.Code, Active: pc.Active, CouponID: couponID}, true, nil
	}
	if err := iter.Err(); err != nil {
		return StripePromotionCode{}, false, fmt.Errorf("error listing stripe promotion codes: %w", err)
	}
	return StripePromotionCode{}, false, nil
}

// CancelSubscription cancels a Stripe subscription. When atPeriodEnd is true the user keeps
// access until the paid period ends (graceful downgrade); otherwise it cancels immediately.
func (g *StripeGateway) CancelSubscription(ctx context.Context, subID string, atPeriodEnd bool) error {
	if atPeriodEnd {
		params := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(true)}
		params.Context = ctx
		if _, err := subscription.Update(subID, params); err != nil {
			return fmt.Errorf("error scheduling stripe subscription cancel: %w", err)
		}
		return nil
	}

	params := &stripe.SubscriptionCancelParams{}
	params.Context = ctx
	if _, err := subscription.Cancel(subID, params); err != nil {
		return fmt.Errorf("error cancelling stripe subscription: %w", err)
	}
	return nil
}

// ConstructWebhookEvent verifies the Stripe-Signature header and returns the parsed event.
func (g *StripeGateway) ConstructWebhookEvent(payload []byte, sigHeader string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, sigHeader, g.webhookSecret)
}
