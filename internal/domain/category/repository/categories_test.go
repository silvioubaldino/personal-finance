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

	"personal-finance/internal/domain/category/repository"
	"personal-finance/internal/model"
)

var (
	now            = time.Now()
	categoriesMock = []model.Category{
		{
			ID:          1,
			Description: "Alimentacao",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          2,
			Description: "Casa",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          3,
			Description: "Carro",
			DateCreate:  now,
			DateUpdate:  now,
		},
	}
)

func TestPgRepository_Add(t *testing.T) {
	tt := []struct {
		name             string
		inputCategory    model.Category
		expectedCategory model.Category
		mockedErr        error
		expectedErr      error
		mockFunc         func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputCategory: model.Category{
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
				DateCreate:  categoriesMock[0].DateCreate,
				DateUpdate:  categoriesMock[0].DateUpdate,
			},
			expectedCategory: categoriesMock[0],
			mockedErr:        nil,
			expectedErr:      nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "categories" ("description","user_id","date_create","date_update")
				VALUES ($1,$2,$3,$4) RETURNING "id"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(categoriesMock[0].ID,
							categoriesMock[0].Description,
							categoriesMock[0].UserID,
							categoriesMock[0].DateCreate,
							categoriesMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name: "error",
			inputCategory: model.Category{
				Description: "Alimentacao",
				DateCreate:  now,
				DateUpdate:  now,
			},
			expectedCategory: model.Category{},
			mockedErr:        errors.New("gorm error"),
			expectedErr:      errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`INSERT INTO "categories" ("description","user_id","date_create","date_update")
					VALUES ($1,$2,$3,$4) RETURNING "id"`)).
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

			result, err := repo.Add(context.Background(), tc.inputCategory, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategory, result)
		})
	}
}

func TestPgRepository_FindAll(t *testing.T) {
	tt := []struct {
		name               string
		expectedCategories []model.Category
		mockedErr          error
		expectedErr        error
		mockFunc           func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:               "success",
			expectedCategories: categoriesMock,
			mockedErr:          nil,
			expectedErr:        nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories" WHERE user_id=$1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(categoriesMock[0].ID, categoriesMock[0].Description, categoriesMock[0].UserID, categoriesMock[0].DateCreate, categoriesMock[0].DateUpdate).
						AddRow(categoriesMock[1].ID, categoriesMock[1].Description, categoriesMock[1].UserID, categoriesMock[1].DateCreate, categoriesMock[1].DateUpdate).
						AddRow(categoriesMock[2].ID, categoriesMock[2].Description, categoriesMock[2].UserID, categoriesMock[2].DateCreate, categoriesMock[2].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:               "gorm error",
			expectedCategories: []model.Category{},
			mockedErr:          errors.New("gorm error"),
			expectedErr:        errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories`)).
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
			require.Equal(t, tc.expectedCategories, result)
		})
	}
}

func TestPgRepository_FindByID(t *testing.T) {
	tt := []struct {
		name             string
		expectedCategory model.Category
		mockedErr        error
		expectedErr      error
		mockFunc         func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name:             "success",
			expectedCategory: categoriesMock[0],
			mockedErr:        nil,
			expectedErr:      nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories" 
							WHERE user_id=$1 AND "categories"."id" = $2 
							ORDER BY "categories"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "date_create", "date_update"}).
						AddRow(categoriesMock[0].ID, categoriesMock[0].Description, categoriesMock[0].DateCreate, categoriesMock[0].DateUpdate))
				return db, mock, err
			},
		},
		{
			name:             "gorm error",
			expectedCategory: model.Category{},
			mockedErr:        errors.New("gorm error"),
			expectedErr:      errors.New("gorm error"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories" 
							WHERE user_id=$1 AND "categories"."id" = $2
							ORDER BY "categories"."id" LIMIT 1`)).
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
			require.Equal(t, tc.expectedCategory, result)
		})
	}
}

func TestPgRepository_Update(t *testing.T) {
	tt := []struct {
		name             string
		inputCategory    model.Category
		expectedCategory model.Category
		mockedErr        error
		expectedErr      error
		mockFunc         func() (*sql.DB, sqlmock.Sqlmock, error)
	}{
		{
			name: "success",
			inputCategory: model.Category{
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
			},
			expectedCategory: model.Category{
				ID:          2,
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories" 
							WHERE user_id=$1 AND "categories"."id" = $2 
							ORDER BY "categories"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(categoriesMock[1].ID,
							categoriesMock[1].Description,
							categoriesMock[1].UserID,
							categoriesMock[1].DateCreate,
							categoriesMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "categories" 
							SET "description"=$1,"user_id"=$2,"date_create"=$3,"date_update"=$4 
							WHERE "id" = $5`)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				return db, mock, err
			},
		},
		{
			name:             "gorm error SELECT",
			expectedCategory: model.Category{},
			mockedErr:        errors.New("gorm error SELECT"),
			expectedErr:      errors.New("gorm error SELECT"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories" 
							WHERE user_id=$1 AND "categories"."id" = $2 
							ORDER BY "categories"."id" LIMIT 1`)).
					WillReturnError(errors.New("gorm error SELECT"))
				return db, mock, err
			},
		},
		{
			name: "gorm error UPDATE",
			inputCategory: model.Category{
				Description: categoriesMock[0].Description,
			},
			expectedCategory: model.Category{},
			mockedErr:        errors.New("gorm error UPDATE"),
			expectedErr:      errors.New("gorm error UPDATE"),
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories" 
							WHERE user_id=$1 AND "categories"."id" = $2 
							ORDER BY "categories"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "date_create", "date_update"}).
						AddRow(categoriesMock[1].ID,
							categoriesMock[1].Description,
							categoriesMock[1].UserID,
							categoriesMock[1].DateCreate,
							categoriesMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "categories" 
							SET "description"=$1,"user_id"=$2,"date_create"=$3,"date_update"=$4 
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

			result, err := repo.Update(context.Background(), 2, tc.inputCategory, "userID")
			require.Equal(t, tc.expectedErr, err)
			require.Equal(t, tc.expectedCategory.ID, result.ID)
			require.Equal(t, tc.expectedCategory.Description, result.Description)
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
					`DELETE FROM "categories" WHERE "categories"."id" = $1`)).
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
					`DELETE FROM "categories" WHERE "categories"."id" = $1`)).
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
