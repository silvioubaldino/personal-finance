package subcategory

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	subCategoryRepo := reg.GetSubCategoryRepository()
	subCategoryService := usecase.NewSubCategory(subCategoryRepo)
	api.NewSubCategoryV2Handlers(r, subCategoryService)
}
