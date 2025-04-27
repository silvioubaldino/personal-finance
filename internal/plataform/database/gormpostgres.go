package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"personal-finance/internal/bootstrap/environment"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitializeDatabase() *gorm.DB {
	dataSourceName := BuildConnectionString()

	if err := RunMigrations(dataSourceName, GetMigrationsPath()); err != nil {
		fmt.Printf("warning: could not run migrations: %v\n", err)
	}

	return OpenGORMConnection(dataSourceName)
}

func BuildConnectionString() string {
	sslMode := "require"
	if os.Getenv("ENVIRONMENT") == "local" {
		sslMode = "disable"
	}

	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DATABASE"),
		sslMode,
	)
}

func GetMigrationsPath() string {
	env := environment.GetEnvironment()
	if env == environment.Production || env == environment.Staging {
		return "file://../../db/migrations/"
	}

	return "file://db/migrations/"
}

func OpenGORMConnection(dataSourceName string) *gorm.DB {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		})
	gormDB, err := gorm.Open(postgres.Open(dataSourceName), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true,
	})
	if err != nil {
		log.Fatalf("could not create gorm connection: %s", err)
	}
	return gormDB
}

func RunMigrations(dataSourceName string, migrationsPath string) error {
	m, err := migrate.New(
		migrationsPath,
		dataSourceName)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	return nil
}
