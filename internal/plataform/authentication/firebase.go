package authentication

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"

	"personal-finance/internal/model"
	"personal-finance/pkg/log"
)

const (
	UserID    = "user_id"
	UserToken = "user_token"
)

type Authenticator interface {
	Authenticate() gin.HandlerFunc
	DeleteUser(ctx context.Context, userID string) error
	AuthClient() *auth.Client
}

type firebaseAuth struct {
	authClient *auth.Client
}

func NewFirebaseAuth() Authenticator {
	projectID := os.Getenv("GOOGLE_PROJECT_ID")
	config := &firebase.Config{ProjectID: projectID}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Fatal("error initializing app", log.Err(err))
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatal("error getting Auth Client", log.Err(err))
	}

	return &firebaseAuth{
		authClient: authClient,
	}
}

func (f *firebaseAuth) AuthClient() *auth.Client {
	return f.authClient
}

func (f *firebaseAuth) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader(UserToken)
		if userToken == "" {
			log.ErrorContext(c.Request.Context(), "empty token")
			c.JSON(http.StatusUnauthorized, model.ErrEmptyToken.Error())
			c.Abort()
			return
		}

		token, err := f.authClient.VerifyIDToken(c.Request.Context(), userToken)
		if err != nil {
			log.ErrorContext(c.Request.Context(), "error verifying ID token", log.Err(err))
			c.JSON(http.StatusUnauthorized, "error verifying ID token: internal error")
			c.Abort()
			return
		}

		plan := f.extractPlanFromClaims(c.Request.Context(), token.UID, token.Claims)
		role := extractRoleFromClaims(token.Claims)
		mpSubscriptionID := extractMPSubscriptionIDFromClaims(token.Claims)
		email := extractEmailFromClaims(token.Claims)
		subscriptionSource := extractSubscriptionSourceFromClaims(token.Claims)
		provisioned := extractProvisionedFromClaims(token.Claims)

		authCtx := NewAuthContext(token.UID, email, plan, role, mpSubscriptionID, subscriptionSource, provisioned)
		ctx := ContextWithAuth(c.Request.Context(), authCtx)
		ctx = context.WithValue(ctx, UserID, token.UID)
		c.Request = c.Request.WithContext(ctx)
	}
}

func (f *firebaseAuth) extractPlanFromClaims(ctx context.Context, uid string, claims map[string]interface{}) Plan {
	if plan, ok := claims["plan"].(string); ok {
		currentPlan := PlanFree
		switch Plan(plan) {
		case PlanFree, PlanPlus:
			currentPlan = Plan(plan)
		}

		if currentPlan == PlanPlus {
			if expiresAt, ok := claims["plan_expires_at"].(float64); ok {
				if time.Now().Unix() > int64(expiresAt) {
					go func() {
						bgCtx := context.Background()

						// Firebase ID token claims contains reserved words (iss, aud, exp, etc). 
						// To update Custom Claims we must fetch the raw Custom Claims from the user record.
						user, err := f.authClient.GetUser(bgCtx, uid)
						if err != nil {
							log.ErrorContext(bgCtx, "failed to fetch user to downgrade expired subscription", log.Err(err))
							return
						}

						userClaims := user.CustomClaims
						if userClaims == nil {
							userClaims = make(map[string]interface{})
						}

						userClaims["plan"] = string(PlanFree)
						delete(userClaims, "plan_expires_at")

						err = f.authClient.SetCustomUserClaims(bgCtx, uid, userClaims)
						if err != nil {
							log.ErrorContext(bgCtx, "failed hard downgrading expired subscription", log.Err(err))
						}
					}()

					return PlanFree
				}
			}
		}

		return currentPlan
	}
	return PlanFree
}

func extractRoleFromClaims(claims map[string]interface{}) Role {
	if role, ok := claims["role"].(string); ok {
		switch Role(role) {
		case RoleUser, RoleAdmin:
			return Role(role)
		}
	}
	return RoleUser
}

func extractMPSubscriptionIDFromClaims(claims map[string]interface{}) string {
	if mpID, ok := claims["mp_subscription_id"].(string); ok {
		return mpID
	}
	return ""
}

func extractEmailFromClaims(claims map[string]interface{}) string {
	if email, ok := claims["email"].(string); ok {
		return email
	}
	return ""
}

func extractProvisionedFromClaims(claims map[string]interface{}) bool {
	if provisioned, ok := claims["provisioned"].(bool); ok {
		return provisioned
	}
	return false
}

func extractSubscriptionSourceFromClaims(claims map[string]interface{}) SubscriptionSource {
	if source, ok := claims["subscription_source"].(string); ok {
		switch SubscriptionSource(source) {
		case SubscriptionSourceMP, SubscriptionSourceIAP, SubscriptionSourceStripe:
			return SubscriptionSource(source)
		}
	}
	return SubscriptionSourceNone
}

func (f firebaseAuth) DeleteUser(ctx context.Context, userID string) error {
	err := f.authClient.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("error deleting user from firebase: %w", err)
	}
	return nil
}
