package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/domain/transactionstatus/service"
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
		c.JSON(http.StatusOK, transactionStatus)
	}
}
