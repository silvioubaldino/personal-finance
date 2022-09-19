package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/model"
)

var (
	now         = time.Now()
	walletsMock = []model.Wallet{
		{
			ID:          1,
			Description: "Nubank",
			Balance:     0,
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          2,
			Description: "Banco do brasil",
			Balance:     0,
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          3,
			Description: "Santander",
			Balance:     0,
			DateCreate:  now,
			DateUpdate:  now,
		},
	}
)

func TestPgRepository_Add(t *testing.T) {
	tt := []struct {
		name           string
		inputWallet    model.Wallet
		expectedWallet model.Wallet
		mockedErr      error
		expectedErr    error
		mockFunc       func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputWallet: model.Wallet{
				Description: walletsMock[0].Description,
				Balance:     walletsMock[0].Balance,
				DateCreate:  walletsMock[0].DateCreate,
				DateUpdate:  walletsMock[0].DateUpdate,
			},
			expectedWallet: walletsMock[0],
			mockedErr:      nil,
			expectedErr:    nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "wallets" ("description","balance","date_create","date_update") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "balance", "date_create", "date_update"}).
						AddRow(walletsMock[0].ID, walletsMock[0].Description, walletsMock[0].Balance, walletsMock[0].DateCreate, walletsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name: "error",
			inputWallet: model.Wallet{
				Description: "Nubank",
				DateCreate:  now,
				DateUpdate:  now,
			},
			expectedWallet: model.Wallet{},
			mockedErr:      errors.New("gorm error"),
			expectedErr:    errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "wallets" ("description","balance","date_create","date_update") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
					WillReturnError(errors.New("gorm error"))
				return db, mock, err
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, _, err := tc.mockFunc()
			require.NoError(t, err)
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: db,
			}), &gorm.Config{SkipDefaultTransaction: true})
			require.NoError(t, err)
			repo := repository.NewPgRepository(gormDB)

			result, err := repo.Add(context.Background(), tc.inputWallet)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallet, result)
		})
	}
}

func TestPgRepository_FindAll(t *testing.T) {
	tt := []struct {
		name            string
		expectedWallets []model.Wallet
		mockedErr       error
		expectedErr     error
		mockFunc        func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:            "success",
			expectedWallets: walletsMock,
			mockedErr:       nil,
			expectedErr:     nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "balance", "date_create", "date_update"}).
						AddRow(walletsMock[0].ID, walletsMock[0].Description, walletsMock[0].Balance, walletsMock[0].DateCreate, walletsMock[0].DateUpdate).
						AddRow(walletsMock[1].ID, walletsMock[1].Description, walletsMock[1].Balance, walletsMock[1].DateCreate, walletsMock[1].DateUpdate).
						AddRow(walletsMock[2].ID, walletsMock[2].Description, walletsMock[2].Balance, walletsMock[2].DateCreate, walletsMock[2].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:            "gorm error",
			expectedWallets: []model.Wallet{},
			mockedErr:       errors.New("gorm error"),
			expectedErr:     errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets`)).
					WillReturnError(errors.New("gorm error"))
				return db, mock, err
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, _, err := tc.mockFunc()
			require.NoError(t, err)
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: db,
			}), &gorm.Config{SkipDefaultTransaction: true})
			require.NoError(t, err)
			repo := repository.NewPgRepository(gormDB)

			result, err := repo.FindAll(context.Background())
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallets, result)
		})
	}
}

func TestPgRepository_FindByID(t *testing.T) {
	tt := []struct {
		name           string
		expectedWallet model.Wallet
		mockedErr      error
		expectedErr    error
		mockFunc       func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:           "success",
			expectedWallet: walletsMock[0],
			mockedErr:      nil,
			expectedErr:    nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" 
							WHERE "wallets"."id" = $1 
							ORDER BY "wallets"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "balance", "date_create", "date_update"}).
						AddRow(walletsMock[0].ID, walletsMock[0].Description, walletsMock[0].Balance, walletsMock[0].DateCreate, walletsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:           "gorm error",
			expectedWallet: model.Wallet{},
			mockedErr:      errors.New("gorm error"),
			expectedErr:    errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" 
							WHERE "wallets"."id" = $1
							ORDER BY "wallets"."id" LIMIT 1`)).
					WillReturnError(errors.New("gorm error"))
				return db, mock, err
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, _, err := tc.mockFunc()
			require.NoError(t, err)
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: db,
			}), &gorm.Config{SkipDefaultTransaction: true})
			require.NoError(t, err)
			repo := repository.NewPgRepository(gormDB)

			result, err := repo.FindByID(context.Background(), 1)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallet, result)
		})
	}
}

func TestPgRepository_Update(t *testing.T) {
	tt := []struct {
		name           string
		inputWallet    model.Wallet
		expectedWallet model.Wallet
		mockedErr      error
		expectedErr    error
		mockFunc       func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputWallet: model.Wallet{
				Description: walletsMock[0].Description,
			},
			expectedWallet: model.Wallet{
				ID:          2,
				Description: walletsMock[0].Description,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" 
							WHERE "wallets"."id" = $1 
							ORDER BY "wallets"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(walletsMock[1].ID,
							walletsMock[1].Description,
							walletsMock[1].DateCreate,
							walletsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "wallets" SET "description"=$1,"balance"=$2,"date_create"=$3,"date_update"=$4 
							WHERE "id" = $5`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:           "gorm error SELECT",
			expectedWallet: model.Wallet{},
			mockedErr:      errors.New("gorm error SELECT"),
			expectedErr:    errors.New("gorm error SELECT"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" 
							WHERE "wallets"."id" = $1 
							ORDER BY "wallets"."id" LIMIT 1`)).
					WillReturnError(errors.New("gorm error SELECT"))
				return db, mock, err
			},
		},
		{
			name: "gorm error UPDATE",
			inputWallet: model.Wallet{
				Description: walletsMock[0].Description,
			},
			expectedWallet: model.Wallet{},
			mockedErr:      errors.New("gorm error UPDATE"),
			expectedErr:    errors.New("gorm error UPDATE"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" 
							WHERE "wallets"."id" = $1 
							ORDER BY "wallets"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(walletsMock[1].ID,
							walletsMock[1].Description,
							walletsMock[1].DateCreate,
							walletsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "wallets" SET "description"=$1,"balance"=$2,"date_create"=$3,"date_update"=$4
							WHERE "id" = $5`)).
					WillReturnError(errors.New("gorm error UPDATE"))
				return db, mock, err
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, _, err := tc.mockFunc()
			require.NoError(t, err)
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: db,
			}), &gorm.Config{SkipDefaultTransaction: true})
			require.NoError(t, err)
			repo := repository.NewPgRepository(gormDB)

			result, err := repo.Update(context.Background(), 2, tc.inputWallet)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedWallet.ID, result.ID)
			require.Equal(t, tc.expectedWallet.Description, result.Description)
		})
	}
}

func TestPgRepository_Delete(t *testing.T) {
	tt := []struct {
		name        string
		mockedErr   error
		expectedErr error
		mockFunc    func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:        "success",
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec(regexp.QuoteMeta(
					`DELETE FROM "wallets" WHERE "wallets"."id" = $1`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:        "gorm error",
			mockedErr:   errors.New("gorm error"),
			expectedErr: errors.New("gorm error DELETE"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec(regexp.QuoteMeta(
					`DELETE FROM "wallets" WHERE "wallets"."id" = $1`)).
					WillReturnError(errors.New("gorm error DELETE"))
				return db, mock, err
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, _, err := tc.mockFunc()
			require.NoError(t, err)
			gormDB, err := gorm.Open(postgres.New(postgres.Config{
				Conn: db,
			}), &gorm.Config{SkipDefaultTransaction: true})
			require.NoError(t, err)
			repo := repository.NewPgRepository(gormDB)

			err = repo.Delete(context.Background(), 1)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
