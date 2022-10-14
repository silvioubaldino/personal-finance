package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"testing"
	"time"

	"personal-finance/internal/domain/transaction/repository"
	"personal-finance/internal/model/eager"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"personal-finance/internal/model"
)

var (
	now              = time.Now()
	transactionsMock = []model.Transaction{
		{
			ID:            1,
			Description:   "Aluguel",
			Amount:        1000.0,
			Date:          time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
		{
			ID:            2,
			Description:   "Energia",
			Amount:        300.0,
			Date:          time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
		{
			ID:            3,
			Description:   "Agua",
			Amount:        120.0,
			Date:          time.Date(2022, time.September, 30, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    now,
			DateUpdate:    now,
		},
	}
	transactionEagerMock = eager.Transaction{
		ID:          1,
		Description: "Aluguel",
		Amount:      1000.0,
		Date:        now,
		WalletID:    0,
		Wallet: model.Wallet{
			ID:          1,
			Description: "Alimentacao",
			Balance:     0,
			DateCreate:  now,
			DateUpdate:  now,
		},
		TypePaymentID: 0,
		TypePayment: model.TypePayment{
			ID:          1,
			Description: "Débito",
			DateCreate:  now,
			DateUpdate:  now,
		},
		CategoryID: 0,
		Category: model.Category{
			ID:          2,
			Description: "Casa",
			DateCreate:  now,
			DateUpdate:  now,
		},
		DateCreate: now,
		DateUpdate: now,
	}
)

func TestPgRepository_Add(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Transaction
		expectedTransaction model.Transaction
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputTransaction: model.Transaction{
				Description:   transactionsMock[0].Description,
				Amount:        transactionsMock[0].Amount,
				Date:          transactionsMock[0].Date,
				WalletID:      transactionsMock[0].WalletID,
				TypePaymentID: transactionsMock[0].TypePaymentID,
				CategoryID:    transactionsMock[0].CategoryID,
				DateCreate:    transactionsMock[0].DateCreate,
				DateUpdate:    transactionsMock[0].DateUpdate,
			},
			expectedTransaction: model.Transaction{
				ID:            1,
				Description:   transactionsMock[0].Description,
				Amount:        transactionsMock[0].Amount,
				Date:          transactionsMock[0].Date,
				WalletID:      transactionsMock[0].WalletID,
				TypePaymentID: transactionsMock[0].TypePaymentID,
				CategoryID:    transactionsMock[0].CategoryID,
				DateCreate:    transactionsMock[0].DateCreate,
				DateUpdate:    transactionsMock[0].DateUpdate,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "transactions" ("description","amount","date","parent_transaction_id","wallet_id","type_payment_id","category_id","transaction_status_id","date_create","date_update") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING "id"`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date", "id_wallet", "id_type_payment",
						"id_category", "date_create", "date_update",
					}).
						AddRow(transactionsMock[0].ID,
							transactionsMock[0].Description,
							transactionsMock[0].Amount,
							transactionsMock[0].Date,
							transactionsMock[0].WalletID,
							transactionsMock[0].TypePaymentID,
							transactionsMock[0].CategoryID,
							transactionsMock[0].DateCreate,
							transactionsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name: "error",
			inputTransaction: model.Transaction{
				Description: "Aluguel",
				DateCreate:  now,
				DateUpdate:  now,
			},
			expectedTransaction: model.Transaction{},
			mockedErr:           errors.New("gorm error"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "transactions" ("description","amount","date","parent_transaction_id","wallet_id","type_payment_id","category_id","transaction_status_id","date_create","date_update") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING "id"`)).
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
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestPgRepository_FindAll(t *testing.T) {
	tt := []struct {
		name                 string
		expectedTransactions []model.Transaction
		mockedErr            error
		expectedErr          error
		mockFunc             func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:                 "success",
			expectedTransactions: transactionsMock,
			mockedErr:            nil,
			expectedErr:          nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions"`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date",
						"wallet_id",
						"type_payment_id",
						"category_id",
						"date_create", "date_update",
					}).
						AddRow(transactionsMock[0].ID,
							transactionsMock[0].Description,
							transactionsMock[0].Amount,
							transactionsMock[0].Date,
							transactionsMock[0].WalletID,
							transactionsMock[0].TypePaymentID,
							transactionsMock[0].CategoryID,
							transactionsMock[0].DateCreate,
							transactionsMock[0].DateUpdate).
						AddRow(transactionsMock[1].ID,
							transactionsMock[1].Description,
							transactionsMock[1].Amount,
							transactionsMock[1].Date,
							transactionsMock[1].WalletID,
							transactionsMock[1].TypePaymentID,
							transactionsMock[1].CategoryID,
							transactionsMock[1].DateCreate,
							transactionsMock[1].DateUpdate).
						AddRow(transactionsMock[2].ID,
							transactionsMock[2].Description,
							transactionsMock[2].Amount,
							transactionsMock[2].Date,
							transactionsMock[2].WalletID,
							transactionsMock[2].TypePaymentID,
							transactionsMock[2].CategoryID,
							transactionsMock[2].DateCreate,
							transactionsMock[2].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                 "gorm error",
			expectedTransactions: []model.Transaction{},
			mockedErr:            errors.New("gorm error"),
			expectedErr:          model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions"`)).
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
			require.Equal(t, tc.expectedTransactions, result)
		})
	}
}

func TestPgRepository_FindByMonth(t *testing.T) {
	tt := []struct {
		name                 string
		expectedTransactions []model.Transaction
		mockedErr            error
		expectedErr          error
		mockFunc             func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			expectedTransactions: []model.Transaction{
				transactionsMock[0],
				transactionsMock[1],
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE date BETWEEN $1 AND $2`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date",
						"wallet_id",
						"type_payment_id",
						"category_id",
						"date_create", "date_update",
					}).
						AddRow(transactionsMock[0].ID,
							transactionsMock[0].Description,
							transactionsMock[0].Amount,
							transactionsMock[0].Date,
							transactionsMock[0].WalletID,
							transactionsMock[0].TypePaymentID,
							transactionsMock[0].CategoryID,
							transactionsMock[0].DateCreate,
							transactionsMock[0].DateUpdate).
						AddRow(transactionsMock[1].ID,
							transactionsMock[1].Description,
							transactionsMock[1].Amount,
							transactionsMock[1].Date,
							transactionsMock[1].WalletID,
							transactionsMock[1].TypePaymentID,
							transactionsMock[1].CategoryID,
							transactionsMock[1].DateCreate,
							transactionsMock[1].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                 "gorm error",
			expectedTransactions: []model.Transaction{},
			mockedErr:            errors.New("gorm error"),
			expectedErr:          model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE date BETWEEN $1 AND $2`)).
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

			result, err := repo.FindByMonth(context.Background(), model.Period{
				From: transactionsMock[0].Date,
				To:   transactionsMock[1].Date,
			})
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransactions, result)
		})
	}
}

func TestPgRepository_FindByID(t *testing.T) {
	tt := []struct {
		name                string
		expectedTransaction model.Transaction
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:                "success",
			expectedTransaction: transactionsMock[0],
			mockedErr:           nil,
			expectedErr:         nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date",
						"wallet_id",
						"type_payment_id",
						"category_id",
						"date_create", "date_update",
					}).
						AddRow(transactionsMock[0].ID,
							transactionsMock[0].Description,
							transactionsMock[0].Amount,
							transactionsMock[0].Date,
							transactionsMock[0].WalletID,
							transactionsMock[0].TypePaymentID,
							transactionsMock[0].CategoryID,
							transactionsMock[0].DateCreate,
							transactionsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                "gorm error",
			expectedTransaction: model.Transaction{},
			mockedErr:           errors.New("gorm error"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
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
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestPgRepository_FindByIDEager(t *testing.T) {
	tt := []struct {
		name                string
		expectedTransaction eager.Transaction
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:                "success",
			expectedTransaction: transactionEagerMock,
			mockedErr:           nil,
			expectedErr:         nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT "transactions"."id","transactions"."description","transactions"."amount","transactions"."date","transactions"."wallet_id","transactions"."type_payment_id","transactions"."category_id","transactions"."date_create","transactions"."date_update","Wallet"."id" AS "Wallet__id","Wallet"."description" AS "Wallet__description","Wallet"."balance" AS "Wallet__balance","Wallet"."date_create" AS "Wallet__date_create","Wallet"."date_update" AS "Wallet__date_update","TypePayment"."id" AS "TypePayment__id","TypePayment"."description" AS "TypePayment__description","TypePayment"."date_create" AS "TypePayment__date_create","TypePayment"."date_update" AS "TypePayment__date_update","Category"."id" AS "Category__id","Category"."description" AS "Category__description","Category"."date_create" AS "Category__date_create","Category"."date_update" AS "Category__date_update" FROM "transactions" LEFT JOIN "wallets" "Wallet" ON "transactions"."wallet_id" = "Wallet"."id" LEFT JOIN "type_payments" "TypePayment" ON "transactions"."type_payment_id" = "TypePayment"."id" LEFT JOIN "categories" "Category" ON "transactions"."category_id" = "Category"."id" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date",
						"Wallet__id", "Wallet__description", "Wallet__date_create", "Wallet__date_update",
						"TypePayment__id", "TypePayment__description", "TypePayment__date_create", "TypePayment__date_update",
						"Category__id", "Category__description", "Category__date_create", "Category__date_update",
						"date_create", "date_update",
					}).
						AddRow(transactionEagerMock.ID,
							transactionEagerMock.Description,
							transactionEagerMock.Amount,
							transactionEagerMock.Date,
							transactionEagerMock.Wallet.ID,
							transactionEagerMock.Wallet.Description,
							transactionEagerMock.Wallet.DateCreate,
							transactionEagerMock.Wallet.DateUpdate,
							transactionEagerMock.TypePayment.ID,
							transactionEagerMock.TypePayment.Description,
							transactionEagerMock.TypePayment.DateCreate,
							transactionEagerMock.TypePayment.DateUpdate,
							transactionEagerMock.Category.ID,
							transactionEagerMock.Category.Description,
							transactionEagerMock.Category.DateCreate,
							transactionEagerMock.Category.DateUpdate,
							transactionEagerMock.DateCreate,
							transactionEagerMock.DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                "gorm error",
			expectedTransaction: eager.Transaction{},
			mockedErr:           errors.New("gorm error"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT "transactions"."id","transactions"."description","transactions"."amount","transactions"."date","transactions"."wallet_id","transactions"."type_payment_id","transactions"."category_id","transactions"."date_create","transactions"."date_update","Wallet"."id" AS "Wallet__id","Wallet"."description" AS "Wallet__description","Wallet"."balance" AS "Wallet__balance","Wallet"."date_create" AS "Wallet__date_create","Wallet"."date_update" AS "Wallet__date_update","TypePayment"."id" AS "TypePayment__id","TypePayment"."description" AS "TypePayment__description","TypePayment"."date_create" AS "TypePayment__date_create","TypePayment"."date_update" AS "TypePayment__date_update","Category"."id" AS "Category__id","Category"."description" AS "Category__description","Category"."date_create" AS "Category__date_create","Category"."date_update" AS "Category__date_update" FROM "transactions" LEFT JOIN "wallets" "Wallet" ON "transactions"."wallet_id" = "Wallet"."id" LEFT JOIN "type_payments" "TypePayment" ON "transactions"."type_payment_id" = "TypePayment"."id" LEFT JOIN "categories" "Category" ON "transactions"."category_id" = "Category"."id" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
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

			result, err := repo.FindByIDEager(context.Background(), 1)
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTransaction, result)
		})
	}
}

func TestPgRepository_Update(t *testing.T) {
	tt := []struct {
		name                string
		inputTransaction    model.Transaction
		expectedTransaction model.Transaction
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputTransaction: model.Transaction{
				Description:   transactionsMock[0].Description,
				Amount:        transactionsMock[0].Amount,
				Date:          transactionsMock[0].Date,
				WalletID:      transactionsMock[0].WalletID,
				TypePaymentID: transactionsMock[0].TypePaymentID,
				CategoryID:    transactionsMock[0].CategoryID,
			},
			expectedTransaction: model.Transaction{
				ID:          2,
				Description: transactionsMock[0].Description,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "description", "amount", "date", "id_wallet", "id_type_payment",
						"id_category", "date_create", "date_update",
					}).
						AddRow(transactionsMock[1].ID,
							transactionsMock[0].Description,
							transactionsMock[0].Amount,
							transactionsMock[0].Date,
							transactionsMock[0].WalletID,
							transactionsMock[0].TypePaymentID,
							transactionsMock[0].CategoryID,
							transactionsMock[0].DateCreate,
							transactionsMock[0].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "transactions" SET "description"=$1,"amount"=$2,"date"=$3,"wallet_id"=$4,"type_payment_id"=$5,"category_id"=$6,"date_create"=$7,"date_update"=$8 WHERE "id" = $9`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:                "gorm error SELECT",
			expectedTransaction: model.Transaction{},
			mockedErr:           errors.New("gorm error SELECT"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error SELECT")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
					WillReturnError(errors.New("gorm error SELECT"))
				return db, mock, err
			},
		},
		{
			name: "gorm error UPDATE",
			inputTransaction: model.Transaction{
				Description: transactionsMock[0].Description,
			},
			expectedTransaction: model.Transaction{},
			mockedErr:           errors.New("gorm error UPDATE"),
			expectedErr:         model.BusinessError{Msg: "repository error", HTTPCode: http.StatusInternalServerError, Cause: errors.New("gorm error UPDATE")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(transactionsMock[1].ID,
							transactionsMock[1].Description,
							transactionsMock[1].DateCreate,
							transactionsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "transactions" SET "description"=$1,"date_create"=$2,"date_update"=$3 WHERE "id" = $4`)).
					WillReturnError(errors.New("gorm error UPDATE"))
				return db, mock, err
			},
		},
		{
			name:                "no changes error",
			inputTransaction:    model.Transaction{},
			expectedTransaction: model.Transaction{},
			mockedErr:           errors.New("no changes"),
			expectedErr:         model.BusinessError{Msg: "no changes", HTTPCode: http.StatusInternalServerError, Cause: errors.New("no changes")},
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "transactions" WHERE "transactions"."id" = $1 ORDER BY "transactions"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(transactionsMock[1].ID,
							transactionsMock[1].Description,
							transactionsMock[1].DateCreate,
							transactionsMock[1].DateUpdate))
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

			result, err := repo.Update(context.Background(), 2, tc.inputTransaction)
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
					`DELETE FROM "transactions" WHERE "transactions"."id" = $1`)).
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
					`DELETE FROM "transactions" WHERE "transactions"."id" = $1`)).
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
