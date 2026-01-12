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

	"personal-finance/internal/domain/category/repository"
	"personal-finance/internal/model"
)

var (
	now   = time.Now()
	uuid1 = uuid.New()
	uuid2 = uuid.New()
	uuid3 = uuid.New()

	categoriesMock = []model.Category{
		{
			ID:          &uuid1,
			Description: "Alimentacao",
			UserID:      "userID",
			Color:       "#FFFFFF",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &uuid2,
			Description: "Casa",
			UserID:      "userID",
			Color:       "#000000",
			DateCreate:  now,
			DateUpdate:  now,
		},
		{
			ID:          &uuid3,
			Description: "Carro",
			UserID:      "userID",
			Color:       "#FF0000",
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
				Color:       categoriesMock[0].Color,
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
					`INSERT INTO "categories" ("description","user_id","color","date_create","date_update")
				VALUES ($1,$2,$3,$4,$5) RETURNING "id"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "color", "date_create", "date_update"}).
						AddRow(categoriesMock[0].ID,
							categoriesMock[0].Description,
							categoriesMock[0].UserID,
							categoriesMock[0].Color,
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
					`INSERT INTO "categories" ("description","user_id","color","date_create","date_update")
					VALUES ($1,$2,$3,$4,$5) RETURNING "id"`)).
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

			result, err := repo.Add(context.Background(), tc.inputCategory)
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
					`SELECT * FROM "categories" WHERE user_id IN($1,$2)`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "color", "date_create", "date_update"}).
						AddRow(categoriesMock[0].ID, categoriesMock[0].Description, categoriesMock[0].UserID, categoriesMock[0].Color, categoriesMock[0].DateCreate, categoriesMock[0].DateUpdate).
						AddRow(categoriesMock[1].ID, categoriesMock[1].Description, categoriesMock[1].UserID, categoriesMock[1].Color, categoriesMock[1].DateCreate, categoriesMock[1].DateUpdate).
						AddRow(categoriesMock[2].ID, categoriesMock[2].Description, categoriesMock[2].UserID, categoriesMock[2].Color, categoriesMock[2].DateCreate, categoriesMock[2].DateUpdate))
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

			result, err := repo.FindAll(context.Background())
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
							WHERE (user_id = $1 OR user_id = $2) AND "categories"."id" = $3 
							ORDER BY "categories"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "color", "date_create", "date_update"}).
						AddRow(categoriesMock[0].ID, categoriesMock[0].Description, categoriesMock[0].Color, categoriesMock[0].DateCreate, categoriesMock[0].DateUpdate))
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
							WHERE (user_id = $1 OR user_id = $2) AND "categories"."id" = $3
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

			result, err := repo.FindByID(context.Background(), uuid1)
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
				Color:       categoriesMock[0].Color,
			},
			expectedCategory: model.Category{
				ID:          &uuid2,
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
				Color:       categoriesMock[0].Color,
			},
			mockedErr:   nil,
			expectedErr: nil,
			mockFunc: func() (*sql.DB, sqlmock.Sqlmock, error) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectQuery(regexp.QuoteMeta(
					`SELECT * FROM "categories" 
							WHERE (user_id = $1 OR user_id = $2) AND "categories"."id" = $3 
							ORDER BY "categories"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "color", "date_create", "date_update"}).
						AddRow(categoriesMock[1].ID,
							categoriesMock[1].Description,
							categoriesMock[1].UserID,
							categoriesMock[1].Color,
							categoriesMock[1].DateCreate,
							categoriesMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "categories" 
							SET "description"=$1,"user_id"=$2,"color"=$3,"date_create"=$4,"date_update"=$5 
							WHERE "id" = $6`)).
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
							WHERE (user_id = $1 OR user_id = $2) AND "categories"."id" = $3 
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
							WHERE (user_id = $1 OR user_id = $2) AND "categories"."id" = $3 
							ORDER BY "categories"."id" LIMIT 1`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "description", "user_id", "color", "date_create", "date_update"}).
						AddRow(categoriesMock[1].ID,
							categoriesMock[1].Description,
							categoriesMock[1].UserID,
							categoriesMock[1].Color,
							categoriesMock[1].DateCreate,
							categoriesMock[1].DateUpdate))
				mock.ExpectExec(regexp.QuoteMeta(
					`UPDATE "categories" 
							SET "description"=$1,"user_id"=$2,"color"=$3,"date_create"=$4,"date_update"=$5 
							WHERE "id" = $6`)).
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

			result, err := repo.Update(context.Background(), uuid2, tc.inputCategory)
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

			err = repo.Delete(context.Background(), uuid1)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
