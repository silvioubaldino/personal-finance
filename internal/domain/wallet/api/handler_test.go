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

	"personal-finance/internal/domain/wallet/api"
	"personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

var (
	mockedTime  = time.Date(2022, 9, 15, 0o7, 30, 0, 0, time.Local)
	walletsMock = []model.Wallet{
		{
			ID:          1,
			Description: "Nubank",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
		{
			ID:          2,
			Description: "Banco do brasil",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
		{
			ID:          3,
			Description: "Santander",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
	}
)

func TestHandler_Add(t *testing.T) {
	tt := []struct {
		name         string
		inputWallet  any
		mockedWallet model.Wallet
		mockedError  error
		expectedBody string
	}{
		{
			name:        "success",
			inputWallet: model.Wallet{Description: "Nubank"},
			mockedWallet: model.Wallet{
				ID:          1,
				Description: "Nubank",
				Balance:     0,
				UserID:      "userID",
				DateCreate:  mockedTime,
				DateUpdate:  mockedTime,
			},
			mockedError:  nil,
			expectedBody: `{"id":1,"description":"Nubank","balance":0}`,
		}, {
			name:         "service error",
			inputWallet:  model.Wallet{Description: "Nubank"},
			mockedWallet: model.Wallet{},
			mockedError:  errors.New("service error"),
			expectedBody: `"service error"`,
		}, {
			name:         "service error",
			inputWallet:  "",
			mockedWallet: model.Wallet{},
			mockedError:  nil,
			expectedBody: `"json: cannot unmarshal string into Go value of type model.Wallet"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Add", tc.inputWallet, "userID").Return(tc.mockedWallet, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewWalletHandlers(r, svcMock)

			server := httptest.NewServer(r)

			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputWallet))
			request, err := http.NewRequest("POST", server.URL+"/wallets", &requestBody)
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
		name         string
		mockedWallet []model.Wallet
		mockedErr    error
		expectedBody string
	}{
		{
			name:         "success",
			mockedWallet: walletsMock,
			mockedErr:    nil,
			expectedBody: `[{"id":1,"description":"Nubank","balance":0},{"id":2,"description":"Banco do brasil","balance":0},{"id":3,"description":"Santander","balance":0}]`,
		}, {
			name:         "not found",
			mockedWallet: []model.Wallet{},
			mockedErr:    errors.New("not found"),
			expectedBody: `"not found"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindAll", "userID").
				Return(tc.mockedWallet, tc.mockedErr)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewWalletHandlers(r, svcMock)

			server := httptest.NewServer(r)

			request, err := http.NewRequest(http.MethodGet, server.URL+"/wallets", nil)
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
		name           string
		mockedWallet   model.Wallet
		mockeddErr     error
		mockedID       any
		expectedWallet model.Wallet
		expectedCode   int
		expectedBody   string
	}{
		{
			name:           "success",
			mockedWallet:   model.Wallet{Description: walletsMock[0].Description, UserID: walletsMock[0].UserID},
			mockeddErr:     nil,
			mockedID:       1,
			expectedWallet: model.Wallet{Description: walletsMock[0].Description, UserID: walletsMock[0].UserID},
			expectedCode:   200,
			expectedBody:   `{"description":"Nubank","balance":0}`,
		},
		{
			name:           "not found",
			mockedWallet:   model.Wallet{},
			mockeddErr:     errors.New("service error"),
			mockedID:       1,
			expectedWallet: model.Wallet{},
			expectedCode:   404,
			expectedBody:   `"service error"`,
		},
		{
			name:           "parse error",
			mockedWallet:   model.Wallet{},
			mockeddErr:     nil,
			mockedID:       "a",
			expectedWallet: model.Wallet{},
			expectedCode:   500,
			expectedBody:   `"strconv.ParseInt: parsing \"\\\"a\\\"\": invalid syntax"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByID", tc.mockedID, "userID").
				Return(tc.mockedWallet, tc.mockeddErr)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewWalletHandlers(r, svcMock)

			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			request, err := http.NewRequest(http.MethodGet, server.URL+"/wallets/"+string(mockerIDString), nil)
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

func TestHandler_Update(t *testing.T) {
	tt := []struct {
		name         string
		inputWallet  any
		mockedWallet model.Wallet
		mockedID     any
		mockedError  error
		expectedBody string
	}{
		{
			name:         "success",
			inputWallet:  model.Wallet{Description: walletsMock[0].Description, UserID: walletsMock[0].UserID},
			mockedWallet: model.Wallet{Description: walletsMock[0].Description, UserID: walletsMock[0].UserID},
			mockedID:     1,
			mockedError:  nil,
			expectedBody: `{"description":"Nubank","balance":0}`,
		}, {
			name:         "service error",
			inputWallet:  model.Wallet{Description: walletsMock[0].Description},
			mockedWallet: model.Wallet{},
			mockedID:     1,
			mockedError:  errors.New("service error"),
			expectedBody: `"service error"`,
		}, {
			name:         "parse error",
			inputWallet:  model.Wallet{Description: walletsMock[0].Description},
			mockedWallet: model.Wallet{},
			mockedID:     "a",
			mockedError:  nil,
			expectedBody: `"strconv.ParseInt: parsing \"\\\"a\\\"\": invalid syntax"`,
		}, {
			name:         "bind error",
			inputWallet:  "",
			mockedWallet: model.Wallet{},
			mockedID:     1,
			mockedError:  nil,
			expectedBody: `"json: cannot unmarshal string into Go value of type model.Wallet"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Update", tc.mockedID, tc.inputWallet, "userID").Return(tc.mockedWallet, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewWalletHandlers(r, svcMock)
			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputWallet))
			request, _ := http.NewRequest("PUT", server.URL+"/wallets/"+string(mockerIDString), &requestBody)
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
			api.NewWalletHandlers(r, &svcMock)

			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			request, _ := http.NewRequest("DELETE", server.URL+"/wallets/"+string(mockerIDString), nil)
			resp, _ := http.DefaultClient.Do(request)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err = resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}
