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
	Plan authentication.Plan
	Role authentication.Role
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

	return UserClaims{
		Plan: plan,
		Role: role,
	}, nil
}

func (g *FirebaseGateway) SetUserPlan(ctx context.Context, userID string, plan authentication.Plan) error {
	user, err := g.authClient.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("error getting user from firebase: %w", err)
	}

	claims := user.CustomClaims
	if claims == nil {
		claims = make(map[string]interface{})
	}

	claims["plan"] = string(plan)

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
