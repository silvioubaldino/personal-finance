package authentication

import (
	"context"
	"fmt"
	"net/http"
	"os"

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

		plan := extractPlanFromClaims(token.Claims)
		role := extractRoleFromClaims(token.Claims)

		authCtx := NewAuthContext(token.UID, plan, role)
		ctx := ContextWithAuth(c.Request.Context(), authCtx)
		ctx = context.WithValue(ctx, UserID, token.UID)
		c.Request = c.Request.WithContext(ctx)
	}
}

func extractPlanFromClaims(claims map[string]interface{}) Plan {
	if plan, ok := claims["plan"].(string); ok {
		switch Plan(plan) {
		case PlanFree, PlanPlus:
			return Plan(plan)
		}
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

func (f firebaseAuth) DeleteUser(ctx context.Context, userID string) error {
	err := f.authClient.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("error deleting user from firebase: %w", err)
	}
	return nil
}
