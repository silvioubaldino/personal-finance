package suite

import (
	"context"
	"database/sql"
	"fmt"
	"net/http/httptest"
	"os"
	"time"

	"github.com/cucumber/godog"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"personal-finance/internal/domain/movement/api"
	movementRepository "personal-finance/internal/domain/movement/repository"
	"personal-finance/internal/domain/movement/service"
	recurrentRepository "personal-finance/internal/domain/recurrentmovement/repository"
	subCategoryRepository "personal-finance/internal/domain/subcategory/repository"
	walletApi "personal-finance/internal/domain/wallet/api"
	walletRepository "personal-finance/internal/domain/wallet/repository"
	walletService "personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

type BaseData struct {
	TestUserID         string
	DefaultWallet      model.Wallet
	DefaultCategory    model.Category
	DefaultSubCategory model.SubCategory
	DefaultTypePayment model.TypePayment
}

type TestSuite struct {
	db       *sql.DB
	gormDB   *gorm.DB
	server   *httptest.Server
	baseData *BaseData
}

func (s *TestSuite) InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		gin.SetMode(gin.TestMode)
		r := gin.Default()

		dbFile := "/tmp/test_personal_finance.db"

		db, err := sql.Open("sqlite3", dbFile)
		if err != nil {
			panic(fmt.Sprintf("erro ao inicializar banco de dados: %v", err))
		}
		s.db = db

		gormDB, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
		if err != nil {
			panic(fmt.Sprintf("erro ao inicializar GORM: %v", err))
		}
		s.gormDB = gormDB

		err = s.runMigrations()
		if err != nil {
			panic(fmt.Sprintf("erro ao executar migrations: %v", err))
		}

		s.baseData, err = s.createBaseData()
		if err != nil {
			panic(fmt.Sprintf("erro ao criar dados base: %v", err))
		}

		s.setupHandlers(r)
		s.server = httptest.NewServer(r)
	})

	ctx.AfterSuite(func() {
		if s.server != nil {
			s.server.Close()
		}
		if s.db != nil {
			s.db.Close()
		}
		os.Remove("/tmp/test_personal_finance.db")
	})
}

func (s *TestSuite) runMigrations() error {
	return s.gormDB.AutoMigrate(
		&model.Movement{},
		&model.RecurrentMovement{},
		&model.Wallet{},
		&model.Category{},
		&model.SubCategory{},
	)
}

func (s *TestSuite) createBaseData() (*BaseData, error) {
	testUserID := "test-user-123"
	now := time.Now()

	walletID := uuid.New()
	wallet := model.Wallet{
		ID:             &walletID,
		Description:    "Test Wallet",
		Balance:        10000.00,
		UserID:         testUserID,
		InitialBalance: 10000.00,
		InitialDate:    now,
		DateCreate:     now,
		DateUpdate:     now,
	}

	categoryID := uuid.New()
	category := model.Category{
		ID:          &categoryID,
		Description: "Test Category",
		UserID:      testUserID,
		IsIncome:    false,
		DateCreate:  now,
		DateUpdate:  now,
	}

	subCategoryID := uuid.New()
	subCategory := model.SubCategory{
		ID:          &subCategoryID,
		Description: "Test SubCategory",
		UserID:      testUserID,
		CategoryID:  &categoryID,
		DateCreate:  now,
		DateUpdate:  now,
	}

	if err := s.gormDB.Create(&wallet).Error; err != nil {
		return nil, err
	}
	if err := s.gormDB.Create(&category).Error; err != nil {
		return nil, err
	}
	if err := s.gormDB.Create(&subCategory).Error; err != nil {
		return nil, err
	}

	return &BaseData{
		TestUserID:         testUserID,
		DefaultWallet:      wallet,
		DefaultCategory:    category,
		DefaultSubCategory: subCategory,
	}, nil
}

func (s *TestSuite) setupHandlers(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), authentication.UserID, s.baseData.TestUserID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	subCategoryRepo := subCategoryRepository.NewPgRepository(s.gormDB)
	recurrentRepo := recurrentRepository.NewRecurrentRepository(s.gormDB)
	walletRepo := walletRepository.NewPgRepository(s.gormDB)
	movementRepo := movementRepository.NewPgRepository(s.gormDB, walletRepo, recurrentRepo)

	// Movement handlers
	movementService := service.NewMovementService(movementRepo, subCategoryRepo, recurrentRepo)
	api.NewMovementHandlers(r, movementService)

	// Wallet handlers
	walletSrv := walletService.NewWalletService(walletRepo)
	walletApi.NewWalletHandlers(r, walletSrv)
}

func (s *TestSuite) GetServer() *httptest.Server {
	return s.server
}

func (s *TestSuite) GetBaseData() *BaseData {
	return s.baseData
}

func (s *TestSuite) CreateTestContext() context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, authentication.UserID, s.baseData.TestUserID)
}

func (s *TestSuite) BeforeScenario(sc *godog.Scenario) {
	movementTables := []string{
		"movements",
		"recurrent_movements",
	}

	for _, table := range movementTables {
		result := s.gormDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if result.Error != nil {
			panic(fmt.Sprintf("erro ao limpar tabela %s: %v", table, result.Error))
		}
	}

	s.gormDB.Model(&s.baseData.DefaultWallet).Update("balance", 10000.00)
}
