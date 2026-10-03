//go:build acceptance

package harness

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"

	"personal-finance/internal/plataform/authentication"
)

// testAuth replaces Firebase in the suite. The user_token header is "<user_id>" or
// "<user_id>;plan=free"; there is no signature, so it only ever lives in this package.
type testAuth struct{}

func (testAuth) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(authentication.UserToken)
		if token == "" {
			c.JSON(http.StatusUnauthorized, "empty token")
			c.Abort()
			return
		}

		userID, plan := parseToken(token)
		authCtx := authentication.NewAuthContext(userID, "", plan, authentication.RoleUser, "", authentication.SubscriptionSourceNone, false)

		ctx := authentication.ContextWithAuth(c.Request.Context(), authCtx)
		ctx = context.WithValue(ctx, authentication.UserID, userID)
		c.Request = c.Request.WithContext(ctx)
	}
}

func (testAuth) DeleteUser(context.Context, string) error { return nil }

// AuthClient is nil: LazyProvisionUser and the gateways tolerate it.
func (testAuth) AuthClient() *auth.Client { return nil }

func parseToken(token string) (string, authentication.Plan) {
	userID, rest, _ := strings.Cut(token, ";")
	plan := authentication.PlanPlus
	if rest == "plan=free" {
		plan = authentication.PlanFree
	}
	return userID, plan
}

func buildToken(userID, plan string) string {
	if plan == string(authentication.PlanFree) {
		return userID + ";plan=free"
	}
	return userID
}
