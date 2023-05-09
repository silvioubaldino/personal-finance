package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"personal-finance/internal/domain/category/api"
	"personal-finance/internal/domain/category/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

var (
	mockedTime     = time.Date(2022, 9, 15, 0o7, 30, 0, 0, time.Local)
	categoriesMock = []model.Category{
		{
			ID:          1,
			Description: "Alimentacao",
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
		{
			ID:          2,
			Description: "Casa",
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
		{
			ID:          3,
			Description: "Carro",
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
	}
)

func TestPing(t *testing.T) {
	r := gin.Default()

	svcMock := service.Mock{}
	api.NewCategoryHandlers(r, &svcMock)

	server := httptest.NewServer(r)

	resp, err := http.Get(server.URL + "/ping")
	require.Nil(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandler_Add(t *testing.T) {
	tt := []struct {
		name           string
		inputCategory  any
		mockedCategory model.Category
		mockedError    error
		expectedBody   string
	}{
		{
			name:          "success",
			inputCategory: model.Category{Description: "Alimentação", UserID: "userID"},
			mockedCategory: model.Category{
				ID:          1,
				Description: "Alimentação",
				UserID:      "userID",
				DateCreate:  mockedTime,
				DateUpdate:  mockedTime,
			},
			mockedError:  nil,
			expectedBody: `{"id":1,"description":"Alimentação"}`,
		}, {
			name:           "service error",
			inputCategory:  model.Category{Description: "Alimentação"},
			mockedCategory: model.Category{},
			mockedError:    errors.New("service error"),
			expectedBody:   `"service error"`,
		}, {
			name:           "service error",
			inputCategory:  "",
			mockedCategory: model.Category{},
			mockedError:    nil,
			expectedBody:   `"json: cannot unmarshal string into Go value of type model.Category"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Add", tc.inputCategory, "userID").Return(tc.mockedCategory, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewCategoryHandlers(r, svcMock)
			server := httptest.NewServer(r)

			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputCategory))
			request, err := http.NewRequest("POST", server.URL+"/categories", &requestBody)
			request.Header.Set("user_token", "userToken")
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(request)
			require.Nil(t, err)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err = resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestHandler_FindAll(t *testing.T) {
	tt := []struct {
		name           string
		inputToken     string
		mockedCategory []model.Category
		mockedErr      error
		expectedBody   string
	}{
		{
			name:           "success",
			inputToken:     "userToken",
			mockedCategory: categoriesMock,
			mockedErr:      nil,
			expectedBody:   `[{"id":1,"description":"Alimentacao"},{"id":2,"description":"Casa"},{"id":3,"description":"Carro"}]`,
		}, {
			name:           "not found",
			inputToken:     "userToken",
			mockedCategory: []model.Category{},
			mockedErr:      errors.New("not found"),
			expectedBody:   `"not found"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindAll", "userID").
				Return(tc.mockedCategory, tc.mockedErr)
			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewCategoryHandlers(r, svcMock)

			req, err := http.NewRequest(http.MethodGet, "/categories", nil)
			require.Nil(t, err)
			req.Header.Set("user_token", tc.inputToken)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)
			body, readingBodyErr := io.ReadAll(rr.Body)
			require.Nil(t, readingBodyErr)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestHandler_FindByID(t *testing.T) {
	tt := []struct {
		name           string
		inputID        any
		inputToken     string
		mockedCategory model.Category
		mockeddErr     error
		expectedCat    model.Category
		expectedCode   int
		expectedBody   string
	}{
		{
			name:       "success",
			inputID:    1,
			inputToken: "userToken",
			mockedCategory: model.Category{
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
			},
			mockeddErr: nil,
			expectedCat: model.Category{
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
			},
			expectedCode: 200,
			expectedBody: `{"description":"Alimentacao"}`,
		},
		{
			name:           "not found",
			inputID:        1,
			inputToken:     "userToken",
			mockedCategory: model.Category{},
			mockeddErr:     errors.New("service error"),
			expectedCat:    model.Category{},
			expectedCode:   404,
			expectedBody:   `"service error"`,
		},
		{
			name:           "parse error",
			inputID:        "a",
			inputToken:     "userToken",
			mockedCategory: model.Category{},
			mockeddErr:     nil,
			expectedCat:    model.Category{},
			expectedCode:   500,
			expectedBody:   `"strconv.ParseInt: parsing \"\\\"a\\\"\": invalid syntax"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByID", tc.inputID, "userID").
				Return(tc.mockedCategory, tc.mockeddErr)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())
			api.NewCategoryHandlers(r, svcMock)

			mockerIDString, err := json.Marshal(tc.inputID)
			require.Nil(t, err)
			req, err := http.NewRequest(http.MethodGet, "/categories/"+string(mockerIDString), nil)
			require.Nil(t, err)
			req.Header.Set("user_token", tc.inputToken)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)
			body, readingBodyErr := io.ReadAll(rr.Body)
			require.Nil(t, readingBodyErr)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestHandler_Update(t *testing.T) {
	tt := []struct {
		name           string
		inputCategory  any
		mockedCategory model.Category
		mockedID       any
		mockedError    error
		expectedBody   string
	}{
		{
			name: "success",
			inputCategory: model.Category{
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
			},
			mockedCategory: model.Category{
				Description: categoriesMock[0].Description,
				UserID:      categoriesMock[0].UserID,
			},
			mockedID:     1,
			mockedError:  nil,
			expectedBody: `{"description":"Alimentacao"}`,
		}, {
			name:           "service error",
			inputCategory:  model.Category{Description: categoriesMock[0].Description},
			mockedCategory: model.Category{},
			mockedID:       1,
			mockedError:    errors.New("service error"),
			expectedBody:   `"service error"`,
		}, {
			name:           "parse error",
			inputCategory:  model.Category{Description: categoriesMock[0].Description},
			mockedCategory: model.Category{},
			mockedID:       "a",
			mockedError:    nil,
			expectedBody:   `"strconv.ParseInt: parsing \"\\\"a\\\"\": invalid syntax"`,
		}, {
			name:           "bind error",
			inputCategory:  "",
			mockedCategory: model.Category{},
			mockedID:       1,
			mockedError:    nil,
			expectedBody:   `"json: cannot unmarshal string into Go value of type model.Category"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Update", tc.inputCategory, "userID").Return(tc.mockedCategory, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewCategoryHandlers(r, svcMock)
			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputCategory))
			request, _ := http.NewRequest("PUT", server.URL+"/categories/"+string(mockerIDString), &requestBody)
			request.Header.Set("user_token", "userToken")
			resp, _ := http.DefaultClient.Do(request)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err = resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tt := []struct {
		name         string
		mockedErr    error
		mockedID     any
		expectedBody string
	}{
		{
			name:         "success",
			mockedErr:    nil,
			mockedID:     1,
			expectedBody: ``,
		}, {
			name:         "service error",
			mockedErr:    errors.New("service error"),
			mockedID:     1,
			expectedBody: `"service error"`,
		}, {
			name:         "service error",
			mockedErr:    nil,
			mockedID:     "a",
			expectedBody: `{"Func":"ParseInt","Num":"\"a\"","Err":{}}`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := service.Mock{}
			svcMock.On("Delete", mock.Anything).
				Return(tc.mockedErr)

			r := gin.Default()
			api.NewCategoryHandlers(r, &svcMock)

			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			request, _ := http.NewRequest("DELETE", server.URL+"/categories/"+string(mockerIDString), nil)
			resp, _ := http.DefaultClient.Do(request)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err = resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}
