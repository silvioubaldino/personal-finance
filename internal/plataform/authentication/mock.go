package authentication

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userToken", "userID")
	}
}
