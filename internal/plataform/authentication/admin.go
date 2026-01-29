package authentication

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"personal-finance/pkg/log"
)

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth, ok := AuthFromContext(c.Request.Context())
		if !ok {
			log.ErrorContext(c.Request.Context(), "auth context not found")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		if !auth.IsAdmin() {
			log.WarnContext(c.Request.Context(), "forbidden: admin role required",
				log.String("user_id", auth.UserID),
				log.String("role", string(auth.Role)),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: admin role required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
