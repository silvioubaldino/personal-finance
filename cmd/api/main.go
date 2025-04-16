package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"

	balanceApi "personal-finance/internal/domain/balance/api"
	balanceService "personal-finance/internal/domain/balance/service"
	categApi "personal-finance/internal/domain/category/api"
	categRepository "personal-finance/internal/domain/category/repository"
	categService "personal-finance/internal/domain/category/service"
	estimateApi "personal-finance/internal/domain/estimate/api"
	estimateRepository "personal-finance/internal/domain/estimate/repository"
	estimateService "personal-finance/internal/domain/estimate/service"
	movementApi "personal-finance/internal/domain/movement/api"
	movementRepository "personal-finance/internal/domain/movement/repository"
	movementService "personal-finance/internal/domain/movement/service"
	recurrentRepository "personal-finance/internal/domain/recurrentmovement/repository"
	subCategoryApi "personal-finance/internal/domain/subcategory/api"
	subCategoryRepository "personal-finance/internal/domain/subcategory/repository"
	typePaymentApi "personal-finance/internal/domain/typepayment/api"
	typePaymentRepository "personal-finance/internal/domain/typepayment/repository"
	typePaymentService "personal-finance/internal/domain/typepayment/service"
	walletApi "personal-finance/internal/domain/wallet/api"
	walletRepository "personal-finance/internal/domain/wallet/repository"
	walletService "personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/plataform/database"
	"personal-finance/internal/plataform/session"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("error running app: %v", err)
	}
}

func run() error {
	r := gin.Default()

	r.GET("/ping", ping())

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // TODO
	config.AllowHeaders = []string{"user_token", "Content-Type"}

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("error reading '.env' file: %w\n", err)
	}

	sessionControl := session.NewControl()
	authenticator := authentication.NewFirebaseAuth(sessionControl)
	r.Use(
		cors.New(config),
		authenticator.Authenticate())
	r.GET("/logout", authenticator.Logout())

	dataSourceName := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DATABASE"))

	db := database.OpenGORMConnection(dataSourceName)

	if err := runMigrations(dataSourceName); err != nil {
		log.Fatalf("could not run migrations: %v", err)
	}

	categoryRepo := categRepository.NewPgRepository(db)
	categoryService := categService.NewCategoryService(categoryRepo)
	categApi.NewCategoryHandlers(r, categoryService)

	walletRepo := walletRepository.NewPgRepository(db)
	walletService := walletService.NewWalletService(walletRepo)
	walletApi.NewWalletHandlers(r, walletService)

	typePaymentRepo := typePaymentRepository.NewPgRepository(db)
	typePaymentService := typePaymentService.NewTypePaymentService(typePaymentRepo)
	typePaymentApi.NewTypePaymentHandlers(r, typePaymentService)

	recurrentRepo := recurrentRepository.NewRecurrentRepository(db)

	movementRepo := movementRepository.NewPgRepository(db, walletRepo, recurrentRepo)

	subCategoryRepo := subCategoryRepository.NewPgRepository(db)
	subCategoryApi.NewSubCategoryHandlers(r, subCategoryRepo)

	estimateRepo := estimateRepository.NewPgRepository(db, subCategoryRepo)
	estimateService := estimateService.NewEstimateService(estimateRepo)
	estimateApi.NewBalanceHandlers(r, estimateService)

	balanceService := balanceService.NewBalanceService(movementRepo, estimateRepo)
	balanceApi.NewBalanceHandlers(r, balanceService)

	movementService := movementService.NewMovementService(movementRepo, subCategoryRepo, recurrentRepo)
	movementApi.NewMovementHandlers(r, movementService)

	fmt.Println("connected")

	if err := r.Run(); err != nil {
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}

func runMigrations(dataSourceName string) error {
	m, err := migrate.New(
		"file://../../db/migrations/",
		dataSourceName)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not run up migrations: %w", err)
	}

	return nil
}

func ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	}
}
