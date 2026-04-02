package authentication

import (
	"context"
	"os"
	"strconv"
)

type Plan string

const (
	PlanFree Plan = "free"
	PlanPlus Plan = "plus"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type SubscriptionSource string

const (
	SubscriptionSourceNone SubscriptionSource = ""
	SubscriptionSourceMP   SubscriptionSource = "mp"
	SubscriptionSourceIAP  SubscriptionSource = "iap"
)

type AuthContext struct {
	UserID             string
	Email              string
	Plan               Plan
	Role               Role
	MPSubscriptionID   string
	SubscriptionSource SubscriptionSource
}

type authContextKey struct{}

func NewAuthContext(userID, email string, plan Plan, role Role, mpSubscriptionID string, subscriptionSource SubscriptionSource) AuthContext {
	return AuthContext{
		UserID:             userID,
		Email:              email,
		Plan:               plan,
		Role:               role,
		MPSubscriptionID:   mpSubscriptionID,
		SubscriptionSource: subscriptionSource,
	}
}

func (a AuthContext) IsFree() bool {
	return a.Plan == PlanFree
}

func (a AuthContext) IsPlus() bool {
	return a.Plan == PlanPlus
}

func (a AuthContext) IsAdmin() bool {
	return a.Role == RoleAdmin
}

func ContextWithAuth(ctx context.Context, auth AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, auth)
}

func AuthFromContext(ctx context.Context) (AuthContext, bool) {
	auth, ok := ctx.Value(authContextKey{}).(AuthContext)
	return auth, ok
}

func UserIDFromContext(ctx context.Context) string {
	auth, ok := AuthFromContext(ctx)
	if !ok {
		if userID, ok := ctx.Value(UserID).(string); ok {
			return userID
		}
		return ""
	}
	return auth.UserID
}

type PlanLimits struct {
	Wallets             int `json:"wallets"`
	CreditCards         int `json:"credit_cards"`
	MovementsPerMonth   int `json:"movements_per_month"`
	RecurrencesPerMonth int `json:"recurrences_per_month"`
}

func GetFreePlanLimits() PlanLimits {
	return PlanLimits{
		Wallets:             getEnvInt("PLAN_FREE_WALLETS_LIMIT", 2),
		CreditCards:         getEnvInt("PLAN_FREE_CREDIT_CARDS_LIMIT", 1),
		MovementsPerMonth:   getEnvInt("PLAN_FREE_MOVEMENTS_PER_MONTH_LIMIT", 50),
		RecurrencesPerMonth: getEnvInt("PLAN_FREE_RECURRENCES_PER_MONTH_LIMIT", 3),
	}
}

func getEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}
