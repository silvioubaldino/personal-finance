package main

import (
	"fmt"
	"net/http"
	"os"

	"personal-finance/internal/bootstrap"
	"personal-finance/internal/bootstrap/environment"
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
	"personal-finance/pkg/log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
		logLevel = "json"
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "text"
	}

	logger := log.New(
		log.WithLevel(logLevel),
		log.WithFormat(logFormat),
	)

	log.SetGlobalLogger(logger)

	return logger
}

func setupGin(logger log.Logger) *gin.Engine {
	r := gin.New()
	if environment.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r.Use(gin.Recovery())

	r.Use(log.GinLoggerMiddleware(logger))

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true // TODO
	corsConfig.AllowHeaders = []string{authentication.UserToken, "Content-Type"}
	r.Use(cors.New(corsConfig))

	sessionControl := session.NewControl()
	authenticator := authentication.NewFirebaseAuth(sessionControl)
	r.Use(authenticator.Authenticate())

	r.GET("/ping", ping())
	r.GET("/logout", authenticator.Logout())

	return r
}

func run() error {
	logger := configureLogger()

	err := godotenv.Load(".env")
	if err != nil {
		log.Error("error reading '.env' file:", log.Err(err))
	}

	r := setupGin(logger)

	db := database.InitializeDatabase()

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

	bootstrap.SetupCleanArchComponents(r, db)

	if err := r.Run(); err != nil {
		log.Error("error running web application", log.Err(err))
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}

func ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.InfoContext(c.Request.Context(), "Ping success")

		c.JSON(http.StatusOK, "pong")
	}
}
