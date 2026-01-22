package authentication

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	APIKeyHeader  = "X-API-Key"
	APIKeyEnvName = "INTERNAL_API_KEY"
)

func InternalAPIKeyAuth() gin.HandlerFunc {
	expectedKey := os.Getenv(APIKeyEnvName)

	return func(c *gin.Context) {
		if expectedKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal api key not configured"})
			c.Abort()
			return
		}

		providedKey := c.GetHeader(APIKeyHeader)
		if providedKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			c.Abort()
			return
		}

		if providedKey != expectedKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			c.Abort()
			return
		}

		c.Next()
	}
}
