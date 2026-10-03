package main

import (
	"context"
	"fmt"
	"os"

	"personal-finance/internal/app"
	"personal-finance/internal/bootstrap/environment"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/plataform/database"
	"personal-finance/pkg/log"
	"personal-finance/pkg/metrics"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("error running app: %v", err)
	}
}

func configureLogger() log.Logger {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		// Production emits JSON so Google Cloud Logging parses severity and
		// structured fields; other environments keep human-readable text.
		if environment.IsProduction() {
			logFormat = "json"
		} else {
			logFormat = "text"
		}
	}

	logger := log.New(
		log.WithLevel(logLevel),
		log.WithFormat(logFormat),
	)

	log.SetGlobalLogger(logger)

	return logger
}

func run() error {
	err := godotenv.Load(".env")
	if err != nil {
		log.Error("error reading '.env' file:", log.Err(err))
	}

	logger := configureLogger()

	ctx := context.Background()
	shutdownMetrics, err := metrics.InitMeterProvider(ctx)
	if err != nil {
		log.Error("error initializing meter provider", log.Err(err))
	} else {
		defer func() {
			if err := shutdownMetrics(context.Background()); err != nil {
				log.Error("error shutting down meter provider", log.Err(err))
			}
		}()
	}

	db := database.InitializeDatabase()

	r, err := app.New(app.Config{
		DB:            db,
		Authenticator: authentication.NewFirebaseAuth(),
		Logger:        logger,
	})
	if err != nil {
		return fmt.Errorf("error building application: %w", err)
	}

	log.Info("application started")

	if err := r.Run(); err != nil {
		log.Error("error running web application", log.Err(err))
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}
