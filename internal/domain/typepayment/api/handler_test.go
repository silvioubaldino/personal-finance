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

	"personal-finance/internal/domain/typepayment/api"
	"personal-finance/internal/domain/typepayment/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

var (
	mockedTime       = time.Date(2022, 9, 15, 0o7, 30, 0, 0, time.Local)
	typePaymentsMock = []model.TypePayment{
		{
			ID:          1,
			Description: "Débito",
			UserID:      "userID",
			DateUpdate:  mockedTime,
		},
		{
			ID:          2,
			Description: "Crédito",
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
		{
			ID:          3,
			Description: "Pix",
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
	}
)

func TestHandler_Add(t *testing.T) {
	tt := []struct {
		name              string
		inputTypePayment  any
		mockedTypePayment model.TypePayment
		mockedError       error
		expectedBody      string
	}{
		{
			name:             "success",
			inputTypePayment: model.TypePayment{Description: "Débito"},
			mockedTypePayment: model.TypePayment{
				ID:          1,
				Description: "Débito",
				UserID:      "userID",
				DateCreate:  mockedTime,
				DateUpdate:  mockedTime,
			},
			mockedError:  nil,
			expectedBody: `{"id":1,"description":"Débito","user_id":"userID","date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"}`,
		}, {
			name:              "service error",
			inputTypePayment:  model.TypePayment{Description: "Débito"},
			mockedTypePayment: model.TypePayment{},
			mockedError:       errors.New("service error"),
			expectedBody:      `"service error"`,
		}, {
			name:              "service error",
			inputTypePayment:  "",
			mockedTypePayment: model.TypePayment{},
			mockedError:       nil,
			expectedBody:      `"json: cannot unmarshal string into Go value of type model.TypePayment"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Add", tc.inputTypePayment, "userID").Return(tc.mockedTypePayment, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewTypePaymentHandlers(r, svcMock)
			server := httptest.NewServer(r)

			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputTypePayment))
			request, err := http.NewRequest("POST", server.URL+"/typePayments", &requestBody)
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
		name              string
		mockedTypePayment []model.TypePayment
		mockedErr         error
		expectedBody      string
	}{
		{
			name:              "success",
			mockedTypePayment: typePaymentsMock,
			mockedErr:         nil,
			expectedBody: `[{"id":1,"description":"Débito","user_id":"userID","date_create":"0001-01-01T00:00:00Z","date_update":"2022-09-15T07:30:00-04:00"},` +
				`{"id":2,"description":"Crédito","user_id":"userID","date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"},` +
				`{"id":3,"description":"Pix","user_id":"userID","date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"}]`,
		},
		{
			name:              "not found",
			mockedTypePayment: []model.TypePayment{},
			mockedErr:         errors.New("not found"),
			expectedBody:      `"not found"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindAll", "userID").
				Return(tc.mockedTypePayment, tc.mockedErr)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewTypePaymentHandlers(r, svcMock)

			server := httptest.NewServer(r)

			request, err := http.NewRequest(http.MethodGet, server.URL+"/typePayments", nil)
			request.Header.Set("user_token", "userToken")
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(request)
			require.Nil(t, err)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			require.Equal(t, tc.expectedBody, string(body))

			err = resp.Body.Close()
			if err != nil {
				return
			}
		})
	}
}

func TestHandler_FindByID(t *testing.T) {
	tt := []struct {
		name                string
		mockedTypePayment   model.TypePayment
		mockeddErr          error
		mockedID            any
		expectedTypePayment model.TypePayment
		expectedCode        int
		expectedBody        string
	}{
		{
			name:                "success",
			mockedTypePayment:   model.TypePayment{Description: typePaymentsMock[0].Description, UserID: "userID"},
			mockeddErr:          nil,
			mockedID:            1,
			expectedTypePayment: model.TypePayment{Description: typePaymentsMock[0].Description, UserID: "userID"},
			expectedCode:        200,
			expectedBody:        `{"description":"Débito","user_id":"userID","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:                "not found",
			mockedTypePayment:   model.TypePayment{},
			mockeddErr:          errors.New("service error"),
			mockedID:            1,
			expectedTypePayment: model.TypePayment{},
			expectedCode:        404,
			expectedBody:        `"service error"`,
		},
		{
			name:                "parse error",
			mockedTypePayment:   model.TypePayment{},
			mockeddErr:          nil,
			mockedID:            "a",
			expectedTypePayment: model.TypePayment{},
			expectedCode:        500,
			expectedBody:        `"strconv.ParseInt: parsing \"\\\"a\\\"\": invalid syntax"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByID", tc.mockedID, "userID").
				Return(tc.mockedTypePayment, tc.mockeddErr)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewTypePaymentHandlers(r, svcMock)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)

			server := httptest.NewServer(r)

			request, err := http.NewRequest(http.MethodGet, server.URL+"/typePayments/"+string(mockerIDString), nil)
			require.Nil(t, err)
			request.Header.Set("user_token", "userToken")

			resp, err := http.DefaultClient.Do(request)
			require.Nil(t, err)
			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			require.Equal(t, tc.expectedBody, string(body))

			err = resp.Body.Close()
			if err != nil {
				return
			}
		})
	}
}

func TestHandler_Update(t *testing.T) {
	tt := []struct {
		name              string
		inputTypePayment  any
		mockedTypePayment model.TypePayment
		mockedID          any
		mockedError       error
		expectedBody      string
	}{
		{
			name:              "success",
			inputTypePayment:  model.TypePayment{Description: typePaymentsMock[0].Description, UserID: typePaymentsMock[0].UserID},
			mockedTypePayment: model.TypePayment{Description: typePaymentsMock[0].Description, UserID: typePaymentsMock[0].UserID},
			mockedID:          1,
			mockedError:       nil,
			expectedBody:      `{"description":"Débito","user_id":"userID","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"}`,
		}, {
			name:              "service error",
			inputTypePayment:  model.TypePayment{Description: typePaymentsMock[0].Description},
			mockedTypePayment: model.TypePayment{},
			mockedID:          1,
			mockedError:       errors.New("service error"),
			expectedBody:      `"service error"`,
		}, {
			name:              "parse error",
			inputTypePayment:  model.TypePayment{Description: typePaymentsMock[0].Description},
			mockedTypePayment: model.TypePayment{},
			mockedID:          "a",
			mockedError:       nil,
			expectedBody:      `"strconv.ParseInt: parsing \"\\\"a\\\"\": invalid syntax"`,
		}, {
			name:              "bind error",
			inputTypePayment:  "",
			mockedTypePayment: model.TypePayment{},
			mockedID:          1,
			mockedError:       nil,
			expectedBody:      `"json: cannot unmarshal string into Go value of type model.TypePayment"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Update", tc.inputTypePayment, "userID").Return(tc.mockedTypePayment, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewTypePaymentHandlers(r, svcMock)
			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputTypePayment))
			request, _ := http.NewRequest("PUT", server.URL+"/typePayments/"+string(mockerIDString), &requestBody)
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
			name:         "parse error",
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
			api.NewTypePaymentHandlers(r, &svcMock)

			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			request, _ := http.NewRequest("DELETE", server.URL+"/typePayments/"+string(mockerIDString), nil)
			resp, _ := http.DefaultClient.Do(request)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err = resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}
