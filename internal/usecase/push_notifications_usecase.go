package usecase

import (
	"context"
	"fmt"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/infrastructure/push"
	"personal-finance/internal/infrastructure/repository"

	"personal-finance/pkg/log"
)

const (
	pushTitle = "Lembrete de pagamento:"
)

type PushSender interface {
	Send(ctx context.Context, tokens []string, title, body string) (push.SendResult, error)
}

type PushMovementRepository interface {
	FindUnpaidByDate(ctx context.Context, date time.Time) ([]repository.UnpaidMovement, error)
}

type PushDeviceRepository interface {
	FindByUserIDs(ctx context.Context, userIDs []string) ([]domain.Device, error)
	DeleteByTokens(ctx context.Context, tokens []string) error
}

type PushNotifications struct {
	movementRepo PushMovementRepository
	deviceRepo   PushDeviceRepository
	pushSender   PushSender
}

func NewPushNotifications(
	movementRepo PushMovementRepository,
	deviceRepo PushDeviceRepository,
	pushSender PushSender,
) PushNotifications {
	return PushNotifications{
		movementRepo: movementRepo,
		deviceRepo:   deviceRepo,
		pushSender:   pushSender,
	}
}

type PushJobResult struct {
	MovementsFound int `json:"movements_found"`
	PushSent       int `json:"push_sent"`
	PushFailed     int `json:"push_failed"`
	InvalidTokens  int `json:"invalid_tokens"`
}

func (u *PushNotifications) SendDailyUnpaidPush(ctx context.Context, date time.Time) (PushJobResult, error) {
	result := PushJobResult{}

	unpaidMovements, err := u.movementRepo.FindUnpaidByDate(ctx, date)
	if err != nil {
		return result, fmt.Errorf("error finding unpaid movements: %w", err)
	}

	if len(unpaidMovements) == 0 {
		log.Info("no unpaid movements found for date", log.String("date", date.Format("2006-01-02")))
		return result, nil
	}

	result.MovementsFound = len(unpaidMovements)

	tokensByUserID, err := u.getTokensByUserID(ctx, unpaidMovements)
	if err != nil {
		return result, err
	}

	if len(tokensByUserID) == 0 {
		log.Info("no devices found for users with unpaid movements")
		return result, nil
	}

	invalidTokens := u.sendPushForMovements(ctx, unpaidMovements, tokensByUserID, &result)

	u.cleanupInvalidTokens(ctx, invalidTokens, &result)

	log.Info("push job completed",
		log.Int("movements_found", result.MovementsFound),
		log.Int("push_sent", result.PushSent),
		log.Int("push_failed", result.PushFailed),
		log.Int("invalid_tokens", result.InvalidTokens),
	)

	return result, nil
}

func (u *PushNotifications) getTokensByUserID(ctx context.Context, movements []repository.UnpaidMovement) (map[string][]string, error) {
	userIDs := extractUniqueUserIDs(movements)

	devices, err := u.deviceRepo.FindByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("error finding devices: %w", err)
	}

	tokensByUserID := make(map[string][]string)
	for _, device := range devices {
		tokensByUserID[device.UserID] = append(tokensByUserID[device.UserID], device.ExpoPushToken)
	}

	return tokensByUserID, nil
}

func (u *PushNotifications) sendPushForMovements(
	ctx context.Context,
	movements []repository.UnpaidMovement,
	tokensByUserID map[string][]string,
	result *PushJobResult,
) []string {

	var invalidTokens []string

	for _, movement := range movements {
		tokens, hasTokens := tokensByUserID[movement.UserID]
		if !hasTokens {
			continue
		}

		sendResult, err := u.pushSender.Send(ctx, tokens, pushTitle, movement.Description)
		if err != nil {
			log.Error("error sending push for movement",
				log.String("movement_id", movement.ID),
				log.Err(err),
			)
			result.PushFailed++
			continue
		}

		result.PushSent += sendResult.SuccessCount
		result.PushFailed += sendResult.FailureCount
		invalidTokens = append(invalidTokens, sendResult.InvalidTokens...)
	}

	return invalidTokens
}

func (u *PushNotifications) cleanupInvalidTokens(ctx context.Context, invalidTokens []string, result *PushJobResult) {
	if len(invalidTokens) == 0 {
		return
	}

	logger := log.FromContext(ctx)
	result.InvalidTokens = len(invalidTokens)

	if err := u.deviceRepo.DeleteByTokens(ctx, invalidTokens); err != nil {
		logger.Error("error deleting invalid tokens", log.Err(err))
	}
}

func extractUniqueUserIDs(movements []repository.UnpaidMovement) []string {
	seen := make(map[string]struct{})
	for _, movement := range movements {
		seen[movement.UserID] = struct{}{}
	}

	userIDs := make([]string, 0, len(seen))
	for userID := range seen {
		userIDs = append(userIDs, userID)
	}

	return userIDs
}
