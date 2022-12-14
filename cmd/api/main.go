package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	categApi "personal-finance/internal/domain/category/api"
	categRepository "personal-finance/internal/domain/category/repository"
	categService "personal-finance/internal/domain/category/service"
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
	"personal-finance/internal/plataform/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("error running app: %v", err)
	}
}

func run() error {
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	err := godotenv.Load(".env")
	if err != nil {
		return fmt.Errorf("error reading '.env' file: %w", err)
	}

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

	transactionRepo := transactionRepository.NewPgRepository(db)
	transactService := transactionService.NewTransactionService(transactionRepo, walletService)
	consolidatedService := transactionService.NewConsolidatedService(transactService, transactionRepo)
	transactionApi.NewTransactionHandlers(r, transactService, consolidatedService)

	fmt.Println("connected")

	if err := r.Run(); err != nil {
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}
