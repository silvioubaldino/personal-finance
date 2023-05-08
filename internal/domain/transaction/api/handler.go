package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	movementService "personal-finance/internal/domain/movement/service"
	transactionService "personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/plataform/session"
)

type handler struct {
	service        movementService.Movement
	transaction    transactionService.Transaction
	sessionControl session.Control
}

const (
	// Base URIs
	_transactions = "/transactions"
	_balance      = "/balance"

	// URIs
	_period = "/period"
)

func NewTransactionHandlers(r *gin.Engine, sessionControl session.Control, srv movementService.Movement, transaction transactionService.Transaction) {
	handler := handler{
		service:        srv,
		transaction:    transaction,
		sessionControl: sessionControl,
	}

	transactionGroup := r.Group(_transactions)

	// transactionGroup.GET("/", handler.FindAll())
	transactionGroup.GET("/:id", handler.FindByID())
	transactionGroup.GET("/period", handler.FindByPeriod())

	r.GET(_balance+_period, handler.BalanceByPeriod())
}

/*func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		transactions, err := h.service.FindAll(c.Request.Context())
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")

		id, err := uuid.Parse(idParam)
		if err != nil {
			handlerError(c, model.BuildErrValidation(fmt.Sprintf("id must be valid: %s", idParam)))
		}

		transaction, err := h.service.FindByID(c.Request.Context(), id)
		if err != nil {
			handlerError(c, err)
			return
		}
		c.JSON(http.StatusOK, transaction)
	}
}

func (h handler) FindByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		var period model.Period
		var err error
		if fromString := c.Query("from"); fromString != "" {
			period.From, err = time.Parse("2006-01-02", fromString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if toString := c.Query("to"); toString != "" {
			period.To, err = time.Parse("2006-01-02", toString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}

		err = period.Validate()
		if err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("period invalid: %s", err.Error()))
			return
		}

		transactions, err := h.service.FindByPeriod(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, transactions)
	}
}*/

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
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if toString := c.Query("to"); toString != "" {
			period.To, err = time.Parse("2006-01-02", toString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}

		err = period.Validate()
		if err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("period invalid: %s", err.Error()))
			return
		}

		transactions, err := h.transaction.FindByPeriod(c.Request.Context(), period, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		output := make([]model.TransactionOutput, len(transactions))

		for i, transaction := range transactions {
			output[i] = model.ToOutput(transaction)
		}
		c.JSON(http.StatusOK, output)
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
		c.JSON(http.StatusOK, parentTransaction)
	}
}

func (h handler) BalanceByPeriod() gin.HandlerFunc {
	return func(c *gin.Context) {
		var period model.Period
		var err error
		if fromString := c.Query("from"); fromString != "" {
			period.From, err = time.Parse("2006-01-02", fromString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if toString := c.Query("to"); toString != "" {
			period.To, err = time.Parse("2006-01-02", toString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, err.Error())
				return
			}
		}

		err = period.Validate()
		if err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("period invalid: %s", err.Error()))
			return
		}

		userID, err := h.getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}
		balance, err := h.service.BalanceByPeriod(c.Request.Context(), period, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, balance)
	}
}

func (h handler) getUserID(c *gin.Context) (string, error) {
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		return "", errors.New("user_token must`n be empty")
	}

	userID, err := h.sessionControl.Get(userToken)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func handlerError(c *gin.Context, err error) {
	var customError model.BusinessError
	if errors.As(err, &customError) {
		c.JSON(customError.HTTPCode, err.Error())
		return
	}
	c.JSON(http.StatusInternalServerError, model.BusinessError{Msg: "unexpected error"})
}
