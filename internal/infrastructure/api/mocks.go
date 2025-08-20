package api

import (
	"context"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockMovementUseCase struct {
	mock.Mock
}

func (m *MockMovementUseCase) Add(ctx context.Context, movement domain.Movement) (domain.Movement, error) {
	args := m.Called(ctx, movement)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementUseCase) FindByPeriod(ctx context.Context, period domain.Period) (domain.MovementList, error) {
	args := m.Called(ctx, period)
	return args.Get(0).([]domain.Movement), args.Error(1)
}

func (m *MockMovementUseCase) Pay(ctx context.Context, id uuid.UUID, date time.Time) (domain.Movement, error) {
	args := m.Called(ctx, id, date)
	return args.Get(0).(domain.Movement), args.Error(1)
}

func (m *MockMovementUseCase) RevertPay(ctx context.Context, id uuid.UUID) (domain.Movement, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Movement), args.Error(1)
}

type MockCreditCardUseCase struct {
	mock.Mock
}

func (m *MockCreditCardUseCase) Add(ctx context.Context, creditCard domain.CreditCard) (domain.CreditCard, error) {
	args := m.Called(ctx, creditCard)
	return args.Get(0).(domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardUseCase) FindAll(ctx context.Context) ([]domain.CreditCard, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardUseCase) FindByID(ctx context.Context, id uuid.UUID) (domain.CreditCard, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardUseCase) Update(ctx context.Context, id uuid.UUID, creditCard domain.CreditCard) (domain.CreditCard, error) {
	args := m.Called(ctx, id, creditCard)
	return args.Get(0).(domain.CreditCard), args.Error(1)
}

func (m *MockCreditCardUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCreditCardUseCase) FindWithOpenInvoices(ctx context.Context) ([]domain.CreditCardWithOpenInvoices, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.CreditCardWithOpenInvoices), args.Error(1)
}

type MockInvoiceUseCase struct {
	mock.Mock
}

func (m *MockInvoiceUseCase) FindDetailedInvoicesByPeriod(ctx context.Context, period domain.Period) ([]domain.DetailedInvoice, error) {
	args := m.Called(ctx, period)
	return args.Get(0).([]domain.DetailedInvoice), args.Error(1)
}

func (m *MockInvoiceUseCase) FindByMonth(ctx context.Context, date time.Time) ([]domain.Invoice, error) {
	args := m.Called(ctx, date)
	return args.Get(0).([]domain.Invoice), args.Error(1)
}

func (m *MockInvoiceUseCase) FindByID(ctx context.Context, id uuid.UUID) (domain.Invoice, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Invoice), args.Error(1)
}

func (m *MockInvoiceUseCase) Pay(ctx context.Context, id uuid.UUID, walletID uuid.UUID, paymentDate *time.Time) (domain.Invoice, error) {
	args := m.Called(ctx, id, walletID, paymentDate)
	return args.Get(0).(domain.Invoice), args.Error(1)
}
