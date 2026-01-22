package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) Upsert(_ context.Context, device domain.Device) (domain.Device, error) {
	args := m.Called(device)
	return args.Get(0).(domain.Device), args.Error(1)
}

func (m *MockDeviceRepository) FindByUserID(_ context.Context) ([]domain.Device, error) {
	args := m.Called()
	return args.Get(0).([]domain.Device), args.Error(1)
}

func (m *MockDeviceRepository) DeleteByToken(_ context.Context, token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func TestDevice_Upsert(t *testing.T) {
	now := time.Now()
	deviceID := uuid.New()

	tests := map[string]struct {
		input          DeviceInput
		mockSetup      func(mock *MockDeviceRepository)
		expectedDevice domain.Device
		expectedErr    error
	}{
		"should upsert device successfully": {
			input: DeviceInput{
				ExpoPushToken: "ExponentPushToken[xxxx]",
				Platform:      "ios",
			},
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("Upsert", domain.Device{
					ExpoPushToken: "ExponentPushToken[xxxx]",
					Platform:      domain.PlatformIOS,
				}).Return(domain.Device{
					ID:            deviceID,
					UserID:        "user-123",
					ExpoPushToken: "ExponentPushToken[xxxx]",
					Platform:      domain.PlatformIOS,
					DateCreate:    now,
					DateUpdate:    now,
					LastSeenAt:    &now,
				}, nil)
			},
			expectedDevice: domain.Device{
				ID:            deviceID,
				UserID:        "user-123",
				ExpoPushToken: "ExponentPushToken[xxxx]",
				Platform:      domain.PlatformIOS,
				DateCreate:    now,
				DateUpdate:    now,
				LastSeenAt:    &now,
			},
			expectedErr: nil,
		},
		"should upsert android device successfully": {
			input: DeviceInput{
				ExpoPushToken: "ExponentPushToken[yyyy]",
				Platform:      "android",
			},
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("Upsert", domain.Device{
					ExpoPushToken: "ExponentPushToken[yyyy]",
					Platform:      domain.PlatformAndroid,
				}).Return(domain.Device{
					ID:            deviceID,
					UserID:        "user-123",
					ExpoPushToken: "ExponentPushToken[yyyy]",
					Platform:      domain.PlatformAndroid,
					DateCreate:    now,
					DateUpdate:    now,
					LastSeenAt:    &now,
				}, nil)
			},
			expectedDevice: domain.Device{
				ID:            deviceID,
				UserID:        "user-123",
				ExpoPushToken: "ExponentPushToken[yyyy]",
				Platform:      domain.PlatformAndroid,
				DateCreate:    now,
				DateUpdate:    now,
				LastSeenAt:    &now,
			},
			expectedErr: nil,
		},
		"should return error when token is empty": {
			input: DeviceInput{
				ExpoPushToken: "",
				Platform:      "ios",
			},
			mockSetup:      func(mockRepo *MockDeviceRepository) {},
			expectedDevice: domain.Device{},
			expectedErr:    domain.WrapInvalidInput(ErrEmptyToken, "expo_push_token is required"),
		},
		"should return error when platform is invalid": {
			input: DeviceInput{
				ExpoPushToken: "ExponentPushToken[xxxx]",
				Platform:      "windows",
			},
			mockSetup:      func(mockRepo *MockDeviceRepository) {},
			expectedDevice: domain.Device{},
			expectedErr:    domain.WrapInvalidInput(ErrInvalidPlatform, "platform must be 'ios' or 'android'"),
		},
		"should return error when repository fails": {
			input: DeviceInput{
				ExpoPushToken: "ExponentPushToken[xxxx]",
				Platform:      "ios",
			},
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("Upsert", domain.Device{
					ExpoPushToken: "ExponentPushToken[xxxx]",
					Platform:      domain.PlatformIOS,
				}).Return(domain.Device{}, errors.New("database error"))
			},
			expectedDevice: domain.Device{},
			expectedErr:    errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(MockDeviceRepository)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			uc := NewDevice(mockRepo)
			ctx := context.Background()

			result, err := uc.Upsert(ctx, tt.input)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDevice, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDevice_List(t *testing.T) {
	now := time.Now()
	deviceID1 := uuid.New()
	deviceID2 := uuid.New()

	tests := map[string]struct {
		mockSetup       func(mock *MockDeviceRepository)
		expectedDevices []domain.Device
		expectedErr     error
	}{
		"should list devices successfully": {
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("FindByUserID").Return([]domain.Device{
					{
						ID:            deviceID1,
						UserID:        "user-123",
						ExpoPushToken: "ExponentPushToken[xxxx]",
						Platform:      domain.PlatformIOS,
						DateCreate:    now,
						DateUpdate:    now,
						LastSeenAt:    &now,
					},
					{
						ID:            deviceID2,
						UserID:        "user-123",
						ExpoPushToken: "ExponentPushToken[yyyy]",
						Platform:      domain.PlatformAndroid,
						DateCreate:    now,
						DateUpdate:    now,
						LastSeenAt:    &now,
					},
				}, nil)
			},
			expectedDevices: []domain.Device{
				{
					ID:            deviceID1,
					UserID:        "user-123",
					ExpoPushToken: "ExponentPushToken[xxxx]",
					Platform:      domain.PlatformIOS,
					DateCreate:    now,
					DateUpdate:    now,
					LastSeenAt:    &now,
				},
				{
					ID:            deviceID2,
					UserID:        "user-123",
					ExpoPushToken: "ExponentPushToken[yyyy]",
					Platform:      domain.PlatformAndroid,
					DateCreate:    now,
					DateUpdate:    now,
					LastSeenAt:    &now,
				},
			},
			expectedErr: nil,
		},
		"should return empty list when no devices": {
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("FindByUserID").Return([]domain.Device{}, nil)
			},
			expectedDevices: []domain.Device{},
			expectedErr:     nil,
		},
		"should return error when repository fails": {
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("FindByUserID").Return([]domain.Device{}, errors.New("database error"))
			},
			expectedDevices: []domain.Device{},
			expectedErr:     errors.New("database error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(MockDeviceRepository)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			uc := NewDevice(mockRepo)
			ctx := context.Background()

			result, err := uc.List(ctx)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDevices, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDevice_Delete(t *testing.T) {
	tests := map[string]struct {
		token       string
		mockSetup   func(mock *MockDeviceRepository)
		expectedErr error
	}{
		"should delete device successfully": {
			token: "ExponentPushToken[xxxx]",
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("DeleteByToken", "ExponentPushToken[xxxx]").Return(nil)
			},
			expectedErr: nil,
		},
		"should return error when token is empty": {
			token:       "",
			mockSetup:   func(mockRepo *MockDeviceRepository) {},
			expectedErr: domain.WrapInvalidInput(ErrEmptyToken, "token is required"),
		},
		"should return error when repository fails": {
			token: "ExponentPushToken[xxxx]",
			mockSetup: func(mockRepo *MockDeviceRepository) {
				mockRepo.On("DeleteByToken", "ExponentPushToken[xxxx]").Return(errors.New("not found"))
			},
			expectedErr: errors.New("not found"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := new(MockDeviceRepository)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			uc := NewDevice(mockRepo)
			ctx := context.Background()

			err := uc.Delete(ctx, tt.token)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
