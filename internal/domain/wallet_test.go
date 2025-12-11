package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWallet_HasSufficientBalance(t *testing.T) {
	id := uuid.New()
	tests := map[string]struct {
		wallet  Wallet
		amount  float64
		expects bool
	}{
		"sufficient balance for debit": {
			wallet:  Wallet{ID: &id, Balance: 100.0},
			amount:  -50.0,
			expects: true,
		},
		"insufficient balance for debit": {
			wallet:  Wallet{ID: &id, Balance: 30.0},
			amount:  -50.0,
			expects: false,
		},
		"exact balance for debit": {
			wallet:  Wallet{ID: &id, Balance: 50.0},
			amount:  -50.0,
			expects: true,
		},
		"credit does not require balance": {
			wallet:  Wallet{ID: &id, Balance: 0.0},
			amount:  100.0,
			expects: true,
		},
		"zero debit": {
			wallet:  Wallet{ID: &id, Balance: 10.0},
			amount:  0.0,
			expects: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := tc.wallet.HasSufficientBalance(tc.amount)
			assert.Equal(t, tc.expects, result)
		})
	}
}
