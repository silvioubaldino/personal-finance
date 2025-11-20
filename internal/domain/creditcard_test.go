package domain

import (
	"testing"
)

func TestCreditCard_HasSufficientLimit(t *testing.T) {
	tests := []struct {
		name        string
		creditLimit float64
		amount      float64
		want        bool
	}{
		{
			name:        "limite suficiente para despesa",
			creditLimit: 5000.0,
			amount:      -1000.0,
			want:        true,
		},
		{
			name:        "limite insuficiente para despesa",
			creditLimit: 500.0,
			amount:      -1000.0,
			want:        false,
		},
		{
			name:        "limite exato para despesa",
			creditLimit: 1000.0,
			amount:      -1000.0,
			want:        true,
		},
		{
			name:        "limite zero com despesa",
			creditLimit: 0.0,
			amount:      -100.0,
			want:        false,
		},
		{
			name:        "limite zero sem movimentação",
			creditLimit: 0.0,
			amount:      0.0,
			want:        true,
		},
		{
			name:        "aumentando limite (reversão de despesa)",
			creditLimit: 1000.0,
			amount:      500.0,
			want:        true,
		},
		{
			name:        "limite negativo com nova despesa",
			creditLimit: -100.0,
			amount:      -50.0,
			want:        false,
		},
		{
			name:        "limite negativo com reversão suficiente",
			creditLimit: -100.0,
			amount:      200.0,
			want:        true,
		},
		{
			name:        "limite negativo com reversão insuficiente",
			creditLimit: -100.0,
			amount:      50.0,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CreditCard{
				CreditLimit: tt.creditLimit,
			}
			if got := c.HasSufficientLimit(tt.amount); got != tt.want {
				t.Errorf("CreditCard.HasSufficientLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}
