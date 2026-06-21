// Package health exposes Kubernetes/Cloud Run style liveness and readiness
// probes. These routes are registered before authentication so probes can run
// unauthenticated.
package health

import (
	"net/http"

	"personal-finance/pkg/log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Register wires the /healthz (liveness) and /readyz (readiness) endpoints onto
// the router. Liveness has no dependencies; readiness pings the database.
func Register(r gin.IRouter, db *gorm.DB) {
	r.GET("/healthz", liveness())
	r.GET("/readyz", readiness(db))
}

// liveness reports that the process is up. It must not touch dependencies.
func liveness() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// readiness reports whether the service can serve traffic by pinging the DB.
func readiness(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		sqlDB, err := db.DB()
		if err != nil {
			log.ErrorContext(ctx, "readiness: failed to access sql.DB", log.Err(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}

		if err := sqlDB.PingContext(ctx); err != nil {
			log.ErrorContext(ctx, "readiness: database ping failed", log.Err(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
