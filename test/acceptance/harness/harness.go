//go:build acceptance

package harness

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"

	"gorm.io/gorm"

	"personal-finance/internal/app"
	"personal-finance/internal/bootstrap/environment"
	"personal-finance/internal/plataform/database"
	"personal-finance/pkg/log"
)

type Env struct {
	BaseURL string
	DB      *gorm.DB
	Client  *Client

	server  *httptest.Server
	cleanup func()
}

func Start(ctx context.Context) (*Env, error) {
	if environment.IsProduction() {
		return nil, errors.New("refusing to run the acceptance suite with ENVIRONMENT=production")
	}
	if err := os.Setenv("ENVIRONMENT", environment.Test); err != nil {
		return nil, err
	}

	logLevel := os.Getenv("ACCEPTANCE_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "fatal"
	}
	if err := os.Setenv("LOG_LEVEL", logLevel); err != nil {
		return nil, err
	}
	logger := log.New(log.WithLevel(logLevel), log.WithFormat("text"))
	log.SetGlobalLogger(logger)

	dsn, stopPostgres, err := startPostgres(ctx)
	if err != nil {
		return nil, err
	}

	if err := database.RunMigrations(dsn, migrationsURL()); err != nil {
		stopPostgres()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	db := database.OpenGORMConnection(dsn)

	engine, err := app.New(app.Config{DB: db, Authenticator: testAuth{}, Logger: logger})
	if err != nil {
		stopPostgres()
		return nil, fmt.Errorf("building the app: %w", err)
	}

	server := httptest.NewServer(withClock(engine))

	return &Env{
		BaseURL: server.URL,
		DB:      db,
		Client:  NewClient(server.URL),
		server:  server,
		cleanup: func() {
			server.Close()
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
			stopPostgres()
		},
	}, nil
}

func (e *Env) Close() {
	if e.cleanup != nil {
		e.cleanup()
	}
}
