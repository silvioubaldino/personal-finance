package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	movementService "personal-finance/internal/domain/movement/service"
	transactionService "personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

type handler struct {
	service     movementService.Movement
	transaction transactionService.Transaction
}

const (
	_transactions = "/transactions"

	_period = "/period"
)

func NewTransactionHandlers(r *gin.Engine, srv movementService.Movement, transaction transactionService.Transaction) {
	handler := handler{
		service:     srv,
		transaction: transaction,
	}

	transactionGroup := r.Group(_transactions)

	transactionGroup.GET("/:id", handler.FindByID())
	transactionGroup.GET(_period, handler.FindByPeriod())
}

func (h handler) FindByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		var period model.Period
		if fromString := c.Query("from"); fromString != "" {
			period.From, err = time.Parse("2006-01-02", fromString)
			if err != nil {
				log.Printf("Error: %v", err)
				handlerError(c, model.BuildErrParsing(err))
				return
			}
		}
		if toString := c.Query("to"); toString != "" {
			period.To, err = time.Parse("2006-01-02", toString)
			if err != nil {
				log.Printf("Error: %v", err)
				handlerError(c, model.BuildErrParsing(err))
				return
			}
		}

		err = period.Validate()
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusBadRequest, fmt.Sprintf("period invalid: %s", err.Error()))
			return
		}

		transactions, err := h.transaction.FindByPeriod(c.Request.Context(), period, userID)
		if err != nil {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		outputTransaction := make([]model.TransactionOutput, len(transactions))
		for i, transaction := range transactions {
			outputTransaction[i] = model.ToTransactionOutput(transaction)
		}
		c.JSON(http.StatusOK, outputTransaction)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}
		parentTransaction, err := h.transaction.FindByID(c.Request.Context(), id, "userID")
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.ToTransactionOutput(parentTransaction))
	}
}

func handlerError(c *gin.Context, err error) {
	var customError model.BusinessError
	if errors.As(err, &customError) {
		c.JSON(customError.HTTPCode, err.Error())
		return
	}
	c.JSON(http.StatusInternalServerError, model.BusinessError{Msg: "unexpected error"})
}
