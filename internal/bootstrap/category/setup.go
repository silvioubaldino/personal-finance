package category

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	categoryRepo := reg.GetCategoryRepository()
	categoryService := usecase.NewCategory(categoryRepo)
	api.NewCategoryV2Handlers(r, categoryService)
}
