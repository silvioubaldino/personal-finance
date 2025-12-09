package api

import (
	"context"
	"net/http"

	"personal-finance/internal/domain"
	"personal-finance/internal/domain/output"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	CreditCardUsecase interface {
		Add(ctx context.Context, creditCard domain.CreditCard) (domain.CreditCard, error)
		FindAll(ctx context.Context) ([]domain.CreditCard, error)
		FindByID(ctx context.Context, id uuid.UUID) (domain.CreditCard, error)
		FindWithOpenInvoices(ctx context.Context) ([]domain.CreditCardWithOpenInvoices, error)
		Update(ctx context.Context, id uuid.UUID, creditCard domain.CreditCard) (domain.CreditCard, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}
	CreditCardHandler struct {
		usecase CreditCardUsecase
	}
)

func NewCreditCardV2Handlers(r *gin.Engine, srv CreditCardUsecase) {
	handler := CreditCardHandler{
		usecase: srv,
	}

	creditCardGroup := r.Group("/v2/creditcards")

	creditCardGroup.POST("/", handler.Add())
	creditCardGroup.GET("/", handler.FindAll())
	creditCardGroup.GET("/:id", handler.FindByID())
	creditCardGroup.GET("/with-open-invoices", handler.FindWithOpenInvoices())
	creditCardGroup.PUT("/:id", handler.Update())
	creditCardGroup.DELETE("/:id", handler.Delete())
}

func (h CreditCardHandler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var creditCard domain.CreditCard

		err := c.ShouldBindJSON(&creditCard)
		if err != nil {
			err = domain.WrapInvalidInput(err, "error unmarshalling input")
			HandleErr(c, ctx, err)
			return
		}

		savedCreditCard, err := h.usecase.Add(ctx, creditCard)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusCreated, output.ToCreditCardOutput(savedCreditCard))
	}
}

func (h CreditCardHandler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		creditCards, err := h.usecase.FindAll(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		outputCreditCards := make([]output.CreditCardOutput, len(creditCards))
		for i, creditCard := range creditCards {
			outputCreditCards[i] = output.ToCreditCardOutput(creditCard)
		}

		c.JSON(http.StatusOK, outputCreditCards)
	}
}

func (h CreditCardHandler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		creditCard, err := h.usecase.FindByID(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToCreditCardOutput(creditCard))
	}
}

func (h CreditCardHandler) FindWithOpenInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		creditCards, err := h.usecase.FindWithOpenInvoices(ctx)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		outputCreditCards := make([]output.CreditCardWithOpenInvoicesOutput, len(creditCards))
		for i, creditCard := range creditCards {
			outputCreditCards[i] = output.ToCreditCardWithOpenInvoicesOutput(creditCard)
		}

		c.JSON(http.StatusOK, outputCreditCards)
	}
}

func (h CreditCardHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		var creditCard domain.CreditCard
		err = c.ShouldBindJSON(&creditCard)
		if err != nil {
			err = domain.WrapInvalidInput(err, "error unmarshalling input")
			HandleErr(c, ctx, err)
			return
		}

		updatedCreditCard, err := h.usecase.Update(ctx, id, creditCard)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, output.ToCreditCardOutput(updatedCreditCard))
	}
}

func (h CreditCardHandler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			HandleErr(c, ctx, domain.WrapInvalidInput(err, "id must be valid"))
			return
		}

		err = h.usecase.Delete(ctx, id)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}
