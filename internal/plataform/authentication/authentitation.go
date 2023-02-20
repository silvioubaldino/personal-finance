package authentication

import (
	"github.com/gin-gonic/gin"
)

type Auth interface {
	Middleware() gin.HandlerFunc
}
