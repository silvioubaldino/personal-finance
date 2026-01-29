package authentication

import (
	"context"

	"firebase.google.com/go/v4/auth"
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

func (m *Mock) DeleteUser(_ context.Context, _ string) error {
	return nil
}

func (m *Mock) AuthClient() *auth.Client {
	return nil
}
