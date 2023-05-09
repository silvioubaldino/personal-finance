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
	"github.com/stretchr/testify/require"

	"personal-finance/internal/domain/movement/api"
	"personal-finance/internal/domain/movement/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"
)

var (
	mockedUUID     = uuid.New()
	mockedDateFrom = time.Date(2023, 0o1, 0o1, 0, 0, 0, 0, time.UTC)
	mockedDateTo   = time.Date(2023, 0o1, 30, 0, 0, 0, 0, time.UTC)
)

func TestHandler_Add(t *testing.T) {
	tt := []struct {
		name           string
		input          any
		mockedMovement model.Movement
		mockedError    error
		expectedBody   string
	}{
		{
			name:           "success",
			input:          mockMovement(nil, "success", 2),
			mockedMovement: mockMovement(&mockedUUID, "success", 2),
			mockedError:    nil,
			expectedBody:   `{"id":"` + mockedUUID.String() + `","description":"success","amount":0,"parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"0001-01-01T00:00:00Z"}`,
		},
		{
			name:           "error bindJSON",
			input:          "",
			mockedMovement: model.Movement{},
			mockedError:    nil,
			expectedBody:   `"json: cannot unmarshal string into Go value of type model.Movement"`,
		},
		{
			name:           "error empty statusID",
			input:          mockMovement(nil, "empty statusID", 0),
			mockedMovement: model.Movement{},
			mockedError:    nil,
			expectedBody:   `"status_id must be valid"`,
		},
		{
			name:           "service error",
			input:          mockMovement(nil, "empty statusID", 2),
			mockedMovement: model.Movement{},
			mockedError:    errors.New("service error"),
			expectedBody:   `"service error"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("Add", tc.input, "userID").Return(tc.mockedMovement, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewMovementHandlers(r, svcMock)
			server := httptest.NewServer(r)

			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.input))
			request, err := http.NewRequest("POST", server.URL+"/movements", &requestBody)
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

func TestHandler_FindByPeriod(t *testing.T) {
	tt := []struct {
		name            string
		input           any
		inputPeriod     model.Period
		inputPeriodPath string
		mockedMovement  []model.Movement
		mockedError     error
		expectedBody    string
	}{
		{
			name:        "success",
			inputPeriod: mockPeriod(mockedDateFrom, mockedDateTo),
			inputPeriodPath: "?from=" + mockedDateFrom.Format("2006-01-02") +
				"&to=" + mockedDateTo.Format("2006-01-02"),
			mockedMovement: []model.Movement{mockMovement(&mockedUUID, "success", 2)},
			mockedError:    nil,
			expectedBody:   `[{"id":"` + mockedUUID.String() + `","description":"success","amount":0,"parent_transaction_id":null,"wallets":{"balance":0},"type_payments":{},"categories":{},"date_update":"0001-01-01T00:00:00Z"}]`,
		},
		{
			name:        "from parse error",
			inputPeriod: mockPeriod(mockedDateFrom, mockedDateTo),
			inputPeriodPath: "?from=" + mockedDateFrom.Format("2006-01") +
				"&to=" + mockedDateTo.Format("2006-01-02"),
			mockedMovement: nil,
			mockedError:    nil,
			expectedBody:   `"parsing time \"2023-01\" as \"2006-01-02\": cannot parse \"\" as \"-\""`,
		},
		{
			name:        "to parse error",
			inputPeriod: mockPeriod(mockedDateFrom, mockedDateTo),
			inputPeriodPath: "?from=" + mockedDateFrom.Format("2006-01-02") +
				"&to=" + mockedDateTo.Format("2006-01"),
			mockedMovement: nil,
			mockedError:    nil,
			expectedBody:   `"parsing time \"2023-01\" as \"2006-01-02\": cannot parse \"\" as \"-\""`,
		},
		{
			name:        "to before from error",
			inputPeriod: mockPeriod(mockedDateFrom, mockedDateTo),
			inputPeriodPath: "?from=" + mockedDateTo.Format("2006-01-02") +
				"&to=" + mockedDateFrom.Format("2006-01-02"),
			mockedMovement: nil,
			mockedError:    nil,
			expectedBody:   `"period invalid: 'from' must be before 'to'"`,
		},
		{
			name:        "service error",
			inputPeriod: mockPeriod(mockedDateFrom, mockedDateTo),
			inputPeriodPath: "?from=" + mockedDateFrom.Format("2006-01-02") +
				"&to=" + mockedDateTo.Format("2006-01-02"),
			mockedMovement: nil,
			mockedError:    errors.New("service error"),
			expectedBody:   `"service error"`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			svcMock := &service.Mock{}
			svcMock.On("FindByPeriod", tc.inputPeriod, "userID").Return(tc.mockedMovement, tc.mockedError)

			r := gin.Default()
			authenticator := authentication.Mock{}
			r.Use(authenticator.Authenticate())

			api.NewMovementHandlers(r, svcMock)
			server := httptest.NewServer(r)

			requestBody := bytes.Buffer{}
			require.Nil(t, json.NewEncoder(&requestBody).Encode(tc.input))
			request, err := http.NewRequest("GET",
				server.URL+"/movements/period"+tc.inputPeriodPath, nil)
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

func mockMovement(id *uuid.UUID, description string, statusID int) model.Movement {
	return model.Movement{
		ID:          id,
		Description: description,
		StatusID:    statusID,
	}
}

func mockPeriod(from, to time.Time) model.Period {
	return model.Period{
		From: from,
		To:   to,
	}
}
