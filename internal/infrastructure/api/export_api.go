package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"

	"github.com/gin-gonic/gin"
)

type (
	ExportUseCase interface {
		ExportUserData(ctx context.Context) (domain.UserDataExport, error)
	}

	ExportHandler struct {
		usecase ExportUseCase
	}
)

func NewExportHandlers(r *gin.Engine, srv ExportUseCase) {
	handler := ExportHandler{
		usecase: srv,
	}

	meGroup := r.Group("/me")

	meGroup.GET("/export", handler.ExportUserData())
}

func (h ExportHandler) ExportUserData() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		export, err := h.usecase.ExportUserData(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.Header("Content-Disposition", "attachment; filename=user_data_export.json")
		c.Header("Content-Type", "application/json")

		c.JSON(http.StatusOK, export)
	}
}
