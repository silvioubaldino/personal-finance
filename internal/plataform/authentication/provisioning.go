package authentication

import (
	"context"

	"personal-finance/pkg/log"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// UserProvisioner ensures a row exists in the users table for the authenticated user.
// Implemented by the user repository.
type UserProvisioner interface {
	EnsureExists(ctx context.Context, userID string) error
}

// LazyProvisionUser runs after Authenticate() and inserts the user row on first authenticated
// request. It first checks the `provisioned` custom claim on the Firebase token to skip the DB
// hit on subsequent requests. After a successful provision it persists the claim via the Admin
// SDK so future tokens carry it. Failures are logged but do not block the request — the row is
// best-effort and the next request will retry.
func LazyProvisionUser(provisioner UserProvisioner, authClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		authCtx, ok := AuthFromContext(ctx)
		if !ok || authCtx.UserID == "" {
			return
		}

		if authCtx.Provisioned {
			return
		}

		if err := provisioner.EnsureExists(ctx, authCtx.UserID); err != nil {
			log.ErrorContext(ctx, "failed to provision user row", log.Err(err))
			return
		}

		if authClient == nil {
			return
		}

		go setProvisionedClaim(authClient, authCtx.UserID)
	}
}

// setProvisionedClaim merges `provisioned: true` into the user's existing custom claims.
// Custom claim writes overwrite the full claim map, so existing claims (plan, role, etc.)
// must be fetched and preserved.
func setProvisionedClaim(authClient *auth.Client, userID string) {
	ctx := context.Background()

	user, err := authClient.GetUser(ctx, userID)
	if err != nil {
		log.ErrorContext(ctx, "failed to fetch user to set provisioned claim", log.Err(err))
		return
	}

	claims := user.CustomClaims
	if claims == nil {
		claims = make(map[string]interface{})
	}
	claims["provisioned"] = true

	if err := authClient.SetCustomUserClaims(ctx, userID, claims); err != nil {
		log.ErrorContext(ctx, "failed to set provisioned claim", log.Err(err))
	}
}
