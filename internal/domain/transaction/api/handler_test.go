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

	"personal-finance/internal/domain/transaction/api"
	"personal-finance/internal/domain/transaction/service"
	"personal-finance/internal/model"
)

var (
	mockedTime       = time.Date(2022, 9, 15, 0o7, 30, 0, 0, time.Local)
	transactionsMock = []model.Transaction{
		{
			ID:            1,
			Description:   "Aluguel",
			Amount:        1000.0,
			Date:          time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    mockedTime,
			DateUpdate:    mockedTime,
		},
		{
			ID:            2,
			Description:   "Energia",
			Amount:        300.0,
			Date:          time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    mockedTime,
			DateUpdate:    mockedTime,
		},
		{
			ID:            3,
			Description:   "Agua",
			Amount:        120.0,
			Date:          time.Date(2022, time.September, 30, 0, 0, 0, 0, time.Local),
			WalletID:      1,
			TypePaymentID: 1,
			CategoryID:    2,
			DateCreate:    mockedTime,
			DateUpdate:    mockedTime,
		},
	}
	balancesMock = []model.Balance{
		{
			Period: model.Period{
				From: time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local),
				To:   time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local),
			},
			Expense: -1300.0,
		},
		{
			Period: model.Period{
				From: time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local),
				To:   time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local),
			},
			Expense: 120.0,
		},
	}
)

func TestHandler_Add(t *testing.T) {
	tt := []struct {
		name              string
		inputTransaction  any
		mockedTransaction model.Transaction
		mockedError       error
		expectedBody      string
	}{
		{
			name: "success",
			inputTransaction: model.Transaction{
				Description:   "Aluguel",
				Amount:        1000,
				WalletID:      1,
				TypePaymentID: 1,
				CategoryID:    3,
			},
			mockedTransaction: transactionsMock[0],
			mockedError:       nil,
			expectedBody:      `{"id":1,"description":"Aluguel","amount":1000,"date":"2022-09-01T00:00:00-04:00","parent_transaction_id":0,"wallet_id":1,"type_payment_id":1,"category_id":2,"transaction_status_id":0,"date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"}`,
		}, {
			name:              "service error",
			inputTransaction:  model.Transaction{Description: "Nubank"},
			mockedTransaction: model.Transaction{},
			mockedError:       errors.New("service error"),
			expectedBody:      `"service error"`,
		}, {
			name:              "bind error",
			inputTransaction:  "",
			mockedTransaction: model.Transaction{},
			mockedError:       nil,
			expectedBody:      `"json: cannot unmarshal string into Go value of type model.Transaction"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Add", tc.inputTransaction).Return(tc.mockedTransaction, tc.mockedError)

			r := gin.Default()

			api.NewTransactionHandlers(r, svcMock)
			server := httptest.NewServer(r)

			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputTransaction))
			request, err := http.NewRequest("POST", server.URL+"/transactions", &requestBody)
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
		mockedTransaction []model.Transaction
		mockedErr         error
		expectedBody      string
	}{
		{
			name:              "success",
			mockedTransaction: transactionsMock,
			mockedErr:         nil,
			expectedBody:      `[{"id":1,"description":"Aluguel","amount":1000,"date":"2022-09-01T00:00:00-04:00","parent_transaction_id":0,"wallet_id":1,"type_payment_id":1,"category_id":2,"transaction_status_id":0,"date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"},{"id":2,"description":"Energia","amount":300,"date":"2022-09-15T00:00:00-04:00","parent_transaction_id":0,"wallet_id":1,"type_payment_id":1,"category_id":2,"transaction_status_id":0,"date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"},{"id":3,"description":"Agua","amount":120,"date":"2022-09-30T00:00:00-04:00","parent_transaction_id":0,"wallet_id":1,"type_payment_id":1,"category_id":2,"transaction_status_id":0,"date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"}]`,
		}, {
			name:              "not found",
			mockedTransaction: []model.Transaction{},
			mockedErr: model.BusinessError{
				Msg:      "resource not found",
				HTTPCode: 500,
				Cause:    errors.New("not found"),
			},
			expectedBody: `"resource not found"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindAll").
				Return(tc.mockedTransaction, tc.mockedErr)

			r := gin.Default()
			api.NewTransactionHandlers(r, svcMock)

			server := httptest.NewServer(r)

			resp, err := http.Get(server.URL + "/transactions")
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

func TestHandler_FindByMonth(t *testing.T) {
	tt := []struct {
		name              string
		queryString       string
		mockedTransaction []model.Transaction
		mockedErr         error
		expectedBody      string
	}{
		{
			name:        "success",
			queryString: "/transactions/period?from=2022-08-01&to=2022-08-15",
			mockedTransaction: []model.Transaction{
				transactionsMock[0],
				transactionsMock[1],
			},
			mockedErr:    nil,
			expectedBody: `[{"id":1,"description":"Aluguel","amount":1000,"date":"2022-09-01T00:00:00-04:00","parent_transaction_id":0,"wallet_id":1,"type_payment_id":1,"category_id":2,"transaction_status_id":0,"date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"},{"id":2,"description":"Energia","amount":300,"date":"2022-09-15T00:00:00-04:00","parent_transaction_id":0,"wallet_id":1,"type_payment_id":1,"category_id":2,"transaction_status_id":0,"date_create":"2022-09-15T07:30:00-04:00","date_update":"2022-09-15T07:30:00-04:00"}]`,
		},
		{
			name:              "parse from error",
			queryString:       "/transactions/period?from=a",
			mockedTransaction: []model.Transaction{},
			mockedErr:         errors.New("parsing time \"\" as \"2006-01-02\": cannot parse \"\" as \"2006\""),
			expectedBody:      `"parsing time \"a\" as \"2006-01-02\": cannot parse \"a\" as \"2006\""`,
		},
		{
			name:              "parse to error",
			queryString:       "/transactions/period?from=2022-08-01&to=a",
			mockedTransaction: []model.Transaction{},
			mockedErr:         errors.New("parsing time \"\" as \"2006-01-02\": cannot parse \"\" as \"2006\""),
			expectedBody:      `"parsing time \"a\" as \"2006-01-02\": cannot parse \"a\" as \"2006\""`,
		},
		{
			name:              "no date error",
			queryString:       "/transactions/period",
			mockedTransaction: []model.Transaction{},
			mockedErr:         nil,
			expectedBody:      `"period invalid: date must be informed"`,
		},
		{
			name:              "'from' after 'to' error",
			queryString:       "/transactions/period?from=2022-08-15&to=2022-08-01",
			mockedTransaction: []model.Transaction{},
			mockedErr:         nil,
			expectedBody:      `"period invalid: 'from' must be before 'to'"`,
		},
		{
			name:              "not found error",
			queryString:       "/transactions/period?from=2022-01-01&to=2022-01-30",
			mockedTransaction: []model.Transaction{},
			mockedErr:         errors.New("not found"),
			expectedBody:      `"not found"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByMonth").
				Return(tc.mockedTransaction, tc.mockedErr)

			r := gin.Default()
			api.NewTransactionHandlers(r, svcMock)

			server := httptest.NewServer(r)

			resp, err := http.Get(server.URL + tc.queryString)
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

func TestHandler_BalanceByPeriod(t *testing.T) {
	tt := []struct {
		name              string
		queryString       string
		mockedTransaction []model.Transaction
		mockedBalance     model.Balance
		mockedErr         error
		expectedBody      string
	}{
		{
			name:        "success",
			queryString: "/balance/period?from=2022-08-01&to=2022-08-15",
			mockedTransaction: []model.Transaction{
				transactionsMock[0],
				transactionsMock[1],
			},
			mockedBalance: balancesMock[0],
			mockedErr:     nil,
			expectedBody:  `{"period":{"from":"2022-09-01T00:00:00-04:00","to":"2022-09-15T00:00:00-04:00"},"expense":-1300,"income":0}`,
		},
		{
			name:              "parse from error",
			queryString:       "/balance/period?from=a",
			mockedTransaction: []model.Transaction{},
			mockedErr:         errors.New("parsing time \"\" as \"2006-01-02\": cannot parse \"\" as \"2006\""),
			expectedBody:      `"parsing time \"a\" as \"2006-01-02\": cannot parse \"a\" as \"2006\""`,
		},
		{
			name:              "parse to error",
			queryString:       "/balance/period?from=2022-08-01&to=a",
			mockedTransaction: []model.Transaction{},
			mockedErr:         errors.New("parsing time \"\" as \"2006-01-02\": cannot parse \"\" as \"2006\""),
			expectedBody:      `"parsing time \"a\" as \"2006-01-02\": cannot parse \"a\" as \"2006\""`,
		},
		{
			name:              "no date error",
			queryString:       "/balance/period",
			mockedTransaction: []model.Transaction{},
			mockedErr:         nil,
			expectedBody:      `"period invalid: date must be informed"`,
		},
		{
			name:              "not found error",
			queryString:       "/balance/period?from=2022-01-01&to=2022-01-30",
			mockedTransaction: []model.Transaction{},
			mockedErr:         errors.New("not found"),
			expectedBody:      `"not found"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByMonth").
				Return(tc.mockedTransaction, tc.mockedErr)
			svcMock.On("BalanceByPeriod").
				Return(tc.mockedBalance, tc.mockedErr)

			r := gin.Default()
			api.NewTransactionHandlers(r, svcMock)

			server := httptest.NewServer(r)

			resp, err := http.Get(server.URL + tc.queryString)
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
		mockedTransaction   model.Transaction
		mockedErr           error
		mockedID            any
		expectedTransaction model.Transaction
		expectedCode        int
		expectedBody        string
	}{
		{
			name:                "success",
			mockedTransaction:   model.Transaction{Description: transactionsMock[0].Description},
			mockedErr:           nil,
			mockedID:            1,
			expectedTransaction: model.Transaction{Description: transactionsMock[0].Description},
			expectedCode:        200,
			expectedBody:        `{"description":"Aluguel","amount":0,"date":"0001-01-01T00:00:00Z","parent_transaction_id":0,"wallet_id":0,"type_payment_id":0,"category_id":0,"transaction_status_id":0,"date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:              "not found",
			mockedTransaction: model.Transaction{},
			mockedErr: model.BusinessError{
				Msg:      "resource not found",
				HTTPCode: 500,
				Cause:    errors.New("not found"),
			},
			mockedID:            1,
			expectedTransaction: model.Transaction{},
			expectedCode:        404,
			expectedBody:        `"resource not found"`,
		},
		{
			name:                "parse error",
			mockedTransaction:   model.Transaction{},
			mockedErr:           nil,
			mockedID:            "a",
			expectedTransaction: model.Transaction{},
			expectedCode:        500,
			expectedBody:        `"id must be valid: \"a\""`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByID", mock.Anything).
				Return(tc.mockedTransaction, tc.mockedErr)

			r := gin.Default()
			api.NewTransactionHandlers(r, svcMock)

			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			resp, err := http.Get(server.URL + "/transactions/" + string(mockerIDString))
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
		inputTransaction  any
		mockedTransaction model.Transaction
		mockedID          any
		mockedError       error
		expectedBody      string
	}{
		{
			name: "success",
			inputTransaction: model.Transaction{
				Description:   transactionsMock[0].Description,
				WalletID:      1,
				TypePaymentID: 1,
				CategoryID:    2,
			},
			mockedTransaction: model.Transaction{
				Description:   transactionsMock[0].Description,
				WalletID:      1,
				TypePaymentID: 1,
				CategoryID:    2,
			},
			mockedID:     1,
			mockedError:  nil,
			expectedBody: `{"description":"Aluguel","amount":0,"date":"0001-01-01T00:00:00Z","parent_transaction_id":0,"wallet_id":1,"type_payment_id":1,"category_id":2,"transaction_status_id":0,"date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"}`,
		}, {
			name:              "service error",
			inputTransaction:  model.Transaction{Description: transactionsMock[0].Description},
			mockedTransaction: model.Transaction{},
			mockedID:          1,
			mockedError:       errors.New("service error"),
			expectedBody:      `"service error"`,
		}, {
			name:              "parse error",
			inputTransaction:  model.Transaction{Description: transactionsMock[0].Description},
			mockedTransaction: model.Transaction{},
			mockedID:          "a",
			mockedError:       nil,
			expectedBody:      `"strconv.ParseInt: parsing \"\\\"a\\\"\": invalid syntax"`,
		}, {
			name:              "bind error",
			inputTransaction:  "",
			mockedTransaction: model.Transaction{},
			mockedID:          1,
			mockedError:       nil,
			expectedBody:      `"json: cannot unmarshal string into Go value of type model.Transaction"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Update", tc.inputTransaction).Return(tc.mockedTransaction, tc.mockedError)

			r := gin.Default()

			api.NewTransactionHandlers(r, svcMock)
			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.inputTransaction))
			request, _ := http.NewRequest("PUT", server.URL+"/transactions/"+string(mockerIDString), &requestBody)

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
			api.NewTransactionHandlers(r, &svcMock)

			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.mockedID)
			require.Nil(t, err)
			request, _ := http.NewRequest("DELETE", server.URL+"/transactions/"+string(mockerIDString), nil)
			resp, _ := http.DefaultClient.Do(request)

			body, readingBodyErr := io.ReadAll(resp.Body)
			require.Nil(t, readingBodyErr)

			err = resp.Body.Close()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBody, string(body))
		})
	}
}
