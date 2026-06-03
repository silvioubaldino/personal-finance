package gateway

import (
	"context"
	"fmt"

	"firebase.google.com/go/v4/auth"

	"personal-finance/internal/plataform/authentication"
)

type FirebaseGateway struct {
	authClient *auth.Client
}

func NewFirebaseGateway(authClient *auth.Client) *FirebaseGateway {
	return &FirebaseGateway{
		authClient: authClient,
	}
}

type UserClaims struct {
	Plan               authentication.Plan
	Role               authentication.Role
	MPSubscriptionID   string
	SubscriptionSource authentication.SubscriptionSource
	PlanExpiresAt      int64
}

func (g *FirebaseGateway) GetUserClaims(ctx context.Context, userID string) (UserClaims, error) {
	user, err := g.authClient.GetUser(ctx, userID)
	if err != nil {
		return UserClaims{}, fmt.Errorf("error getting user from firebase: %w", err)
	}

	plan := authentication.PlanFree
	if p, ok := user.CustomClaims["plan"].(string); ok {
		switch authentication.Plan(p) {
		case authentication.PlanFree, authentication.PlanPlus:
			plan = authentication.Plan(p)
		}
	}

	role := authentication.RoleUser
	if r, ok := user.CustomClaims["role"].(string); ok {
		switch authentication.Role(r) {
		case authentication.RoleUser, authentication.RoleAdmin:
			role = authentication.Role(r)
		}
	}

	mpSubscriptionID := ""
	if mpID, ok := user.CustomClaims["mp_subscription_id"].(string); ok {
		mpSubscriptionID = mpID
	}

	subscriptionSource := authentication.SubscriptionSourceNone
	if source, ok := user.CustomClaims["subscription_source"].(string); ok {
		switch authentication.SubscriptionSource(source) {
		case authentication.SubscriptionSourceMP, authentication.SubscriptionSourceIAP, authentication.SubscriptionSourceStripe:
			subscriptionSource = authentication.SubscriptionSource(source)
		}
	}

	planExpiresAt := int64(0)
	if expiresAt, ok := user.CustomClaims["plan_expires_at"].(float64); ok {
		planExpiresAt = int64(expiresAt)
	}

	return UserClaims{
		Plan:               plan,
		Role:               role,
		MPSubscriptionID:   mpSubscriptionID,
		SubscriptionSource: subscriptionSource,
		PlanExpiresAt:      planExpiresAt,
	}, nil
}

func (g *FirebaseGateway) SetUserPlan(ctx context.Context, userID string, plan authentication.Plan, expiresAt *int64) error {
	user, err := g.authClient.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("error getting user from firebase: %w", err)
	}

	claims := user.CustomClaims
	if claims == nil {
		claims = make(map[string]interface{})
	}

	claims["plan"] = string(plan)

	if expiresAt != nil {
		if *expiresAt > 0 {
			claims["plan_expires_at"] = *expiresAt
		} else {
			delete(claims, "plan_expires_at")
		}
	} else if plan == authentication.PlanFree {
		delete(claims, "plan_expires_at")
	}

	err = g.authClient.SetCustomUserClaims(ctx, userID, claims)
	if err != nil {
		return fmt.Errorf("error setting custom claims: %w", err)
	}

	return nil
}

func (g *FirebaseGateway) SetUserSubscription(ctx context.Context, userID string, plan authentication.Plan, mpSubscriptionID string, subscriptionSource authentication.SubscriptionSource, expiresAt int64) error {
	user, err := g.authClient.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("error getting user from firebase: %w", err)
	}

	claims := user.CustomClaims
	if claims == nil {
		claims = make(map[string]interface{})
	}

	claims["plan"] = string(plan)
	if mpSubscriptionID != "" {
		claims["mp_subscription_id"] = mpSubscriptionID
	} else {
		delete(claims, "mp_subscription_id")
	}

	if subscriptionSource != authentication.SubscriptionSourceNone {
		claims["subscription_source"] = string(subscriptionSource)
	} else {
		delete(claims, "subscription_source")
	}

	if expiresAt > 0 {
		claims["plan_expires_at"] = expiresAt
	} else {
		delete(claims, "plan_expires_at")
	}

	err = g.authClient.SetCustomUserClaims(ctx, userID, claims)
	if err != nil {
		return fmt.Errorf("error setting custom claims: %w", err)
	}

	return nil
}

func (g *FirebaseGateway) SetUserRole(ctx context.Context, userID string, role authentication.Role) error {
	user, err := g.authClient.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("error getting user from firebase: %w", err)
	}

	claims := user.CustomClaims
	if claims == nil {
		claims = make(map[string]interface{})
	}

	claims["role"] = string(role)

	err = g.authClient.SetCustomUserClaims(ctx, userID, claims)
	if err != nil {
		return fmt.Errorf("error setting custom claims: %w", err)
	}

	return nil
}
