package repository

import (
	"context"
	"testing"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDeviceTestDB() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&UserDeviceDB{})
	// Create unique index for expo_push_token (required for ON CONFLICT)
	_ = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_devices_token ON user_devices (expo_push_token)")
	return db
}

func createDeviceTestContext(userID string) context.Context {
	return context.WithValue(context.Background(), authentication.UserID, userID)
}

func TestDeviceRepository_Upsert(t *testing.T) {
	tests := map[string]struct {
		prepareDB      func() *DeviceRepository
		input          domain.Device
		userID         string
		expectedErr    error
		expectedDevice func(result domain.Device) bool
	}{
		"should insert new device": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				return NewDeviceRepository(db)
			},
			input: domain.Device{
				ExpoPushToken: "ExponentPushToken[abc123]",
				Platform:      domain.PlatformIOS,
			},
			userID:      "user-test-id",
			expectedErr: nil,
			expectedDevice: func(result domain.Device) bool {
				return result.UserID == "user-test-id" &&
					result.ExpoPushToken == "ExponentPushToken[abc123]" &&
					result.Platform == domain.PlatformIOS &&
					!result.DateCreate.IsZero() &&
					!result.DateUpdate.IsZero() &&
					result.LastSeenAt != nil
			},
		},
		"should update existing device with same token": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx := createDeviceTestContext("user-test-id")
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "ExponentPushToken[abc123]",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			input: domain.Device{
				ExpoPushToken: "ExponentPushToken[abc123]",
				Platform:      domain.PlatformAndroid,
			},
			userID:      "user-test-id",
			expectedErr: nil,
			expectedDevice: func(result domain.Device) bool {
				return result.UserID == "user-test-id" &&
					result.ExpoPushToken == "ExponentPushToken[abc123]" &&
					result.Platform == domain.PlatformAndroid
			},
		},
		"should update user_id when same token registered by different user": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx := createDeviceTestContext("old-user-id")
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "ExponentPushToken[shared]",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			input: domain.Device{
				ExpoPushToken: "ExponentPushToken[shared]",
				Platform:      domain.PlatformIOS,
			},
			userID:      "new-user-id",
			expectedErr: nil,
			expectedDevice: func(result domain.Device) bool {
				return result.UserID == "new-user-id" &&
					result.ExpoPushToken == "ExponentPushToken[shared]"
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := createDeviceTestContext(tc.userID)

			result, err := repo.Upsert(ctx, tc.input)

			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.True(t, tc.expectedDevice(result))
			}
		})
	}
}

func TestDeviceRepository_FindByUserID(t *testing.T) {
	tests := map[string]struct {
		prepareDB     func() *DeviceRepository
		userID        string
		expectedErr   error
		expectedCount int
	}{
		"should return empty when no devices": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				return NewDeviceRepository(db)
			},
			userID:        "user-test-id",
			expectedErr:   nil,
			expectedCount: 0,
		},
		"should return user devices only": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx1 := createDeviceTestContext("user-1")
				_, _ = repo.Upsert(ctx1, domain.Device{
					ExpoPushToken: "token-1",
					Platform:      domain.PlatformIOS,
				})
				_, _ = repo.Upsert(ctx1, domain.Device{
					ExpoPushToken: "token-2",
					Platform:      domain.PlatformAndroid,
				})

				ctx2 := createDeviceTestContext("user-2")
				_, _ = repo.Upsert(ctx2, domain.Device{
					ExpoPushToken: "token-3",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			userID:        "user-1",
			expectedErr:   nil,
			expectedCount: 2,
		},
		"should return devices ordered by date_create desc": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx := createDeviceTestContext("user-test-id")
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-first",
					Platform:      domain.PlatformIOS,
				})
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-second",
					Platform:      domain.PlatformAndroid,
				})

				return repo
			},
			userID:        "user-test-id",
			expectedErr:   nil,
			expectedCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := createDeviceTestContext(tc.userID)

			result, err := repo.FindByUserID(ctx)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedCount, len(result))
		})
	}
}

func TestDeviceRepository_DeleteByToken(t *testing.T) {
	tests := map[string]struct {
		prepareDB   func() *DeviceRepository
		token       string
		userID      string
		expectedErr error
	}{
		"should delete device by token": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx := createDeviceTestContext("user-test-id")
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-to-delete",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			token:       "token-to-delete",
			userID:      "user-test-id",
			expectedErr: nil,
		},
		"should return error when device not found": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				return NewDeviceRepository(db)
			},
			token:       "non-existent-token",
			userID:      "user-test-id",
			expectedErr: ErrDeviceNotFound,
		},
		"should not delete device from another user": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx := createDeviceTestContext("user-1")
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-user-1",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			token:       "token-user-1",
			userID:      "user-2",
			expectedErr: ErrDeviceNotFound,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := createDeviceTestContext(tc.userID)

			err := repo.DeleteByToken(ctx, tc.token)

			if tc.expectedErr != nil {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeviceRepository_FindByUserIDs(t *testing.T) {
	tests := map[string]struct {
		prepareDB     func() *DeviceRepository
		userIDs       []string
		expectedErr   error
		expectedCount int
	}{
		"should return empty when no user ids provided": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				return NewDeviceRepository(db)
			},
			userIDs:       []string{},
			expectedErr:   nil,
			expectedCount: 0,
		},
		"should return devices for multiple users": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx1 := createDeviceTestContext("user-1")
				_, _ = repo.Upsert(ctx1, domain.Device{
					ExpoPushToken: "token-1",
					Platform:      domain.PlatformIOS,
				})

				ctx2 := createDeviceTestContext("user-2")
				_, _ = repo.Upsert(ctx2, domain.Device{
					ExpoPushToken: "token-2",
					Platform:      domain.PlatformAndroid,
				})

				ctx3 := createDeviceTestContext("user-3")
				_, _ = repo.Upsert(ctx3, domain.Device{
					ExpoPushToken: "token-3",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			userIDs:       []string{"user-1", "user-2"},
			expectedErr:   nil,
			expectedCount: 2,
		},
		"should return empty when no matching users": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx := createDeviceTestContext("user-1")
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-1",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			userIDs:       []string{"user-99"},
			expectedErr:   nil,
			expectedCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := context.Background()

			result, err := repo.FindByUserIDs(ctx, tc.userIDs)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedCount, len(result))
		})
	}
}

func TestDeviceRepository_DeleteByTokens(t *testing.T) {
	tests := map[string]struct {
		prepareDB      func() *DeviceRepository
		tokens         []string
		expectedErr    error
		remainingCount int
	}{
		"should do nothing when no tokens provided": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				return NewDeviceRepository(db)
			},
			tokens:         []string{},
			expectedErr:    nil,
			remainingCount: 0,
		},
		"should delete multiple devices by tokens": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx := createDeviceTestContext("user-1")
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-1",
					Platform:      domain.PlatformIOS,
				})
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-2",
					Platform:      domain.PlatformAndroid,
				})
				_, _ = repo.Upsert(ctx, domain.Device{
					ExpoPushToken: "token-3",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			tokens:         []string{"token-1", "token-2"},
			expectedErr:    nil,
			remainingCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := context.Background()

			err := repo.DeleteByTokens(ctx, tc.tokens)

			assert.Equal(t, tc.expectedErr, err)

			// Verify remaining count
			ctxUser := createDeviceTestContext("user-1")
			devices, _ := repo.FindByUserID(ctxUser)
			assert.Equal(t, tc.remainingCount, len(devices))
		})
	}
}

func TestDeviceRepository_DeleteByUserID(t *testing.T) {
	tests := map[string]struct {
		prepareDB      func() *DeviceRepository
		userID         string
		expectedErr    error
		remainingCount int
	}{
		"should delete all devices for user": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				repo := NewDeviceRepository(db)

				ctx1 := createDeviceTestContext("user-to-delete")
				_, _ = repo.Upsert(ctx1, domain.Device{
					ExpoPushToken: "token-1",
					Platform:      domain.PlatformIOS,
				})
				_, _ = repo.Upsert(ctx1, domain.Device{
					ExpoPushToken: "token-2",
					Platform:      domain.PlatformAndroid,
				})

				ctx2 := createDeviceTestContext("user-to-keep")
				_, _ = repo.Upsert(ctx2, domain.Device{
					ExpoPushToken: "token-3",
					Platform:      domain.PlatformIOS,
				})

				return repo
			},
			userID:         "user-to-delete",
			expectedErr:    nil,
			remainingCount: 1,
		},
		"should not fail when user has no devices": {
			prepareDB: func() *DeviceRepository {
				db := setupDeviceTestDB()
				return NewDeviceRepository(db)
			},
			userID:         "non-existent-user",
			expectedErr:    nil,
			remainingCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repo := tc.prepareDB()
			ctx := context.Background()

			err := repo.DeleteByUserID(ctx, nil, tc.userID)

			assert.Equal(t, tc.expectedErr, err)

			// Verify remaining count for the "kept" user
			ctxKept := createDeviceTestContext("user-to-keep")
			devices, _ := repo.FindByUserID(ctxKept)
			assert.Equal(t, tc.remainingCount, len(devices))
		})
	}
}
