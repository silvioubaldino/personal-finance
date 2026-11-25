package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/internal/usecase"

	"github.com/stretchr/testify/assert"
)

func TestToAPIError(t *testing.T) {
	tests := map[string]struct {
		// input
		err error
		// expected
		expectedCode int
	}{
		"should map ErrInvoiceAlreadyPaid to 422": {
			err:          usecase.ErrInvoiceAlreadyPaid,
			expectedCode: http.StatusUnprocessableEntity,
		},
		"should map wrapped ErrInvoiceAlreadyPaid to 422": {
			err:          fmt.Errorf("confirm invoice: %w", usecase.ErrInvoiceAlreadyPaid),
			expectedCode: http.StatusUnprocessableEntity,
		},
		"should map ErrCreditCardNotFound to 404": {
			err:          repository.ErrCreditCardNotFound,
			expectedCode: http.StatusNotFound,
		},
		"should map wrapped ErrCreditCardNotFound (ConfirmInvoice propagation chain) to 404": {
			err: fmt.Errorf("find credit card: %w",
				fmt.Errorf("error finding credit card: %w", repository.ErrCreditCardNotFound)),
			expectedCode: http.StatusNotFound,
		},
		"should map ErrWalletInsufficient to 422 (existing behavior, regression guard)": {
			err:          domain.ErrWalletInsufficient,
			expectedCode: http.StatusUnprocessableEntity,
		},
		"should map unknown error to 500": {
			err:          errors.New("boom"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Act
			result := toAPIError(tc.err)

			// Assert
			assert.Equal(t, tc.expectedCode, result.Error.Code)
		})
	}
}
