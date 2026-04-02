package usecase

import (
	"context"
	"time"

	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/plataform/authentication"
)

type FirebaseClaimsGateway interface {
	GetUserClaims(ctx context.Context, userID string) (gateway.UserClaims, error)
	SetUserPlan(ctx context.Context, userID string, plan authentication.Plan, expiresAt *int64) error
	SetUserRole(ctx context.Context, userID string, role authentication.Role) error
}

type Admin struct {
	firebaseGateway FirebaseClaimsGateway
}

func NewAdmin(firebaseGateway FirebaseClaimsGateway) *Admin {
	return &Admin{
		firebaseGateway: firebaseGateway,
	}
}

type UserClaimsResponse struct {
	UserID        string    `json:"user_id"`
	Plan          string    `json:"plan"`
	Role          string    `json:"role"`
	PlanExpiresAt time.Time `json:"plan_expires_at"`
}

func (a *Admin) GetUserClaims(ctx context.Context, userID string) (UserClaimsResponse, error) {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return UserClaimsResponse{}, ErrUnauthorized
	}

	if !auth.IsAdmin() {
		return UserClaimsResponse{}, ErrForbidden
	}

	claims, err := a.firebaseGateway.GetUserClaims(ctx, userID)
	if err != nil {
		return UserClaimsResponse{}, err
	}

	return UserClaimsResponse{
		UserID:        userID,
		Plan:          string(claims.Plan),
		Role:          string(claims.Role),
		PlanExpiresAt: time.Unix(claims.PlanExpiresAt, 0),
	}, nil
}

func (a *Admin) SetUserPlan(ctx context.Context, userID string, plan string, expiresAt *time.Time) error {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return ErrUnauthorized
	}

	if !auth.IsAdmin() {
		return ErrForbidden
	}

	p := authentication.Plan(plan)
	if p != authentication.PlanFree && p != authentication.PlanPlus {
		return ErrInvalidPlan
	}

	var expiresAtInt64 *int64
	if expiresAt != nil {
		val := expiresAt.Unix()
		expiresAtInt64 = &val
	}

	return a.firebaseGateway.SetUserPlan(ctx, userID, p, expiresAtInt64)
}

func (a *Admin) SetUserRole(ctx context.Context, userID string, role string) error {
	auth, ok := authentication.AuthFromContext(ctx)
	if !ok {
		return ErrUnauthorized
	}

	if !auth.IsAdmin() {
		return ErrForbidden
	}

	r := authentication.Role(role)
	if r != authentication.RoleUser && r != authentication.RoleAdmin {
		return ErrInvalidRole
	}

	return a.firebaseGateway.SetUserRole(ctx, userID, r)
}
