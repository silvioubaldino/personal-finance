package authentication

import (
	"context"

	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
)

// UserProvisioner ensures a row exists in the users table for the authenticated user.
// Implemented by the user repository.
type UserProvisioner interface {
	EnsureExists(ctx context.Context, userID string) error
}

// LazyProvisionUser runs after Authenticate() and inserts the user row on first authenticated
// request. Failures are logged but do not block the request — the row is best-effort and the
// next request will retry.
func LazyProvisionUser(provisioner UserProvisioner) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		userID, ok := ctx.Value(UserID).(string)
		if !ok || userID == "" {
			return
		}

		if err := provisioner.EnsureExists(ctx, userID); err != nil {
			log.ErrorContext(ctx, "failed to provision user row", log.Err(err))
		}
	}
}
