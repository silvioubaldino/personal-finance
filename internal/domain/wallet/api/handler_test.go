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
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"personal-finance/internal/domain/wallet/api"
	"personal-finance/internal/domain/wallet/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/internal/usecase"
)

var (
	mockedTime = time.Date(2022, 9, 15, 0o7, 30, 0, 0, time.Local)

	wid1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	wid2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	wid3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")

	walletsMock = []model.Wallet{
		{
			ID:          &wid1,
			Description: "Nubank",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
		{
			ID:          &wid2,
			Description: "Banco do brasil",
			Balance:     0,
			UserID:      "userID",
			DateCreate:  mockedTime,
			DateUpdate:  mockedTime,
		},
		{
			ID:          &wid3,
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
		expectedCode int
		expectedBody string
	}{
		{
			name:         "success",
			inputWallet:  model.Wallet{Description: "Nubank"},
			mockedWallet: walletsMock[0],
			mockedError:  nil,
			expectedCode: http.StatusCreated,
			expectedBody: `{"id":"00000000-0000-0000-0000-000000000001","description":"Nubank","balance":0,"initial_balance":0,"initial_date":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:         "service error",
			inputWallet:  model.Wallet{Description: "Nubank"},
			mockedWallet: model.Wallet{},
			mockedError:  errors.New("service error"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: `"service error"`,
		},
		{
			name:         "wallet limit reached",
			inputWallet:  model.Wallet{Description: "Nubank"},
			mockedWallet: model.Wallet{},
			mockedError:  usecase.ErrWalletLimitReached,
			expectedCode: http.StatusForbidden,
			expectedBody: `{"error":{"code":403,"message":"wallet limit reached for your plan"}}`,
		},
		{
			name:         "bind error",
			inputWallet:  "",
			mockedWallet: model.Wallet{},
			mockedError:  nil,
			expectedCode: http.StatusBadRequest,
			expectedBody: `"json: cannot unmarshal string into Go value of type model.Wallet"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			if w, ok := tc.inputWallet.(model.Wallet); ok {
				svcMock.On("Add", w).Return(tc.mockedWallet, tc.mockedError)
			}

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

			require.Equal(t, tc.expectedCode, resp.StatusCode)
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
			expectedBody: `[{"id":"00000000-0000-0000-0000-000000000001","description":"Nubank","balance":0,"initial_balance":0,"initial_date":"0001-01-01T00:00:00Z"},{"id":"00000000-0000-0000-0000-000000000002","description":"Banco do brasil","balance":0,"initial_balance":0,"initial_date":"0001-01-01T00:00:00Z"},{"id":"00000000-0000-0000-0000-000000000003","description":"Santander","balance":0,"initial_balance":0,"initial_date":"0001-01-01T00:00:00Z"}]`,
		},
		{
			name:         "not found",
			mockedWallet: []model.Wallet{},
			mockedErr:    errors.New("not found"),
			expectedBody: `"not found"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindAll").
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
		name         string
		mockedWallet model.Wallet
		mockedErr    error
		mockedID     *uuid.UUID
		expectedCode int
		expectedBody string
	}{
		{
			name:         "success",
			mockedWallet: model.Wallet{Description: walletsMock[0].Description, UserID: walletsMock[0].UserID},
			mockedErr:    nil,
			mockedID:     &wid1,
			expectedCode: 200,
			expectedBody: `{"description":"Nubank","balance":0,"initial_balance":0,"initial_date":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:         "not found",
			mockedWallet: model.Wallet{},
			mockedErr:    errors.New("service error"),
			mockedID:     &wid1,
			expectedCode: 404,
			expectedBody: `"service error"`,
		},
		{
			name:         "parse error",
			mockedWallet: model.Wallet{},
			mockedErr:    nil,
			mockedID:     nil,
			expectedCode: 500,
			expectedBody: `"invalid UUID length: 3"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			if tc.mockedID != nil {
				svcMock.On("FindByID", tc.mockedID).
					Return(tc.mockedWallet, tc.mockedErr)
			}

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewWalletHandlers(r, svcMock)

			server := httptest.NewServer(r)

			var idStr string
			if tc.mockedID != nil {
				idStr = tc.mockedID.String()
			} else {
				idStr = "bad"
			}
			request, err := http.NewRequest(http.MethodGet, server.URL+"/wallets/"+idStr, nil)
			request.Header.Set("user_token", "userToken")
			require.Nil(t, err)

			resp, err := http.DefaultClient.Do(request)
			require.Nil(t, err)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			require.Equal(t, tc.expectedCode, resp.StatusCode)
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
		mockedID     *uuid.UUID
		mockedError  error
		expectedBody string
	}{
		{
			name:         "success",
			inputWallet:  model.Wallet{Description: walletsMock[0].Description, UserID: walletsMock[0].UserID},
			mockedWallet: model.Wallet{Description: walletsMock[0].Description, UserID: walletsMock[0].UserID},
			mockedID:     &wid1,
			mockedError:  nil,
			expectedBody: `{"description":"Nubank","balance":0,"initial_balance":0,"initial_date":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:         "service error",
			inputWallet:  model.Wallet{Description: walletsMock[0].Description},
			mockedWallet: model.Wallet{},
			mockedID:     &wid1,
			mockedError:  errors.New("service error"),
			expectedBody: `"service error"`,
		},
		{
			name:         "parse error",
			inputWallet:  model.Wallet{Description: walletsMock[0].Description},
			mockedWallet: model.Wallet{},
			mockedID:     nil,
			mockedError:  nil,
			expectedBody: `"invalid UUID length: 3"`,
		},
		{
			name:         "bind error",
			inputWallet:  "",
			mockedWallet: model.Wallet{},
			mockedID:     &wid1,
			mockedError:  nil,
			expectedBody: `"json: cannot unmarshal string into Go value of type model.Wallet"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			if tc.mockedID != nil {
				if w, ok := tc.inputWallet.(model.Wallet); ok {
					svcMock.On("Update", tc.mockedID, w).Return(tc.mockedWallet, tc.mockedError)
				}
			}

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewWalletHandlers(r, svcMock)
			server := httptest.NewServer(r)

			var idStr string
			if tc.mockedID != nil {
				idStr = tc.mockedID.String()
			} else {
				idStr = "bad"
			}
			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputWallet))
			request, _ := http.NewRequest("PUT", server.URL+"/wallets/"+idStr, &requestBody)
			request.Header.Set("user_token", "userToken")

			resp, _ := http.DefaultClient.Do(request)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err := resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tt := []struct {
		name         string
		mockedErr    error
		mockedID     *uuid.UUID
		expectedBody string
	}{
		{
			name:         "success",
			mockedErr:    nil,
			mockedID:     &wid1,
			expectedBody: ``,
		},
		{
			name:         "service error",
			mockedErr:    errors.New("service error"),
			mockedID:     &wid1,
			expectedBody: `"service error"`,
		},
		{
			name:         "parse error",
			mockedErr:    nil,
			mockedID:     nil,
			expectedBody: `{}`,
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

			var idStr string
			if tc.mockedID != nil {
				idStr = tc.mockedID.String()
			} else {
				idStr = "bad"
			}
			request, _ := http.NewRequest("DELETE", server.URL+"/wallets/"+idStr, nil)
			resp, _ := http.DefaultClient.Do(request)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err := resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}
