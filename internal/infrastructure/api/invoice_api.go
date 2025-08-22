package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	InvoiceUsecase interface {
		FindDetailedInvoicesByPeriod(ctx context.Context, period domain.Period) ([]domain.DetailedInvoice, error)
		FindByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.Invoice, error)
		Pay(ctx context.Context, id uuid.UUID, walletID uuid.UUID, paymentDate *time.Time) (domain.Invoice, error)
		RevertPayment(ctx context.Context, id uuid.UUID) (domain.Invoice, error)
	}
	InvoiceHandler struct {
		usecase InvoiceUsecase
	}

	PayInvoiceRequest struct {
		WalletID    uuid.UUID  `json:"wallet_id" binding:"required"`
		PaymentDate *time.Time `json:"payment_date,omitempty"`
	}
)

func NewInvoiceV2Handlers(r *gin.Engine, srv InvoiceUsecase) {
	handler := InvoiceHandler{
		usecase: srv,
	}

	invoiceGroup := r.Group("/v2/invoices")

	invoiceGroup.GET("/detailed", handler.FindDetailedInvoicesByPeriod())
	invoiceGroup.GET("/date", handler.FindByMonth())
	invoiceGroup.GET("/:id", handler.FindByID())
	invoiceGroup.POST("/:id/pay", handler.Pay())
	invoiceGroup.POST("/:id/revert-pay", handler.RevertPayment())
}

func (h InvoiceHandler) FindDetailedInvoicesByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		period, err := h.parsePeriod(c)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		invoices, err := h.usecase.FindDetailedInvoicesByPeriod(ctx, period)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		outputInvoices := make([]output.DetailedInvoiceOutput, len(invoices))
		for i, invoice := range invoices {
			outputInvoices[i] = output.ToDetailedInvoiceOutput(invoice)
		}

		c.JSON(http.StatusOK, outputInvoices)
	}
}

func (h InvoiceHandler) FindByMonth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var date time.Time
		var err error
		dateString := c.Query("date")
		if dateString != "" {
			date, err = time.Parse("2006-01-02", dateString)
			if err != nil {
				HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid from date format"))
				return
			}
		}

		invoices, err := h.usecase.FindByMonth(ctx, date)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		outputInvoices := make([]output.InvoiceOutput, len(invoices))
		for i, invoice := range invoices {
			outputInvoices[i] = output.ToInvoiceOutput(invoice)
		}

		c.JSON(http.StatusOK, outputInvoices)
	}
}

func (h InvoiceHandler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		invoice, err := h.usecase.FindByID(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToInvoiceOutput(invoice))
	}
}

func (h InvoiceHandler) Pay() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var request PayInvoiceRequest
		err = c.ShouldBindJSON(&request)
		if err != nil {
			err = domain.WrapInvalidInput(err, "error unmarshalling input")
			HandleErr(c, ctx, err)
			return
		}

		paidInvoice, err := h.usecase.Pay(ctx, id, request.WalletID, request.PaymentDate)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToInvoiceOutput(paidInvoice))
	}
}

func (h InvoiceHandler) RevertPayment() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		revertedInvoice, err := h.usecase.RevertPayment(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToInvoiceOutput(revertedInvoice))
	}
}

func (h InvoiceHandler) parsePeriod(c *gin.Context) (domain.Period, error) {
	var period domain.Period
	var err error

	fromString := c.Query("from")
	if fromString != "" {
		period.From, err = time.Parse("2006-01-02", fromString)
		if err != nil {
			return domain.Period{}, domain.WrapInvalidInput(err, "invalid from date format")
		}
	}

	toString := c.Query("to")
	if toString != "" {
		period.To, err = time.Parse("2006-01-02", toString)
		if err != nil {
			return domain.Period{}, domain.WrapInvalidInput(err, "invalid to date format")
		}
	}

	err = period.Validate()
	if err != nil {
		return domain.Period{}, domain.WrapInvalidInput(err, "invalid period")
	}

	return period, nil
}
