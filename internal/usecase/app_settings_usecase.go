package usecase

import (
	"context"
	"errors"
	"fmt"
)

var ErrInvalidPlusPrice = errors.New("invalid plus price")

type AppSettingsReader interface {
	GetFloat(ctx context.Context, key string) (float64, error)
	SetFloat(ctx context.Context, key string, value float64) error
}

type PlusPriceResponse struct {
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

type AppSettings struct {
	settingsRepo AppSettingsReader
}

func NewAppSettings(settingsRepo AppSettingsReader) *AppSettings {
	return &AppSettings{settingsRepo: settingsRepo}
}

func (s *AppSettings) GetPlusPrice(ctx context.Context) (PlusPriceResponse, error) {
	price, err := s.settingsRepo.GetFloat(ctx, "plus_price")
	if err != nil {
		return PlusPriceResponse{}, fmt.Errorf("error reading plus_price: %w", err)
	}
	return PlusPriceResponse{Price: price, Currency: "BRL"}, nil
}

func (s *AppSettings) SetPlusPrice(ctx context.Context, price float64) error {
	if price <= 0 {
		return fmt.Errorf("%w: price must be positive", ErrInvalidPlusPrice)
	}
	return s.settingsRepo.SetFloat(ctx, "plus_price", price)
}
