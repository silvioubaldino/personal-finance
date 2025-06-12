package repository

import (
	"context"
	"fmt"

	"personal-finance/internal/plataform/authentication"

	"gorm.io/gorm"
)

func BuildBaseQuery(ctx context.Context, query *gorm.DB, tableName string) *gorm.DB {
	userID := ctx.Value(authentication.UserID).(string)

	return query.WithContext(ctx).
		Table(tableName).
		Where(fmt.Sprintf("%s.user_id = ?", tableName), userID)
}
