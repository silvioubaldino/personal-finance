// Package app is the single composition root of the API: cmd/api and the acceptance
// suite (test/acceptance) both build the HTTP engine through New, changing only the
// injected dependencies (database and authenticator).
package app

import (
	"errors"
	"net/http"

	"personal-finance/internal/bootstrap"
	"personal-finance/internal/bootstrap/environment"
	"personal-finance/internal/bootstrap/registry"
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
	walletApi "personal-finance/internal/domain/wallet/api"
	walletRepository "personal-finance/internal/domain/wallet/repository"
	walletService "personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/plataform/health"
	"personal-finance/pkg/log"
	"personal-finance/pkg/metrics"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Config holds the dependencies that differ between production and the acceptance suite.
type Config struct {
	DB            *gorm.DB
	Authenticator authentication.Authenticator
	Logger        log.Logger
}

// New builds the complete *gin.Engine: recovery, logging, metrics, CORS, health, /ping,
// internal jobs, public routes, authentication + user provisioning, legacy and clean-arch
// routes.
func New(cfg Config) (*gin.Engine, error) {
	if cfg.DB == nil {
		return nil, errors.New("app: database is required")
	}
	if cfg.Authenticator == nil {
		return nil, errors.New("app: authenticator is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("app: logger is required")
	}

	r := setupGin(cfg)

	setupLegacyComponents(r, cfg.DB)

	bootstrap.SetupCleanArchComponents(r, cfg.DB, cfg.Authenticator)

	return r, nil
}

func setupGin(cfg Config) *gin.Engine {
	gin.DefaultWriter = log.NewLoggerWriter(cfg.Logger, log.InfoLevel)
	gin.DefaultErrorWriter = log.NewLoggerWriter(cfg.Logger, log.ErrorLevel)

	switch {
	case environment.IsProduction():
		gin.SetMode(gin.ReleaseMode)
	case environment.IsTest():
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()

	// Structured panic recovery (logs panic + stack, returns 500).
	r.Use(log.GinRecoveryMiddleware())

	r.Use(log.GinLoggerMiddleware(cfg.Logger))

	// HTTP server metrics (request count, duration, active requests).
	r.Use(metrics.HTTPMetricsMiddleware())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true // TODO
	corsConfig.AllowHeaders = []string{
		authentication.UserToken,
		authentication.APIKeyHeader,
		"Content-Type",
		"X-Request-ID",
	}
	r.Use(cors.New(corsConfig))

	// Liveness/readiness probes are unauthenticated, registered before auth.
	health.Register(r, cfg.DB)

	r.GET("/ping", ping())

	bootstrap.SetupInternalJobs(r, cfg.DB)

	bootstrap.SetupPublicComponents(r, cfg.DB, cfg.Authenticator)

	r.Use(cfg.Authenticator.Authenticate())
	r.Use(authentication.LazyProvisionUser(registry.NewRegistry(cfg.DB).GetUserRepository(), cfg.Authenticator.AuthClient()))

	return r
}

// setupLegacyComponents wires the features that still live under internal/domain/{feature}.
func setupLegacyComponents(r *gin.Engine, db *gorm.DB) {
	categoryRepo := categRepository.NewPgRepository(db)
	categoryService := categService.NewCategoryService(categoryRepo)
	categApi.NewCategoryHandlers(r, categoryService)

	walletRepo := walletRepository.NewPgRepository(db)
	walletLimitsValidator := registry.NewRegistry(db).GetPlanLimitsValidator()
	walletService := walletService.NewWalletService(walletRepo, walletLimitsValidator)
	walletApi.NewWalletHandlers(r, walletService)

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
}

func ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.InfoContext(c.Request.Context(), "Ping success")

		c.JSON(http.StatusOK, "pong")
	}
}
