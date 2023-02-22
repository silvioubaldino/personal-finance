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

	"personal-finance/internal/domain/typepayment/repository"
	"personal-finance/internal/model"
)

var (
	now              = time.Now()
	typePaymentsMock = []model.TypePayment{
		{
			ID:          1,
			Description: "Débito",
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          2,
			Description: "Crédito",
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          3,
			Description: "Pix",
			UserID:      "userID",
			DateCreate:  now,
			DateUpdate:  now,
		},
	}
)

func TestPgRepository_Add(t *testing.T) {
	tt := []struct {
		name                string
		inputTypePayment    model.TypePayment
		expectedTypePayment model.TypePayment
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputTypePayment: model.TypePayment{
				Description: typePaymentsMock[0].Description,
				DateCreate:  typePaymentsMock[0].DateCreate,
				DateUpdate:  typePaymentsMock[0].DateUpdate,
			},
			expectedTypePayment: typePaymentsMock[0],
			mockedErr:           nil,
			expectedErr:         nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "type_payments" ("description","user_id","date_create","date_update") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(typePaymentsMock[0].ID, typePaymentsMock[0].Description, typePaymentsMock[0].UserID, typePaymentsMock[0].DateCreate, typePaymentsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name: "error",
			inputTypePayment: model.TypePayment{
				Description: "Débito",
				DateCreate:  now,
				DateUpdate:  now,
			},
			expectedTypePayment: model.TypePayment{},
			mockedErr:           errors.New("gorm error"),
			expectedErr:         errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "type_payments" ("description","user_id","date_create","date_update") VALUES ($1,$2,$3,$4) RETURNING "id"`)).
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

			result, err := repo.Add(context.Background(), tc.inputTypePayment, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTypePayment, result)
		})
	}
}

func TestPgRepository_FindAll(t *testing.T) {
	tt := []struct {
		name                 string
		expectedTypePayments []model.TypePayment
		mockedErr            error
		expectedErr          error
		mockFunc             func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:                 "success",
			expectedTypePayments: typePaymentsMock,
			mockedErr:            nil,
			expectedErr:          nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "type_payments"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(typePaymentsMock[0].ID, typePaymentsMock[0].Description, typePaymentsMock[0].UserID, typePaymentsMock[0].DateCreate, typePaymentsMock[0].DateUpdate).
						AddRow(typePaymentsMock[1].ID, typePaymentsMock[1].Description, typePaymentsMock[1].UserID, typePaymentsMock[1].DateCreate, typePaymentsMock[1].DateUpdate).
						AddRow(typePaymentsMock[2].ID, typePaymentsMock[2].Description, typePaymentsMock[2].UserID, typePaymentsMock[2].DateCreate, typePaymentsMock[2].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                 "gorm error",
			expectedTypePayments: []model.TypePayment{},
			mockedErr:            errors.New("gorm error"),
			expectedErr:          errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "type_payments"`)).
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

			result, err := repo.FindAll(context.Background(), "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTypePayments, result)
		})
	}
}

func TestPgRepository_FindByID(t *testing.T) {
	tt := []struct {
		name                string
		expectedTypePayment model.TypePayment
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:                "success",
			expectedTypePayment: typePaymentsMock[0],
			mockedErr:           nil,
			expectedErr:         nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "type_payments" 
							WHERE user_id=$1 AND "type_payments"."id" = $2 
							ORDER BY "type_payments"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(typePaymentsMock[0].ID, typePaymentsMock[0].Description, typePaymentsMock[0].UserID, typePaymentsMock[0].DateCreate, typePaymentsMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:                "gorm error",
			expectedTypePayment: model.TypePayment{},
			mockedErr:           errors.New("gorm error"),
			expectedErr:         errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "type_payments" 
							WHERE user_id=$1 AND "type_payments"."id" = $2 
							ORDER BY "type_payments"."id" LIMIT 1`)).
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

			result, err := repo.FindByID(context.Background(), 1, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTypePayment, result)
		})
	}
}

func TestPgRepository_Update(t *testing.T) {
	tt := []struct {
		name                string
		inputTypePayment    model.TypePayment
		expectedTypePayment model.TypePayment
		mockedErr           error
		expectedErr         error
		mockFunc            func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputTypePayment: model.TypePayment{
				Description: typePaymentsMock[0].Description,
			},
			expectedTypePayment: model.TypePayment{
				ID:          2,
				Description: typePaymentsMock[0].Description,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "type_payments" 
							WHERE user_id=$1 AND "type_payments"."id" = $2 
							ORDER BY "type_payments"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(typePaymentsMock[1].ID,
							typePaymentsMock[1].Description,
							typePaymentsMock[1].DateCreate,
							typePaymentsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "type_payments" SET "description"=$1,"user_id"=$2,"date_create"=$3,"date_update"=$4 
							WHERE "id" = $5`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:                "gorm error SELECT",
			expectedTypePayment: model.TypePayment{},
			mockedErr:           errors.New("gorm error SELECT"),
			expectedErr:         errors.New("gorm error SELECT"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "type_payments" 
							WHERE user_id=$1 AND "type_payments"."id" = $2 
							ORDER BY "type_payments"."id" LIMIT 1`)).
					WillReturnError(errors.New("gorm error SELECT"))
				return db, mock, err
			},
		},
		{
			name: "gorm error UPDATE",
			inputTypePayment: model.TypePayment{
				Description: typePaymentsMock[0].Description,
			},
			expectedTypePayment: model.TypePayment{},
			mockedErr:           errors.New("gorm error UPDATE"),
			expectedErr:         errors.New("gorm error UPDATE"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "type_payments" 
							WHERE user_id=$1 AND "type_payments"."id" = $2 
							ORDER BY "type_payments"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(typePaymentsMock[1].ID,
							typePaymentsMock[1].Description,
							typePaymentsMock[1].DateCreate,
							typePaymentsMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "type_payments" SET "description"=$1,"user_id"=$2,"date_create"=$3,"date_update"=$4 
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

			result, err := repo.Update(context.Background(), 2, tc.inputTypePayment, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedTypePayment.ID, result.ID)
			require.Equal(t, tc.expectedTypePayment.Description, result.Description)
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
					`DELETE FROM "type_payments" WHERE "type_payments"."id" = $1`)).
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
					`DELETE FROM "type_payments" WHERE "type_payments"."id" = $1`)).
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
