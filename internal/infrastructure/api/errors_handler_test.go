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
		"should map ErrCreditLimitInsufficient to 422": {
			err:          domain.WrapCreditLimitInsufficient(domain.New("amount exceeds limit"), "validate credit limit"),
			expectedCode: http.StatusUnprocessableEntity,
		},
		"should keep usecase ErrInsufficientCreditLimit (statement import) mapped to 403": {
			err:          usecase.ErrInsufficientCreditLimit,
			expectedCode: http.StatusForbidden,
		},
		"should map invoice already paid on pay (domain conflict) to 409": {
			err:          domain.WrapConflict(usecase.ErrInvoiceAlreadyPaid, "pay invoice"),
			expectedCode: http.StatusConflict,
		},
		"should map invoice not paid on revert (domain conflict) to 409": {
			err:          domain.WrapConflict(usecase.ErrInvoiceNotPaid, "revert invoice payment"),
			expectedCode: http.StatusConflict,
		},
		"should map invalid payment amount to 400": {
			err:          domain.WrapInvalidInput(usecase.ErrInvalidPaymentAmount, "pay invoice"),
			expectedCode: http.StatusBadRequest,
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
