package authentication

import (
	"context"
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

const (
	UserID    = "user_id"
	UserToken = "user_token"
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

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	authClient, err := app.Auth(ctx)
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
		userToken := c.GetHeader(UserToken)
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
				c.Abort()
				return
			}
			f.sessionControl.Set(userToken, userID)
		}

		ctx := context.WithValue(c.Request.Context(), UserID, userID)
		c.Request = c.Request.WithContext(ctx)
	}
}

func (f firebaseAuth) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader(UserToken)
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
