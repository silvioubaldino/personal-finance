package usecase

import (
	"context"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/google/uuid"
)

type UserConsentInput struct {
	TermVersion string `json:"term_version"`
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
}

type UserConsentRepository interface {
	Save(ctx context.Context, consent domain.UserConsent) (domain.UserConsent, error)
	FindByUserID(ctx context.Context) ([]domain.UserConsent, error)
	FindLatestByTermVersion(ctx context.Context, termVersion string) (domain.UserConsent, error)
	HasConsentedToVersion(ctx context.Context, termVersion string) (bool, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.UserConsent, error)
}

type UserConsent struct {
	repo UserConsentRepository
}

func NewUserConsent(repo UserConsentRepository) UserConsent {
	return UserConsent{
		repo: repo,
	}
}

func (u *UserConsent) RecordConsent(ctx context.Context, input UserConsentInput) (domain.UserConsent, error) {
	if err := u.validateInput(input); err != nil {
		return domain.UserConsent{}, err
	}

	userID := ctx.Value(authentication.UserID).(string)

	consent := domain.NewUserConsent(
		userID,
		input.TermVersion,
		input.IPAddress,
		input.UserAgent,
	)

	return u.repo.Save(ctx, consent)
}

func (u *UserConsent) GetAllConsents(ctx context.Context) ([]domain.UserConsent, error) {
	return u.repo.FindByUserID(ctx)
}

func (u *UserConsent) HasConsentedToVersion(ctx context.Context, termVersion string) (bool, error) {
	if termVersion == "" {
		return false, domain.WrapInvalidInput(ErrTermVersionRequired, "term_version is required")
	}

	return u.repo.HasConsentedToVersion(ctx, termVersion)
}

func (u *UserConsent) GetConsentByID(ctx context.Context, id uuid.UUID) (domain.UserConsent, error) {
	return u.repo.FindByID(ctx, id)
}

func (u *UserConsent) validateInput(input UserConsentInput) error {
	if input.TermVersion == "" {
		return domain.WrapInvalidInput(ErrTermVersionRequired, "term_version is required")
	}

	return nil
}
