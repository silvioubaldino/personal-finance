package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/push"
	"personal-finance/internal/infrastructure/repository"
	"personal-finance/pkg/log"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPushMovementRepository struct {
	mock.Mock
}

func (m *MockPushMovementRepository) FindUnpaidByDate(_ context.Context, date time.Time) ([]repository.UnpaidMovement, error) {
	args := m.Called(date)
	return args.Get(0).([]repository.UnpaidMovement), args.Error(1)
}

type MockPushDeviceRepository struct {
	mock.Mock
}

func (m *MockPushDeviceRepository) FindByUserIDs(_ context.Context, userIDs []string) ([]domain.Device, error) {
	args := m.Called(userIDs)
	return args.Get(0).([]domain.Device), args.Error(1)
}

func (m *MockPushDeviceRepository) DeleteByTokens(_ context.Context, tokens []string) error {
	args := m.Called(tokens)
	return args.Error(0)
}

type MockPushSender struct {
	mock.Mock
}

func (m *MockPushSender) Send(_ context.Context, tokens []string, title, body string) (push.SendResult, error) {
	args := m.Called(tokens, title, body)
	return args.Get(0).(push.SendResult), args.Error(1)
}

func TestPushNotifications_SendDailyUnpaidPush(t *testing.T) {
	log.Initialize()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	deviceID := uuid.New()

	tests := map[string]struct {
		mockSetup      func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender)
		expectedResult PushJobResult
		expectedErr    error
	}{
		"should send push for each movement": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{
					{ID: "mov-1", Description: "Aluguel", UserID: "user-1"},
					{ID: "mov-2", Description: "Internet", UserID: "user-1"},
					{ID: "mov-3", Description: "Luz", UserID: "user-2"},
				}, nil)

				devRepo.On("FindByUserIDs", mock.AnythingOfType("[]string")).Return([]domain.Device{
					{ID: deviceID, UserID: "user-1", ExpoPushToken: "token-1"},
					{ID: deviceID, UserID: "user-2", ExpoPushToken: "token-2"},
				}, nil)

				sender.On("Send", []string{"token-1"}, "Lembrete de pagamento", "Aluguel").
					Return(push.SendResult{SuccessCount: 1}, nil).Maybe()
				sender.On("Send", []string{"token-1"}, "Lembrete de pagamento", "Internet").
					Return(push.SendResult{SuccessCount: 1}, nil).Maybe()
				sender.On("Send", []string{"token-2"}, "Lembrete de pagamento", "Luz").
					Return(push.SendResult{SuccessCount: 1}, nil).Maybe()
			},
			expectedResult: PushJobResult{
				MovementsFound: 3,
				PushSent:       3,
				PushFailed:     0,
				InvalidTokens:  0,
			},
			expectedErr: nil,
		},
		"should return empty result when no unpaid movements": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{}, nil)
			},
			expectedResult: PushJobResult{},
			expectedErr:    nil,
		},
		"should return empty result when no devices found": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{
					{ID: "mov-1", Description: "Aluguel", UserID: "user-1"},
				}, nil)

				devRepo.On("FindByUserIDs", mock.AnythingOfType("[]string")).Return([]domain.Device{}, nil)
			},
			expectedResult: PushJobResult{
				MovementsFound: 1,
			},
			expectedErr: nil,
		},
		"should handle invalid tokens and delete them": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{
					{ID: "mov-1", Description: "Aluguel", UserID: "user-1"},
				}, nil)

				devRepo.On("FindByUserIDs", mock.AnythingOfType("[]string")).Return([]domain.Device{
					{ID: deviceID, UserID: "user-1", ExpoPushToken: "invalid-token"},
				}, nil)

				sender.On("Send", []string{"invalid-token"}, "Lembrete de pagamento", "Aluguel").
					Return(push.SendResult{
						SuccessCount:  0,
						FailureCount:  1,
						InvalidTokens: []string{"invalid-token"},
					}, nil)

				devRepo.On("DeleteByTokens", []string{"invalid-token"}).Return(nil)
			},
			expectedResult: PushJobResult{
				MovementsFound: 1,
				PushSent:       0,
				PushFailed:     1,
				InvalidTokens:  1,
			},
			expectedErr: nil,
		},
		"should return error when movement repository fails": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{}, errors.New("database error"))
			},
			expectedResult: PushJobResult{},
			expectedErr:    errors.New("error finding unpaid movements: database error"),
		},
		"should return error when device repository fails": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{
					{ID: "mov-1", Description: "Aluguel", UserID: "user-1"},
				}, nil)

				devRepo.On("FindByUserIDs", mock.AnythingOfType("[]string")).Return([]domain.Device{}, errors.New("database error"))
			},
			expectedResult: PushJobResult{
				MovementsFound: 1,
			},
			expectedErr: errors.New("error finding devices: database error"),
		},
		"should continue when push sender fails for one movement": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{
					{ID: "mov-1", Description: "Aluguel", UserID: "user-1"},
					{ID: "mov-2", Description: "Internet", UserID: "user-1"},
				}, nil)

				devRepo.On("FindByUserIDs", mock.AnythingOfType("[]string")).Return([]domain.Device{
					{ID: deviceID, UserID: "user-1", ExpoPushToken: "token-1"},
				}, nil)

				sender.On("Send", []string{"token-1"}, "Lembrete de pagamento", "Aluguel").
					Return(push.SendResult{}, errors.New("network error"))
				sender.On("Send", []string{"token-1"}, "Lembrete de pagamento", "Internet").
					Return(push.SendResult{SuccessCount: 1}, nil)
			},
			expectedResult: PushJobResult{
				MovementsFound: 2,
				PushSent:       1,
				PushFailed:     1,
				InvalidTokens:  0,
			},
			expectedErr: nil,
		},
		"should send to multiple devices for same user": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{
					{ID: "mov-1", Description: "Aluguel", UserID: "user-1"},
				}, nil)

				devRepo.On("FindByUserIDs", mock.AnythingOfType("[]string")).Return([]domain.Device{
					{ID: deviceID, UserID: "user-1", ExpoPushToken: "token-1"},
					{ID: uuid.New(), UserID: "user-1", ExpoPushToken: "token-2"},
				}, nil)

				sender.On("Send", []string{"token-1", "token-2"}, "Lembrete de pagamento", "Aluguel").
					Return(push.SendResult{SuccessCount: 2}, nil)
			},
			expectedResult: PushJobResult{
				MovementsFound: 1,
				PushSent:       2,
				PushFailed:     0,
				InvalidTokens:  0,
			},
			expectedErr: nil,
		},
		"should skip movement if user has no device": {
			mockSetup: func(movRepo *MockPushMovementRepository, devRepo *MockPushDeviceRepository, sender *MockPushSender) {
				movRepo.On("FindUnpaidByDate", date).Return([]repository.UnpaidMovement{
					{ID: "mov-1", Description: "Aluguel", UserID: "user-1"},
					{ID: "mov-2", Description: "Internet", UserID: "user-2"},
				}, nil)

				devRepo.On("FindByUserIDs", mock.AnythingOfType("[]string")).Return([]domain.Device{
					{ID: deviceID, UserID: "user-1", ExpoPushToken: "token-1"},
				}, nil)

				sender.On("Send", []string{"token-1"}, "Lembrete de pagamento", "Aluguel").
					Return(push.SendResult{SuccessCount: 1}, nil)
			},
			expectedResult: PushJobResult{
				MovementsFound: 2,
				PushSent:       1,
				PushFailed:     0,
				InvalidTokens:  0,
			},
			expectedErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			movRepo := new(MockPushMovementRepository)
			devRepo := new(MockPushDeviceRepository)
			sender := new(MockPushSender)

			if tt.mockSetup != nil {
				tt.mockSetup(movRepo, devRepo, sender)
			}

			uc := NewPushNotifications(movRepo, devRepo, sender)
			ctx := context.Background()

			result, err := uc.SendDailyUnpaidPush(ctx, date)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
			movRepo.AssertExpectations(t)
			devRepo.AssertExpectations(t)
			sender.AssertExpectations(t)
		})
	}
}
