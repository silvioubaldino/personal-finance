package usecase

import (
	"context"
	"regexp"
	"strings"

	"personal-finance/internal/domain"
)

type UserPreferencesInput struct {
	Language string `json:"language"`
	Currency string `json:"currency"`
}

type UserPreferencesRepository interface {
	GetOrCreateDefaults(ctx context.Context) (domain.UserPreferences, error)
	Upsert(ctx context.Context, prefs domain.UserPreferences) (domain.UserPreferences, error)
}

type UserPreferences struct {
	repo UserPreferencesRepository
}

func NewUserPreferences(repo UserPreferencesRepository) UserPreferences {
	return UserPreferences{
		repo: repo,
	}
}

func (u *UserPreferences) Get(ctx context.Context) (domain.UserPreferences, error) {
	return u.repo.GetOrCreateDefaults(ctx)
}

func (u *UserPreferences) Update(ctx context.Context, input UserPreferencesInput) (domain.UserPreferences, error) {
	if err := u.validateInput(input); err != nil {
		return domain.UserPreferences{}, err
	}

	prefs := domain.UserPreferences{
		Language: input.Language,
	}

	// Currency is optional for now; normalize to uppercase if provided
	if input.Currency != "" {
		prefs.Currency = strings.ToUpper(input.Currency)
	}

	return u.repo.Upsert(ctx, prefs)
}

func (u *UserPreferences) validateInput(input UserPreferencesInput) error {
	if err := u.validateLanguage(input.Language); err != nil {
		return err
	}

	// Currency is optional; only validate format if provided
	if input.Currency != "" {
		if err := u.validateCurrency(input.Currency); err != nil {
			return err
		}
	}

	return nil
}

// validateLanguage checks if the language follows a simple BCP47 format (e.g., pt-BR, en-US, es).
func (u *UserPreferences) validateLanguage(language string) error {
	if language == "" {
		return domain.WrapInvalidInput(ErrLanguageRequired, "language is required")
	}

	// Simple BCP47 pattern: 2-3 letter language code, optionally followed by hyphen and 2-letter region
	pattern := `^[a-zA-Z]{2,3}(-[a-zA-Z]{2})?$`
	matched, _ := regexp.MatchString(pattern, language)
	if !matched {
		return domain.WrapInvalidInput(ErrInvalidLanguageFormat, "language must be in BCP47 format (e.g., pt-BR, en-US)")
	}

	return nil
}

// validateCurrency checks if the currency follows ISO 4217 format (3 uppercase letters).
// Currency is optional; this should only be called when currency is provided.
func (u *UserPreferences) validateCurrency(currency string) error {
	// ISO 4217: exactly 3 letters
	pattern := `^[a-zA-Z]{3}$`
	matched, _ := regexp.MatchString(pattern, currency)
	if !matched {
		return domain.WrapInvalidInput(ErrInvalidCurrencyFormat, "currency must be ISO 4217 format (3 letters, e.g., BRL, USD)")
	}

	return nil
}
