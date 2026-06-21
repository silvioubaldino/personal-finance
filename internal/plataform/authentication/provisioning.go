package authentication

import (
	"context"

	"personal-finance/pkg/log"
	"personal-finance/pkg/metrics"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

type UserProvisioner interface {
	EnsureExists(ctx context.Context, userID string) error
}

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

		// Example business KPI (foundation): a user reaching this branch was not
		// yet provisioned, so this counts first-time provisioning. The remaining
		// AyD §6.2 KPI catalog is follow-up work.
		metrics.BusinessCounter(ctx, "biz_users_provisioned_total", 1)

		if authClient == nil {
			return
		}

		go setProvisionedClaim(authClient, authCtx.UserID)
	}
}

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
