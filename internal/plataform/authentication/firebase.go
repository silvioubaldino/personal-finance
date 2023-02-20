package authentication

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type firebase struct{}

func NewFirebaseAuth() Auth {
	return firebase{}
}

func (f firebase) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("user_token")
		_, err := f.validToken(userToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}
	}
}

func (f firebase) validToken(key string) (string, error) {
	return "userID", nil
}
