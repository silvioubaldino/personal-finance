package authentication

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"

	"personal-finance/internal/model"
	"personal-finance/internal/plataform/session"
)

type Authenticator interface {
	Authenticate() gin.HandlerFunc
	Logout() gin.HandlerFunc
}

type firebaseAuth struct {
	authClient     *auth.Client
	sessionControl session.Control
}

func NewFirebaseAuth(sessionControl session.Control) Authenticator {
	projectID := os.Getenv("GOOGLE_PROJECT_ID")
	config := &firebase.Config{ProjectID: projectID}

	app, err := firebase.NewApp(context.Background(), config)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	authClient, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("error getting Auth Client: %v\n", err)
	}
	firebaseAuth := firebaseAuth{
		authClient:     authClient,
		sessionControl: sessionControl,
	}

	return firebaseAuth
}

func (f firebaseAuth) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("user_token")
		if userToken == "" {
			log.Printf("Error: %v", model.ErrEmptyToken)
			c.JSON(http.StatusUnauthorized, model.ErrEmptyToken.Error())
			c.Abort()
			return
		}

		userID, err := f.sessionControl.Get(userToken)
		if err != nil {
			userID, err = f.verifyIDToken(c, userToken)
			if err != nil {
				c.JSON(http.StatusUnauthorized, err.Error())
				return
			}
			f.sessionControl.Set(userToken, userID)
		}

		c.Set(userToken, userID)
	}
}

func GetUserIDFromContext(c *gin.Context) (string, error) {
	userToken := c.GetHeader("user_token")
	if userToken == "" {
		return "", model.ErrEmptyToken
	}
	userID, ok := c.Get(userToken)
	if !ok {
		return "", errors.New("user_id not found")
	}

	return fmt.Sprint(userID), nil
}

func (f firebaseAuth) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("user_token")
		if userToken != "" {
			c.JSON(http.StatusUnauthorized, model.ErrEmptyToken)
			return
		}
		f.sessionControl.Delete(userToken)
	}
}

func (f firebaseAuth) verifyIDToken(ctx context.Context, token string) (string, error) {
	userID, err := f.authClient.VerifyIDToken(ctx, token)
	if err != nil {
		return "", fmt.Errorf("error verifying ID token: internal error")
	}
	return userID.UID, nil
}
