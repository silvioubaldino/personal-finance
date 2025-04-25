package bootstrap

import (
	"personal-finance/internal/bootstrap/movement"
	"personal-finance/internal/bootstrap/registry"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupCleanArchComponents(r *gin.Engine, db *gorm.DB) {
	reg := registry.NewRegistry(db)

	movement.Setup(r, reg)
}
