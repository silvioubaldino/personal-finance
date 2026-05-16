package usecase

import (
	"context"
	"regexp"
	"strings"

	"personal-finance/internal/domain"
)

type UserInput struct {
	Language string `json:"language"`
	Currency string `json:"currency"`
}

type UserRepository interface {
	Get(ctx context.Context) (domain.User, error)
	Update(ctx context.Context, user domain.User) (domain.User, error)
}

type User struct {
	repo UserRepository
}

func NewUser(repo UserRepository) User {
	return User{repo: repo}
}

func (u *User) Get(ctx context.Context) (domain.User, error) {
	return u.repo.Get(ctx)
}

func (u *User) Update(ctx context.Context, input UserInput) (domain.User, error) {
	if err := u.validateInput(input); err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		Language: input.Language,
	}

	if input.Currency != "" {
		user.Currency = strings.ToUpper(input.Currency)
	}

	return u.repo.Update(ctx, user)
}

func (u *User) validateInput(input UserInput) error {
	if input.Language != "" {
		if err := u.validateLanguage(input.Language); err != nil {
			return err
		}
	}

	if input.Currency != "" {
		if err := u.validateCurrency(input.Currency); err != nil {
			return err
		}
	}

	return nil
}

// validateLanguage checks a simple BCP47 pattern (e.g., pt-BR, en-US, es).
func (u *User) validateLanguage(language string) error {
	pattern := `^[a-zA-Z]{2,3}(-[a-zA-Z]{2})?$`
	matched, _ := regexp.MatchString(pattern, language)
	if !matched {
		return domain.WrapInvalidInput(ErrInvalidLanguageFormat, "language must be in BCP47 format (e.g., pt-BR, en-US)")
	}

	return nil
}

// validateCurrency checks ISO 4217 (3 letters).
func (u *User) validateCurrency(currency string) error {
	pattern := `^[a-zA-Z]{3}$`
	matched, _ := regexp.MatchString(pattern, currency)
	if !matched {
		return domain.WrapInvalidInput(ErrInvalidCurrencyFormat, "currency must be ISO 4217 format (3 letters, e.g., BRL, USD)")
	}

	return nil
}
