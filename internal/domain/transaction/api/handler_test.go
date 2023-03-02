package api_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"personal-finance/internal/domain/transaction/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"personal-finance/internal/domain/transaction/api"
	"personal-finance/internal/model"
)

var (
	mockedUUID = uuid.New()

	mockedTime        = time.Date(2022, 9, 15, 0, 0, 0, 0, time.UTC)
	aluguelmockedTime = time.Date(2022, time.September, 0o1, 0, 0, 0, 0, time.Local)
	energiaMockedTime = time.Date(2022, time.September, 15, 0, 0, 0, 0, time.Local)
	aguaMockedTime    = time.Date(2022, time.September, 30, 0, 0, 0, 0, time.Local)

	transactionsMock = []model.Transaction{
		{
			Estimate: &model.Movement{
				Description:   "Aluguel",
				Amount:        1000.0,
				Date:          &aluguelmockedTime,
				WalletID:      1,
				TypePaymentID: 1,
				CategoryID:    2,
				DateCreate:    mockedTime,
				DateUpdate:    mockedTime,
			},
			Consolidation: &model.Consolidation{
				Estimated: 1000.0,
				Realized:  1000.0,
			},
			DoneList: model.MovementList{
				{
					Description:   "Aluguel",
					Amount:        1000.0,
					Date:          &aluguelmockedTime,
					WalletID:      1,
					TypePaymentID: 1,
					CategoryID:    2,
					DateCreate:    mockedTime,
					DateUpdate:    mockedTime,
				},
			},
		},
		{
			Estimate: &model.Movement{
				Description:   "Energia",
				Amount:        300.0,
				Date:          &energiaMockedTime,
				WalletID:      1,
				TypePaymentID: 1,
				CategoryID:    2,
				DateCreate:    mockedTime,
				DateUpdate:    mockedTime,
			},
			Consolidation: &model.Consolidation{
				Estimated: 300.0,
				Realized:  300.0,
			},
			DoneList: model.MovementList{
				{
					Description:   "Energia",
					Amount:        300.0,
					Date:          &energiaMockedTime,
					WalletID:      1,
					TypePaymentID: 1,
					CategoryID:    2,
					DateCreate:    mockedTime,
					DateUpdate:    mockedTime,
				},
			},
		},
		{
			Estimate:      &model.Movement{},
			Consolidation: &model.Consolidation{},
			DoneList: model.MovementList{
				{
					Description:   "Agua",
					Amount:        120.0,
					Date:          &aguaMockedTime,
					WalletID:      1,
					TypePaymentID: 1,
					CategoryID:    2,
					DateCreate:    mockedTime,
					DateUpdate:    mockedTime,
				},
			},
		},
	}
)

func TestHandler_FindByID(t *testing.T) {
	tt := []struct {
		name              string
		mockedTransaction model.Transaction
		mockedErr         error
		inputID           any
		expectedCode      int
		expectedBody      string
	}{
		{
			name:              "success",
			mockedTransaction: transactionsMock[0],
			mockedErr:         nil,
			inputID:           mockedUUID,
			expectedCode:      200,
			expectedBody:      `{"transaction_id":null,"estimate":{"description":"Aluguel","amount":1000,"date":"2022-09-01T00:00:00-04:00","user_id":"","wallet_id":1,"wallets":{"balance":0,"user_id":"","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"},"type_payment_id":1,"type_payments":{"user_id":"","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"},"category_id":2,"categories":{"user_id":"","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"},"date_create":"2022-09-15T00:00:00Z","date_update":"2022-09-15T00:00:00Z"},"consolidation":{"estimated":1000,"realized":1000,"remaining":0},"done_list":[{"description":"Aluguel","amount":1000,"date":"2022-09-01T00:00:00-04:00","user_id":"","wallet_id":1,"wallets":{"balance":0,"user_id":"","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"},"type_payment_id":1,"type_payments":{"user_id":"","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"},"category_id":2,"categories":{"user_id":"","date_create":"0001-01-01T00:00:00Z","date_update":"0001-01-01T00:00:00Z"},"date_create":"2022-09-15T00:00:00Z","date_update":"2022-09-15T00:00:00Z"}]}`,
		},
		{
			name:              "not found",
			mockedTransaction: transactionsMock[1],
			mockedErr: model.BusinessError{
				Msg:      "resource not found",
				HTTPCode: 500,
				Cause:    errors.New("not found"),
			},
			inputID:      mockedUUID,
			expectedCode: 404,
			expectedBody: `"resource not found"`,
		},
		{
			name:              "parse error",
			mockedTransaction: transactionsMock[2],
			mockedErr:         nil,
			inputID:           "a",
			expectedCode:      500,
			expectedBody:      `"id must be valid: \"a\""`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByID", tc.inputID, "userID").Return(tc.mockedTransaction, tc.mockedErr)

			r := gin.Default()
			api.NewTransactionHandlers(r, nil, svcMock)

			server := httptest.NewServer(r)

			mockerIDString, err := json.Marshal(tc.inputID)
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

func TestHandler_FindByPeriod(t *testing.T) {
	type mocks struct {
		mockedTransaction []model.Transaction
		mockedErr         error
		mockSvc           func() *service.Mock
	}
	tt := []struct {
		name            string
		inputPeriod     model.Period
		inputPeriodPath string
		mocks           mocks
		expectedBody    string
		expectedCode    int
		expectedErr     error
	}{
		{
			name: "success",
			inputPeriod: model.Period{
				From: mockedTime,
				To:   mockedTime.AddDate(0, 3, 0),
			},
			inputPeriodPath: "?from=" + mockedTime.Format("2006-01-02") +
				"&to=" + mockedTime.AddDate(0, 3, 0).Format("2006-01-02"),
			mocks: mocks{
				mockedTransaction: transactionsMock,
				mockedErr:         nil,
				mockSvc: func() *service.Mock {
					svcMock := service.Mock{}
					svcMock.On("FindByPeriod",
						model.Period{
							From: mockedTime,
							To:   mockedTime.AddDate(0, 3, 0),
						}, "userID").
						Return(transactionsMock, nil)
					return &svcMock
				},
			},
			expectedBody: `[{"transaction_id":null,"estimate":{"description":"Aluguel","amount":1000,"date":"2022-09-01T00:00:00-04:00","parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"2022-09-15T00:00:00Z"},"consolidation":{"estimated":1000,"realized":1000,"remaining":0},"done_list":[{"description":"Aluguel","amount":1000,"date":"2022-09-01T00:00:00-04:00","parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"2022-09-15T00:00:00Z"}]},{"transaction_id":null,"estimate":{"description":"Energia","amount":300,"date":"2022-09-15T00:00:00-04:00","parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"2022-09-15T00:00:00Z"},"consolidation":{"estimated":300,"realized":300,"remaining":0},"done_list":[{"description":"Energia","amount":300,"date":"2022-09-15T00:00:00-04:00","parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"2022-09-15T00:00:00Z"}]},{"transaction_id":null,"estimate":{"amount":0,"parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"0001-01-01T00:00:00Z"},"consolidation":{"remaining":0},"done_list":[{"description":"Agua","amount":120,"date":"2022-09-30T00:00:00-04:00","parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"2022-09-15T00:00:00Z"}]}]`,
			expectedCode: 200,
			expectedErr:  nil,
		},
		{
			name:        "parse from error",
			inputPeriod: model.Period{},
			inputPeriodPath: "?from=" + mockedTime.Format("2006-01") +
				"&to=" + mockedTime.AddDate(0, 3, 0).Format("2006-01-02"),
			mocks: mocks{
				mockedTransaction: nil,
				mockedErr:         nil,
				mockSvc: func() *service.Mock {
					svcMock := service.Mock{}
					return &svcMock
				},
			},
			expectedBody: `"parsing time \"2022-09\" as \"2006-01-02\": cannot parse \"\" as \"-\""`,
			expectedCode: 500,
			expectedErr:  nil,
		},
		{
			name:        "parse to error",
			inputPeriod: model.Period{},
			inputPeriodPath: "?from=" + mockedTime.Format("2006-01-02") +
				"&to=" + mockedTime.AddDate(0, 3, 0).Format("2006-01"),
			mocks: mocks{
				mockedTransaction: nil,
				mockedErr:         nil,
				mockSvc: func() *service.Mock {
					svcMock := service.Mock{}
					return &svcMock
				},
			},
			expectedBody: `"parsing time \"2022-12\" as \"2006-01-02\": cannot parse \"\" as \"-\""`,
			expectedCode: 500,
			expectedErr:  nil,
		},
		{
			name:        "period invalid error",
			inputPeriod: model.Period{},
			inputPeriodPath: "?from=" + mockedTime.AddDate(0, 3, 0).Format("2006-01-02") +
				"&to=" + mockedTime.Format("2006-01-02"),
			mocks: mocks{
				mockedTransaction: nil,
				mockedErr:         nil,
				mockSvc: func() *service.Mock {
					svcMock := service.Mock{}
					return &svcMock
				},
			},
			expectedBody: `"period invalid: 'from' must be before 'to'"`,
			expectedCode: 500,
			expectedErr:  nil,
		},
		{
			name: "service error",
			inputPeriod: model.Period{
				From: mockedTime,
				To:   mockedTime.AddDate(0, 3, 0),
			},
			inputPeriodPath: "?from=" + mockedTime.Format("2006-01-02") +
				"&to=" + mockedTime.AddDate(0, 3, 0).Format("2006-01-02"),
			mocks: mocks{
				mockedTransaction: []model.Transaction{},
				mockedErr:         errors.New("service error"),
				mockSvc: func() *service.Mock {
					svcMock := service.Mock{}
					svcMock.On("FindByPeriod",
						model.Period{
							From: mockedTime,
							To:   mockedTime.AddDate(0, 3, 0),
						}, "userID").
						Return([]model.Transaction{}, errors.New("service error"))
					return &svcMock
				},
			},
			expectedBody: `"service error"`,
			expectedCode: 404,
			expectedErr:  nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := tc.mocks.mockSvc()
			r := gin.Default()
			api.NewTransactionHandlers(r, nil, svcMock)

			server := httptest.NewServer(r)

			resp, err := http.Get(server.URL + "/transactions/period" + tc.inputPeriodPath)
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
