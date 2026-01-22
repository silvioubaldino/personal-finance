package pushnotifications

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/infrastructure/push"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func SetupJobs(jobsGroup *gin.RouterGroup, registry *registry.Registry) {
	movementRepo := registry.GetMovementRepository()
	deviceRepo := registry.GetDeviceRepository()
	expoClient := push.NewExpoClient()

	pushService := usecase.NewPushNotifications(movementRepo, deviceRepo, expoClient)

	api.NewPushNotificationsJobHandlers(jobsGroup, &pushService)
}
