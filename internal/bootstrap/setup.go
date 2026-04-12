package bootstrap

import (
	"personal-finance/internal/bootstrap/admin"
	"personal-finance/internal/bootstrap/agent"
	"personal-finance/internal/bootstrap/balance"
	"personal-finance/internal/bootstrap/category"
	"personal-finance/internal/bootstrap/creditcard"
	"personal-finance/internal/bootstrap/deleteaccount"
	"personal-finance/internal/bootstrap/device"
	"personal-finance/internal/bootstrap/estimate"
	"personal-finance/internal/bootstrap/export"
	"personal-finance/internal/bootstrap/invoice"
	"personal-finance/internal/bootstrap/limits"
	"personal-finance/internal/bootstrap/movement"
	"personal-finance/internal/bootstrap/pushnotifications"
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/bootstrap/statement"
	"personal-finance/internal/bootstrap/subcategory"
	"personal-finance/internal/bootstrap/subscription"
	"personal-finance/internal/bootstrap/transfer"
	"personal-finance/internal/bootstrap/userconsent"
	"personal-finance/internal/bootstrap/userpreferences"
	"personal-finance/internal/bootstrap/wallet"
	"personal-finance/internal/plataform/authentication"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupInternalJobs(r *gin.Engine, db *gorm.DB) {
	reg := registry.NewRegistry(db)

	jobsGroup := r.Group("/jobs")
	jobsGroup.Use(authentication.InternalAPIKeyAuth())

	pushnotifications.SetupJobs(jobsGroup, reg)
	agent.SetupJobs(jobsGroup, reg)
}

func SetupPublicComponents(r *gin.Engine, db *gorm.DB, auth authentication.Authenticator) {
	reg := registry.NewRegistry(db)
	reg.SetAuthenticator(auth)
	subscription.Setup(r, reg)
}

func SetupCleanArchComponents(r *gin.Engine, db *gorm.DB, auth authentication.Authenticator) {
	reg := registry.NewRegistry(db)
	reg.SetAuthenticator(auth)

	movement.Setup(r, reg)
	creditcard.Setup(r, reg)
	invoice.Setup(r, reg)
	transfer.Setup(r, reg)
	userpreferences.Setup(r, reg)
	userconsent.Setup(r, reg)
	export.Setup(r, reg)
	deleteaccount.Setup(r, reg)
	device.Setup(r, reg)
	limits.Setup(r, reg)
	admin.Setup(r, reg)
	agent.Setup(r, reg)
	statement.Setup(r, reg)
	category.Setup(r, reg)
	subcategory.Setup(r, reg)
	wallet.Setup(r, reg)
	estimate.Setup(r, reg)
	balance.Setup(r, reg)
}
