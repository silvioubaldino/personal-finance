package api

import (
	"context"
	"io"
	"net/http"

	"personal-finance/internal/domain"

	"github.com/gin-gonic/gin"
)

type (
	StatementUsecase interface {
		Extract(ctx context.Context, fileBytes []byte, mimeType string) (domain.StatementExtractResult, error)
		Confirm(ctx context.Context, input domain.StatementConfirmInput) (domain.StatementConfirmResult, error)
	}

	StatementHandler struct {
		usecase StatementUsecase
	}
)

func NewStatementHandlers(r *gin.Engine, srv StatementUsecase) {
	handler := StatementHandler{
		usecase: srv,
	}

	group := r.Group("/v2/statements")

	group.POST("/extract", handler.Extract())
	group.POST("/confirm", handler.Confirm())
}

func (h StatementHandler) Extract() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "file is required"))
			return
		}
		defer file.Close()

		if header.Size > int64(domain.MaxStatementFileBytes) {
			HandleErr(c, ctx, domain.ErrStatementFileTooLarge)
			return
		}

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInternalError(err, "error reading file"))
			return
		}

		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" || mimeType == "application/octet-stream" {
			mimeType = http.DetectContentType(fileBytes)
		}

		result, err := h.usecase.Extract(ctx, fileBytes, mimeType)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func (h StatementHandler) Confirm() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var input domain.StatementConfirmInput
		if err := c.ShouldBindJSON(&input); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		result, err := h.usecase.Confirm(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
