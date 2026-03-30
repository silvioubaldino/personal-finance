package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"personal-finance/internal/domain/wallet/repository"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

var (
	now  = time.Now()
	rid1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	rid2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	rid3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")

	walletsMock = []model.Wallet{
		{
			ID:          &rid1,
			Description: "Nubank",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &rid2,
			Description: "Banco do brasil",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &rid3,
			Description: "Santander",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
	}
)

func ctxWithUserID() context.Context {
	return context.WithValue(context.Background(), authentication.UserID, "userID")
}

func TestPgRepository_Add(t *testing.T) {
	tt := []struct {
		name        string
		inputWallet model.Wallet
		expectedErr error
		mockFunc    func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputWallet: model.Wallet{
				Description: walletsMock[0].Description,
				Balance:     walletsMock[0].Balance,
				UserID:      walletsMock[0].UserID,
			},
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "wallets"`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
				return db, mock, err
			},
		},
		{
			name: "error",
			inputWallet: model.Wallet{
				Description: "Nubank",
			},
			expectedErr: errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "wallets"`)).
					WillReturnError(errors.New("gorm error"))
				mock.ExpectRollback()
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
			}), &gorm.Config{})
			require.NoError(t, err)
			repo := repository.NewPgRepository(gormDB)

			_, err = repo.Add(ctxWithUserID(), tc.inputWallet)
			if tc.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
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
					`SELECT * FROM "wallets" WHERE user_id=`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "balance", "user_id", "date_create", "date_update"}).
						AddRow(walletsMock[0].ID, walletsMock[0].Description, walletsMock[0].Balance, walletsMock[0].UserID, walletsMock[0].DateCreate, walletsMock[0].DateUpdate).
						AddRow(walletsMock[1].ID, walletsMock[1].Description, walletsMock[1].Balance, walletsMock[1].UserID, walletsMock[1].DateCreate, walletsMock[1].DateUpdate).
						AddRow(walletsMock[2].ID, walletsMock[2].Description, walletsMock[2].Balance, walletsMock[2].UserID, walletsMock[2].DateCreate, walletsMock[2].DateUpdate))
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

			result, err := repo.FindAll(ctxWithUserID())
			require.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				require.Equal(t, tc.expectedWallets, result)
			}
		})
	}
}

func TestPgRepository_FindByID(t *testing.T) {
	tt := []struct {
		name           string
		inputID        *uuid.UUID
		expectedWallet model.Wallet
		mockedErr      error
		expectedErr    error
		mockFunc       func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:           "success",
			inputID:        &rid1,
			expectedWallet: walletsMock[0],
			mockedErr:      nil,
			expectedErr:    nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" WHERE user_id=`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "balance", "user_id", "date_create", "date_update"}).
						AddRow(walletsMock[0].ID, walletsMock[0].Description, walletsMock[0].Balance, walletsMock[0].UserID, walletsMock[0].DateCreate, walletsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:           "gorm error",
			inputID:        &rid1,
			expectedWallet: model.Wallet{},
			mockedErr:      errors.New("gorm error"),
			expectedErr:    errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" WHERE user_id=`)).
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

			result, err := repo.FindByID(ctxWithUserID(), tc.inputID)
			if tc.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedWallet, result)
			}
		})
	}
}

func TestPgRepository_Update(t *testing.T) {
	tt := []struct {
		name           string
		inputWallet    model.Wallet
		expectedWallet model.Wallet
		inputID        *uuid.UUID
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
				ID:          &rid2,
				Description: walletsMock[0].Description,
			},
			inputID:     &rid2,
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" WHERE user_id=`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(walletsMock[1].ID,
							walletsMock[1].Description,
							walletsMock[1].UserID,
							walletsMock[1].DateCreate,
							walletsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "wallets" SET`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:           "gorm error SELECT",
			inputID:        &rid2,
			expectedWallet: model.Wallet{},
			mockedErr:      errors.New("gorm error SELECT"),
			expectedErr:    errors.New("gorm error SELECT"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "wallets" WHERE user_id=`)).
					WillReturnError(errors.New("gorm error SELECT"))
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

			result, err := repo.Update(ctxWithUserID(), tc.inputID, tc.inputWallet)
			if tc.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedWallet.ID, result.ID)
				require.Equal(t, tc.expectedWallet.Description, result.Description)
			}
		})
	}
}

func TestPgRepository_Delete(t *testing.T) {
	tt := []struct {
		name        string
		inputID     *uuid.UUID
		mockedErr   error
		expectedErr error
		mockFunc    func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:        "success",
			inputID:     &rid1,
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
			inputID:     &rid1,
			mockedErr:   errors.New("gorm error DELETE"),
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

			err = repo.Delete(context.Background(), tc.inputID)
			if tc.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
