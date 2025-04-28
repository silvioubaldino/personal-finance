package log

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	_requestIDHeader = "X-Request-ID"
	_debugHeader     = "X-Debug"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

func GinLoggerMiddleware(logger Logger) func(c *gin.Context) {
	return func(c *gin.Context) {
		l := logger

		if c.GetHeader(_debugHeader) == "true" {
			l = New(WithLevel("debug"))
		}

		requestID := c.GetHeader(_requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header(_requestIDHeader, requestID)
		}

		l = l.With(String("request_id", requestID))

		l = l.With(
			String("method", c.Request.Method),
			String("path", c.Request.URL.Path),
		)

		l.Info("request")

		ctx := Context(c.Request.Context(), l)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		l.Info("response",
			Int("status", c.Writer.Status()),
		)
	}
}
