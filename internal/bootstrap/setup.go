package bootstrap

import (
	"personal-finance/internal/bootstrap/creditcard"
	"personal-finance/internal/bootstrap/invoice"
	"personal-finance/internal/bootstrap/movement"
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/bootstrap/transfer"
	"personal-finance/internal/bootstrap/userpreferences"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupCleanArchComponents(r *gin.Engine, db *gorm.DB) {
	reg := registry.NewRegistry(db)

	movement.Setup(r, reg)
	creditcard.Setup(r, reg)
	invoice.Setup(r, reg)
	transfer.Setup(r, reg)
	userpreferences.Setup(r, reg)
}
