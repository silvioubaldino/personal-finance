package main

import (
	"fmt"

	"personal-finance/docs"
	categApi "personal-finance/internal/domain/category/api"
	categRepository "personal-finance/internal/domain/category/repository"
	categService "personal-finance/internal/domain/category/service"
	transactionApi "personal-finance/internal/domain/transaction/api"
	transactionRepository "personal-finance/internal/domain/transaction/repository"
	transactionService "personal-finance/internal/domain/transaction/service"
	typePaymentApi "personal-finance/internal/domain/typepayment/api"
	typePaymentRepository "personal-finance/internal/domain/typepayment/repository"
	typePaymentService "personal-finance/internal/domain/typepayment/service"
	walletApi "personal-finance/internal/domain/wallet/api"
	walletRepository "personal-finance/internal/domain/wallet/repository"
	walletService "personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/plataform/database"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Personal finance API
// @version 1.0
// description Personal finance
func main() {
	if err := run(); err != nil {
		fmt.Printf("error running app: %v", err)
	}
}

func run() error {
	r := gin.Default()

	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	dataSourceName := "postgresql://admin:admin@pg-personal-finance:5432/personal_finance?sslmode=disable"
	// dataSourceNameLocalHost := "postgresql://admin:admin@localhost:5432/personal_finance?sslmode=disable"

	// To run with docker use "dataSourceName"
	// To run with IDE use "dataSourceNameLocalHost"
	db := database.OpenGORMConnection(dataSourceName)

	CategoryRepo := categRepository.NewPgRepository(db)
	CategoryService := categService.NewCategoryService(CategoryRepo)
	categApi.NewCategoryHandlers(r, CategoryService)

	WalletRepo := walletRepository.NewPgRepository(db)
	WalletService := walletService.NewWalletService(WalletRepo)
	walletApi.NewWalletHandlers(r, WalletService)

	TypePaymentRepo := typePaymentRepository.NewPgRepository(db)
	TypePaymentService := typePaymentService.NewTypePaymentService(TypePaymentRepo)
	typePaymentApi.NewTypePaymentHandlers(r, TypePaymentService)

	TransactionRepo := transactionRepository.NewPgRepository(db)
	TransactionService := transactionService.NewTransactionService(TransactionRepo)
	transactionApi.NewTransactionHandlers(r, TransactionService)

	fmt.Println("connected")

	if err := r.Run(); err != nil {
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}
