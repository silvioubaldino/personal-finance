package agent

import (
	"personal-finance/internal/bootstrap/registry"
	"personal-finance/internal/infrastructure/api"
	"personal-finance/internal/infrastructure/gateway"
	"personal-finance/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, reg *registry.Registry) {
	// Repositories
	memoryRepo := reg.GetAgentMemoryRepository()
	convRepo := reg.GetAgentConversationRepository()
	auditRepo := reg.GetAgentAuditRepository()
	financialRepo := reg.GetAgentFinancialRepository()

	// Gateway: ADK + Vertex AI
	agentGateway := gateway.NewADKAgentGateway(memoryRepo, financialRepo)

	// Use case
	agentUseCase := usecase.NewAgentUseCase(
		memoryRepo,
		convRepo,
		auditRepo,
		agentGateway,
	)

	// API handlers (authenticated routes)
	api.NewAgentHandlers(r, agentUseCase)
}

func SetupJobs(jobsGroup *gin.RouterGroup, reg *registry.Registry) {
	memoryRepo := reg.GetAgentMemoryRepository()
	convRepo := reg.GetAgentConversationRepository()
	auditRepo := reg.GetAgentAuditRepository()
	financialRepo := reg.GetAgentFinancialRepository()
	agentGateway := gateway.NewADKAgentGateway(memoryRepo, financialRepo)

	agentUseCase := usecase.NewAgentUseCase(
		memoryRepo,
		convRepo,
		auditRepo,
		agentGateway,
	)

	api.NewAgentJobHandlers(jobsGroup, agentUseCase)
}
