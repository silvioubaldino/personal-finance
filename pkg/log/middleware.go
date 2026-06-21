package log

import (
	"net/http"
	"runtime/debug"

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

// GinRecoveryMiddleware recovers from panics in handlers, logs the panic value
// and stack trace through the context-aware logger (so it carries request_id
// and trace correlation), and responds with a consistent 500. It replaces
// gin.Recovery() to ensure panics are captured by our structured logging.
func GinRecoveryMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				ErrorContext(c.Request.Context(), "panic recovered",
					Any("panic", rec),
					String("stack", string(debug.Stack())),
				)

				if !c.Writer.Written() {
					c.AbortWithStatusJSON(
						http.StatusInternalServerError,
						gin.H{"error": "internal server error"},
					)
				} else {
					c.Abort()
				}
			}
		}()

		c.Next()
	}
}
