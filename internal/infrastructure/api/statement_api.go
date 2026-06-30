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
		Extract(ctx context.Context, fileBytes []byte, mimeType, password, sourceType string) (domain.StatementExtractResult, error)
		Classify(ctx context.Context, input domain.StatementClassifyInput) (domain.StatementClassifyResult, error)
		Confirm(ctx context.Context, input domain.StatementConfirmInput) (domain.StatementConfirmResult, error)
		ConfirmInvoice(ctx context.Context, input domain.InvoiceConfirmInput) (domain.StatementConfirmResult, error)
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
	group.POST("/classify", handler.Classify())
	group.POST("/confirm", handler.Confirm())
	group.POST("/confirm-invoice", handler.ConfirmInvoice())
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

		// Optional: password to open a protected PDF, sent alongside the file.
		password := c.Request.FormValue("password")

		// Optional: client's declared intent ("statement" | "invoice"); absent = auto-detect.
		sourceType := c.Request.FormValue("source_type")

		result, err := h.usecase.Extract(ctx, fileBytes, mimeType, password, sourceType)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func (h StatementHandler) Classify() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var input domain.StatementClassifyInput
		if err := c.ShouldBindJSON(&input); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		result, err := h.usecase.Classify(ctx, input)
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

func (h StatementHandler) ConfirmInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var input domain.InvoiceConfirmInput
		if err := c.ShouldBindJSON(&input); err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid json body"))
			return
		}

		result, err := h.usecase.ConfirmInvoice(ctx, input)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
