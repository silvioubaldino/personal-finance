package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"

	"personal-finance/internal/domain/movement/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"personal-finance/internal/model"
)

var (
	mockedUUID = uuid.New()

	now               = time.Now()
	aluguelmockedTime = time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local)
	energiaMockedTime = time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local)
	aguaMockedTime    = time.Date(2022, time.September, 30, 0, 0, 0, 0, time.Local)
	movementsMock     = []model.Movement{
		{
			ID:               &mockedUUID,
			Description:      "Aluguel",
			Amount:           1000.0,
			Date:             &aluguelmockedTime,
			TransactionID:    &mockedUUID,
			WalletID:         1,
			TypePaymentID:    1,
			CategoryID:       2,
			MovementStatusID: 1,
			DateCreate:       now,
			DateUpdate:       now,
		},
		{
			ID:               &mockedUUID,
			Description:      "Energia",
			Amount:           300.0,
			Date:             &energiaMockedTime,
			TransactionID:    &mockedUUID,
			WalletID:         1,
			TypePaymentID:    1,
			CategoryID:       2,
			MovementStatusID: 1,
			DateCreate:       now,
			DateUpdate:       now,
		},
		{
			ID:               &mockedUUID,
			Description:      "Agua",
			Amount:           120.0,
			Date:             &aguaMockedTime,
			TransactionID:    &mockedUUID,
			WalletID:         1,
			TypePaymentID:    1,
			CategoryID:       2,
			MovementStatusID: 1,
			DateCreate:       now,
			DateUpdate:       now,
		},
	}
)

func TestPgRepository_Add(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Movement
		expectedTransaction model.Movement
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputTransaction: model.Movement{
				Description:      movementsMock[0].Description,
				Amount:           movementsMock[0].Amount,
				Date:             movementsMock[0].Date,
				TransactionID:    movementsMock[0].TransactionID,
				WalletID:         movementsMock[0].WalletID,
				TypePaymentID:    movementsMock[0].TypePaymentID,
				CategoryID:       movementsMock[0].CategoryID,
				MovementStatusID: movementsMock[0].MovementStatusID,
				DateCreate:       movementsMock[0].DateCreate,
				DateUpdate:       movementsMock[0].DateUpdate,
			},
			expectedTransaction: model.Movement{
				Description:      movementsMock[0].Description,
				Amount:           movementsMock[0].Amount,
				Date:             movementsMock[0].Date,
				TransactionID:    movementsMock[0].TransactionID,
				WalletID:         movementsMock[0].WalletID,
				Wallet:           movementsMock[0].Wallet,
				TypePaymentID:    movementsMock[0].TypePaymentID,
				TypePayment:      movementsMock[0].TypePayment,
				CategoryID:       movementsMock[0].CategoryID,
				Category:         movementsMock[0].Category,
				MovementStatusID: movementsMock[0].MovementStatusID,
				DateCreate:       movementsMock[0].DateCreate,
				DateUpdate:       movementsMock[0].DateUpdate,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "movements" ("id","description","amount","date","transaction_id","wallet_id","type_payment_id","category_id","movement_status_id","date_create","date_update") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				return db, mock, err
			},
		},
		{
			name: "error",
			inputTransaction: model.Movement{
				Description: "Aluguel",
				DateCreate:  now,
				DateUpdate:  now,
			},
			expectedTransaction: model.Movement{},
			mockedErr:           errors.New("gorm error"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec(regexp.QuoteMeta(
					`INSERT INTO "movements" ("id","description","amount","date","transaction_id","wallet_id","type_payment_id","category_id","movement_status_id","date_create","date_update") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`)).
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

			result, err := repo.Add(context.Background(), tc.inputTransaction)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction.Description, result.Description)
		})
	}
}

func TestPgRepository_FindByPeriod(t *testing.T) {
	tt := []struct {
		name                 string
		expectedTransactions []model.Movement
		mockedErr            error
		expectedErr          error
		mockFunc             func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			expectedTransactions: []model.Movement{
				movementsMock[0],
				movementsMock[1],
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE date BETWEEN $1 AND $2`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date",
						"transaction_id", "wallet_id",
						"type_payment_id", "category_id",
						"movement_status_id", "date_create", "date_update",
					}).
						AddRow(movementsMock[0].ID,
							movementsMock[0].Description,
							movementsMock[0].Amount,
							movementsMock[0].Date,
							movementsMock[0].TransactionID,
							movementsMock[0].WalletID,
							movementsMock[0].TypePaymentID,
							movementsMock[0].CategoryID,
							movementsMock[0].MovementStatusID,
							movementsMock[0].DateCreate,
							movementsMock[0].DateUpdate).
						AddRow(movementsMock[1].ID,
							movementsMock[1].Description,
							movementsMock[1].Amount,
							movementsMock[1].Date,
							movementsMock[1].TransactionID,
							movementsMock[1].WalletID,
							movementsMock[1].TypePaymentID,
							movementsMock[1].CategoryID,
							movementsMock[1].MovementStatusID,
							movementsMock[1].DateCreate,
							movementsMock[1].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                 "gorm error",
			expectedTransactions: []model.Movement{},
			mockedErr:            errors.New("gorm error"),
			expectedErr:          model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE date BETWEEN $1 AND $2`)).
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

			result, err := repo.FindByPeriod(context.Background(), model.Period{
				From: *movementsMock[0].Date,
				To:   *movementsMock[1].Date,
			})
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransactions, result)
		})
	}
}

func TestPgRepository_FindByID(t *testing.T) {
	tt := []struct {
		name                string
		expectedTransaction model.Movement
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:                "success",
			expectedTransaction: movementsMock[0],
			mockedErr:           nil,
			expectedErr:         nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE "movements"."id" = $1 ORDER BY "movements"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date",
						"transaction_id", "wallet_id",
						"type_payment_id", "category_id",
						"movement_status_id",
						"date_create", "date_update",
					}).
						AddRow(movementsMock[0].ID,
							movementsMock[0].Description,
							movementsMock[0].Amount,
							movementsMock[0].Date,
							movementsMock[0].TransactionID,
							movementsMock[0].WalletID,
							movementsMock[0].TypePaymentID,
							movementsMock[0].CategoryID,
							movementsMock[0].MovementStatusID,
							movementsMock[0].DateCreate,
							movementsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                "gorm error",
			expectedTransaction: model.Movement{},
			mockedErr:           errors.New("gorm error"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE "movements"."id" = $1 ORDER BY "movements"."id" LIMIT 1`)).
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

			result, err := repo.FindByID(context.Background(), mockedUUID)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestPgRepository_Update(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Movement
		expectedTransaction model.Movement
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputTransaction: model.Movement{
				Description:   movementsMock[0].Description,
				Amount:        movementsMock[0].Amount,
				Date:          movementsMock[0].Date,
				WalletID:      movementsMock[0].WalletID,
				TypePaymentID: movementsMock[0].TypePaymentID,
				CategoryID:    movementsMock[0].CategoryID,
			},
			expectedTransaction: model.Movement{
				ID:          &mockedUUID,
				Description: movementsMock[0].Description,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE "movements"."id" = $1 ORDER BY "movements"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date",
						"transaction_id", "id_wallet",
						"id_type_payment", "id_category",
						"movement_status_id", "date_create", "date_update",
					}).
						AddRow(movementsMock[0].ID,
							movementsMock[0].Description,
							movementsMock[0].Amount,
							movementsMock[0].Date,
							movementsMock[0].TransactionID,
							movementsMock[0].WalletID,
							movementsMock[0].TypePaymentID,
							movementsMock[0].CategoryID,
							movementsMock[0].MovementStatusID,
							movementsMock[0].DateCreate,
							movementsMock[0].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "movements" SET "description"=$1,"amount"=$2,"date"=$3,"transaction_id"=$4,"wallet_id"=$5,"type_payment_id"=$6,"category_id"=$7,"movement_status_id"=$8,"date_create"=$9,"date_update"=$10 WHERE "id" = $11`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:                "gorm error SELECT",
			expectedTransaction: model.Movement{},
			mockedErr:           errors.New("gorm error SELECT"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error SELECT")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE "movements"."id" = $1 ORDER BY "movements"."id" LIMIT 1`)).
					WillReturnError(errors.New("gorm error SELECT"))
				return db, mock, err
			},
		},
		{
			name: "gorm error UPDATE",
			inputTransaction: model.Movement{
				Description: movementsMock[0].Description,
			},
			expectedTransaction: model.Movement{},
			mockedErr:           errors.New("gorm error UPDATE"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error UPDATE")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE "movements"."id" = $1 ORDER BY "movements"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(movementsMock[1].ID,
							movementsMock[1].Description,
							movementsMock[1].DateCreate,
							movementsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "movements" SET "description"=$1,"date_create"=$2,"date_update"=$3 WHERE "id" = $4`)).
					WillReturnError(errors.New("gorm error UPDATE"))
				return db, mock, err
			},
		},
		{
			name:                "no changes error",
			inputTransaction:    model.Movement{},
			expectedTransaction: model.Movement{},
			mockedErr:           errors.New("no changes"),
			expectedErr:         model.BusinessError{Msg: "no changes", HTTPCode: http.StatusInternalServerError, Cause: errors.New("no changes")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "movements" WHERE "movements"."id" = $1 ORDER BY "movements"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(movementsMock[1].ID,
							movementsMock[1].Description,
							movementsMock[1].DateCreate,
							movementsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "transactions" SET "description"=$1,"date_create"=$2,"date_update"=$3 WHERE "id" = $4`)).
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

			result, err := repo.Update(context.Background(), mockedUUID, tc.inputTransaction)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction.ID, result.ID)
			require.Equal(t, tc.expectedTransaction.Description, result.Description)
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
					`DELETE FROM "movements" WHERE "movements"."id" = $1`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:        "gorm error",
			mockedErr:   errors.New("gorm error"),
			expectedErr: model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error DELETE")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectExec(regexp.QuoteMeta(
					`DELETE FROM "movements" WHERE "movements"."id" = $1`)).
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

			err = repo.Delete(context.Background(), mockedUUID)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
