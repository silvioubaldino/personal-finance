package api

import (
	"context"
	"net/http"
	"time"

	"personal-finance/internal/domain"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

type (
	PushNotificationsUseCase interface {
		SendDailyUnpaidPush(ctx context.Context, date time.Time) (usecase.PushJobResult, error)
	}

	PushNotificationsHandler struct {
		usecase PushNotificationsUseCase
	}

	PushJobResponse struct {
		MovementsFound int    `json:"movements_found"`
		PushSent       int    `json:"push_sent"`
		PushFailed     int    `json:"push_failed"`
		InvalidTokens  int    `json:"invalid_tokens"`
		Date           string `json:"date"`
	}
)

func NewPushNotificationsJobHandlers(jobsGroup *gin.RouterGroup, srv PushNotificationsUseCase) {
	handler := PushNotificationsHandler{
		usecase: srv,
	}

	jobsGroup.POST("/push-notifications", handler.SendDailyUnpaidPush())
}

func (h PushNotificationsHandler) SendDailyUnpaidPush() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		dateStr := c.Query("date")
		var date time.Time

		if dateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				HandleErr(c, ctx, domain.WrapInvalidInput(err, "invalid date format, use YYYY-MM-DD"))
				return
			}
			date = parsedDate
		} else {
			date = time.Now().UTC()
		}

		result, err := h.usecase.SendDailyUnpaidPush(ctx, date)
		if err != nil {
			HandleErr(c, ctx, err)
			return
		}

		c.JSON(http.StatusOK, PushJobResponse{
			MovementsFound: result.MovementsFound,
			PushSent:       result.PushSent,
			PushFailed:     result.PushFailed,
			InvalidTokens:  result.InvalidTokens,
			Date:           date.Format("2006-01-02"),
		})
	}
}
