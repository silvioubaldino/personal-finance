package usecase

import (
	"context"

	"personal-finance/internal/domain"
)

type DeviceInput struct {
	ExpoPushToken string `json:"expo_push_token"`
	Platform      string `json:"platform"`
}

type DeviceRepository interface {
	Upsert(ctx context.Context, device domain.Device) (domain.Device, error)
	FindByUserID(ctx context.Context) ([]domain.Device, error)
	DeleteByToken(ctx context.Context, token string) error
}

type Device struct {
	repo DeviceRepository
}

func NewDevice(repo DeviceRepository) Device {
	return Device{
		repo: repo,
	}
}

func (u *Device) Upsert(ctx context.Context, input DeviceInput) (domain.Device, error) {
	if err := u.validateInput(input); err != nil {
		return domain.Device{}, err
	}

	device := domain.Device{
		ExpoPushToken: input.ExpoPushToken,
		Platform:      domain.Platform(input.Platform),
	}

	return u.repo.Upsert(ctx, device)
}

func (u *Device) List(ctx context.Context) ([]domain.Device, error) {
	return u.repo.FindByUserID(ctx)
}

func (u *Device) Delete(ctx context.Context, token string) error {
	if token == "" {
		return domain.WrapInvalidInput(ErrEmptyToken, "token is required")
	}

	return u.repo.DeleteByToken(ctx, token)
}

func (u *Device) validateInput(input DeviceInput) error {
	if input.ExpoPushToken == "" {
		return domain.WrapInvalidInput(ErrEmptyToken, "expo_push_token is required")
	}

	platform := domain.Platform(input.Platform)
	if !platform.IsValid() {
		return domain.WrapInvalidInput(ErrInvalidPlatform, "platform must be 'ios' or 'android'")
	}

	return nil
}
