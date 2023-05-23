package main

import (
	"fmt"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	categApi "personal-finance/internal/domain/category/api"
	categRepository "personal-finance/internal/domain/category/repository"
	categService "personal-finance/internal/domain/category/service"
	movementApi "personal-finance/internal/domain/movement/api"
	movementRepository "personal-finance/internal/domain/movement/repository"
	movementService "personal-finance/internal/domain/movement/service"
	transactionApi "personal-finance/internal/domain/transaction/api"
	transactionRepository "personal-finance/internal/domain/transaction/repository"
	transactionService "personal-finance/internal/domain/transaction/service"
	transactionStatusApi "personal-finance/internal/domain/transactionstatus/api"
	transactionStatusRepository "personal-finance/internal/domain/transactionstatus/repository"
	transactionStatusService "personal-finance/internal/domain/transactionstatus/service"
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

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // TODO
	config.AllowHeaders = []string{"user_token", "Content-Type"}

	err := godotenv.Load(".env")
	if err != nil {
		return fmt.Errorf("error reading '.env' file: %w", err)
	}

	sessionControl := session.NewControl()
	authenticator := authentication.NewFirebaseAuth(sessionControl)
	r.Use(
		cors.New(config),
		authenticator.Authenticate())
	r.GET("/logout", authenticator.Logout())

	dataSourceName := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DATABASE"))

	db := database.OpenGORMConnection(dataSourceName)

	categoryRepo := categRepository.NewPgRepository(db)
	categoryService := categService.NewCategoryService(categoryRepo)
	categApi.NewCategoryHandlers(r, categoryService)

	walletRepo := walletRepository.NewPgRepository(db)
	walletService := walletService.NewWalletService(walletRepo)
	walletApi.NewWalletHandlers(r, walletService)

	typePaymentRepo := typePaymentRepository.NewPgRepository(db)
	typePaymentService := typePaymentService.NewTypePaymentService(typePaymentRepo)
	typePaymentApi.NewTypePaymentHandlers(r, typePaymentService)

	transactionStatusRepo := transactionStatusRepository.NewPgRepository(db)
	transactionStatusService := transactionStatusService.NewTransactionStatusService(transactionStatusRepo)
	transactionStatusApi.NewTransactionStatusHandlers(r, transactionStatusService)

	movementRepo := movementRepository.NewPgRepository(db, walletRepo)

	transactionRepo := transactionRepository.NewPgRepository(db, movementRepo, walletRepo)

	transactionService := transactionService.NewTransactionService(transactionRepo, movementRepo)

	movementService := movementService.NewMovementService(movementRepo, transactionService)
	movementApi.NewMovementHandlers(r, movementService)

	transactionApi.NewTransactionHandlers(r, movementService, transactionService)

	fmt.Println("connected")

	if err := r.Run(); err != nil {
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}
