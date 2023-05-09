package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/transactionstatus/service"
	"personal-finance/internal/model"
)

type handler struct {
	srv service.Service
}

func NewTransactionStatusHandlers(r *gin.Engine, srv service.Service) {
	handler := handler{srv: srv}

	r.GET("/transactionStatus", handler.FindAll())
}

func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		transactionStatus, err := h.srv.FindAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		outputStatus := make([]model.TransactionStatusOutput, len(transactionStatus))
		for i, status := range transactionStatus {
			outputStatus[i] = model.ToTransactionStatusOutput(status)
		}
		c.JSON(http.StatusOK, outputStatus)
	}
}
