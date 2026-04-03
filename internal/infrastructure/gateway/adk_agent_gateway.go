package gateway

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"
	"personal-finance/pkg/log"
)

const (
	defaultModel    = "gemini-1.5-flash"
	defaultLocation = "southamerica-east1"
	appName         = "personal_finance_agent"
)

// ADKAgentGateway implements domain.AgentGateway using Google ADK + Vertex AI.
// ADK lives ONLY here — never imported by domain or usecase layers.
type ADKAgentGateway struct {
	memoryRepo    MemoryRepository
	financialRepo FinancialRepository
	projectID     string
	location      string
	modelName     string
}

// MemoryRepository is the minimal interface the gateway needs to execute memory tool calls.
type MemoryRepository interface {
	Save(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.AgentMemory, error)
	Update(ctx context.Context, memory domain.AgentMemory) (domain.AgentMemory, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SearchByContent(ctx context.Context, userID string, query string) ([]domain.AgentMemory, error)
	FindByUserID(ctx context.Context, userID string) ([]domain.AgentMemory, error)
}

// FinancialRepository is the minimal interface the gateway needs to execute financial query tools.
type FinancialRepository interface {
	GetFinancialOverview(ctx context.Context, month, year int) (domain.AgentFinancialOverview, error)
	GetSpendingBreakdown(ctx context.Context, month, year int) (domain.AgentSpendingBreakdown, error)
	GetCreditCardsSummary(ctx context.Context) (domain.AgentCreditCardsSummary, error)
	GetMovements(ctx context.Context, month, year, limit int) (domain.AgentMovementsList, error)
	GetRecurringSummary(ctx context.Context) (domain.AgentRecurringSummary, error)
	GetBudgetStatus(ctx context.Context, month, year int) (domain.AgentBudgetStatus, error)
}

// NewADKAgentGateway creates a new ADKAgentGateway.
func NewADKAgentGateway(memoryRepo MemoryRepository, financialRepo FinancialRepository) *ADKAgentGateway {
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = defaultLocation
	}

	modelName := os.Getenv("VERTEX_MODEL")
	if modelName == "" {
		modelName = defaultModel
	}

	return &ADKAgentGateway{
		memoryRepo:    memoryRepo,
		financialRepo: financialRepo,
		projectID:     os.Getenv("GOOGLE_PROJECT_ID"),
		location:      location,
		modelName:     modelName,
	}
}

// --- Tool argument/result DTOs ---

type saveMemoryArgs struct {
	// MemoryType must be one of: goal, fact, constraint, insight, commitment, risk_profile, life_event
	MemoryType string         `json:"memory_type"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type saveMemoryResult struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type updateMemoryArgs struct {
	ID       string         `json:"id"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type updateMemoryResult struct {
	Message string `json:"message"`
}

type deleteMemoryArgs struct {
	ID string `json:"id"`
}

type deleteMemoryResult struct {
	Message string `json:"message"`
}

type searchMemoriesArgs struct {
	Query string `json:"query,omitempty"`
}

type searchMemoriesResult struct {
	Memories []memoryItem `json:"memories"`
	Count    int          `json:"count"`
}

type memoryItem struct {
	ID         string `json:"id"`
	MemoryType string `json:"memory_type"`
	Content    string `json:"content"`
}

// --- Chat implementation ---

// Chat sends the user message to Vertex AI via ADK and returns the agent response.
func (g *ADKAgentGateway) Chat(
	ctx context.Context,
	systemPrompt string,
	userMessage string,
	history []domain.AgentMessage,
) (domain.AgentGatewayResponse, error) {

	// 1. Build the Gemini model pointing to Vertex AI
	geminiModel, err := gemini.NewModel(ctx, g.modelName, &genai.ClientConfig{
		Project:  g.projectID,
		Location: g.location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return domain.AgentGatewayResponse{}, fmt.Errorf("failed to create gemini model: %w", err)
	}

	// 2. Build the full instruction from system prompt + conversation history
	fullInstruction := buildInstruction(systemPrompt, history)

	// 3. Build all tools (memory + financial)
	memoryTools, err := g.buildMemoryTools(ctx)
	if err != nil {
		return domain.AgentGatewayResponse{}, fmt.Errorf("failed to build memory tools: %w", err)
	}
	financialTools, err := g.buildFinancialTools(ctx)
	if err != nil {
		return domain.AgentGatewayResponse{}, fmt.Errorf("failed to build financial tools: %w", err)
	}
	tools := append(memoryTools, financialTools...)

	// 4. Create the ADK LLM agent
	agentInstance, err := llmagent.New(llmagent.Config{
		Name:        "finance_assistant",
		Description: "Assistente financeiro pessoal inteligente e empático",
		Model:       geminiModel,
		Instruction: fullInstruction,
		Tools:       tools,
	})
	if err != nil {
		return domain.AgentGatewayResponse{}, fmt.Errorf("failed to create ADK agent: %w", err)
	}

	// 5. InMemorySessionService: only for the ADK internal loop
	sessionSvc := session.InMemoryService()
	sessionID := "req-session"
	userID := "agent-user"

	_, err = sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return domain.AgentGatewayResponse{}, fmt.Errorf("failed to create session: %w", err)
	}

	// 6. Create and run the ADK runner
	agentRunner, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          agentInstance,
		SessionService: sessionSvc,
	})
	if err != nil {
		return domain.AgentGatewayResponse{}, fmt.Errorf("failed to create runner: %w", err)
	}

	userContent := genai.NewContentFromText(userMessage, "user")

	// 7. Collect output by ranging over the event stream
	var responseBuilder strings.Builder
	var toolsCalled []string
	var inputTokens, outputTokens int

	for event, err := range agentRunner.Run(ctx, userID, sessionID, userContent, agent.RunConfig{}) {
		if err != nil {
			return domain.AgentGatewayResponse{}, fmt.Errorf("agent run error: %w", err)
		}
		if event == nil {
			continue
		}

		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.FunctionCall != nil {
					toolsCalled = append(toolsCalled, part.FunctionCall.Name)
				}
			}
		}

		if event.LLMResponse.UsageMetadata != nil {
			inputTokens += int(event.LLMResponse.UsageMetadata.PromptTokenCount)
			outputTokens += int(event.LLMResponse.UsageMetadata.CandidatesTokenCount)
		}

		if !event.LLMResponse.Partial && event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" && !part.Thought {
					responseBuilder.WriteString(part.Text)
				}
			}
		}
	}

	return domain.AgentGatewayResponse{
		Content:      responseBuilder.String(),
		ToolsCalled:  toolsCalled,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

func buildInstruction(systemPrompt string, history []domain.AgentMessage) string {
	if len(history) == 0 {
		return systemPrompt
	}

	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\n--- HISTÓRICO DA CONVERSA ---\n")

	for _, msg := range history {
		role := "Usuário"
		if msg.Role == "assistant" {
			role = "Assistente"
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	return sb.String()
}

func (g *ADKAgentGateway) buildMemoryTools(ctx context.Context) ([]tool.Tool, error) {
	saveTool, err := functiontool.New(functiontool.Config{
		Name: "save_memory",
		Description: "Salva uma nova memória persistente sobre o usuário. " +
			"O campo memory_type deve ser um dos valores: goal, fact, constraint, insight, commitment, risk_profile, life_event.",
	}, func(_ tool.Context, args saveMemoryArgs) (saveMemoryResult, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "save_memory"), log.String("memory_type", args.MemoryType))
		userID := authentication.UserIDFromContext(ctx)
		if userID == "" {
			return saveMemoryResult{}, fmt.Errorf("usuário não autenticado")
		}

		memType := domain.AgentMemoryType(args.MemoryType)
		if !memType.IsValid() {
			return saveMemoryResult{}, fmt.Errorf("tipo de memória inválido: %s", args.MemoryType)
		}

		source := domain.MemorySourceExplicit
		if memType == domain.MemoryTypeInsight {
			source = domain.MemorySourceDerived
		}

		memory := domain.NewAgentMemory(userID, memType, args.Content, source)
		if args.Metadata != nil {
			memory.Metadata = args.Metadata
		}

		saved, err := g.memoryRepo.Save(ctx, memory)
		if err != nil {
			return saveMemoryResult{}, fmt.Errorf("erro ao salvar memória: %w", err)
		}

		return saveMemoryResult{
			ID:      saved.ID.String(),
			Message: fmt.Sprintf("Memória do tipo '%s' salva com sucesso.", args.MemoryType),
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create save_memory tool: %w", err)
	}

	updateTool, err := functiontool.New(functiontool.Config{
		Name:        "update_memory",
		Description: "Atualiza o conteúdo ou metadados de uma memória existente pelo ID.",
	}, func(_ tool.Context, args updateMemoryArgs) (updateMemoryResult, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "update_memory"), log.String("memory_id", args.ID))
		parsedID, err := uuid.Parse(args.ID)
		if err != nil {
			return updateMemoryResult{}, fmt.Errorf("id inválido: %s", args.ID)
		}

		existing, err := g.memoryRepo.FindByID(ctx, parsedID)
		if err != nil {
			return updateMemoryResult{}, fmt.Errorf("memória não encontrada: %s", args.ID)
		}

		existing.Content = args.Content
		if args.Metadata != nil {
			existing.Metadata = args.Metadata
		}

		_, err = g.memoryRepo.Update(ctx, existing)
		if err != nil {
			return updateMemoryResult{}, fmt.Errorf("erro ao atualizar memória: %w", err)
		}

		return updateMemoryResult{Message: "Memória atualizada com sucesso."}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create update_memory tool: %w", err)
	}

	deleteTool, err := functiontool.New(functiontool.Config{
		Name:        "delete_memory",
		Description: "Remove uma memória existente pelo ID.",
	}, func(_ tool.Context, args deleteMemoryArgs) (deleteMemoryResult, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "delete_memory"), log.String("memory_id", args.ID))
		parsedID, err := uuid.Parse(args.ID)
		if err != nil {
			return deleteMemoryResult{}, fmt.Errorf("id inválido: %s", args.ID)
		}

		if err := g.memoryRepo.Delete(ctx, parsedID); err != nil {
			return deleteMemoryResult{}, fmt.Errorf("erro ao deletar memória: %w", err)
		}

		return deleteMemoryResult{Message: "Memória removida com sucesso."}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create delete_memory tool: %w", err)
	}

	searchTool, err := functiontool.New(functiontool.Config{
		Name:        "search_memories",
		Description: "Busca memórias persistentes do usuário.",
	}, func(_ tool.Context, args searchMemoriesArgs) (searchMemoriesResult, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "search_memories"), log.String("query", args.Query))
		userID := authentication.UserIDFromContext(ctx)
		var memories []domain.AgentMemory
		var err error

		if args.Query == "" {
			memories, err = g.memoryRepo.FindByUserID(ctx, userID)
		} else {
			memories, err = g.memoryRepo.SearchByContent(ctx, userID, args.Query)
		}
		if err != nil {
			return searchMemoriesResult{}, fmt.Errorf("erro ao buscar memórias: %w", err)
		}

		items := make([]memoryItem, 0, len(memories))
		for _, m := range memories {
			items = append(items, memoryItem{
				ID:         m.ID.String(),
				MemoryType: string(m.Type),
				Content:    m.Content,
			})
		}

		return searchMemoriesResult{Memories: items, Count: len(items)}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create search_memories tool: %w", err)
	}

	return []tool.Tool{saveTool, updateTool, deleteTool, searchTool}, nil
}

// --- Financial tool arg/result DTOs ---

type financialPeriodArgs struct {
	// Month is 1-12. If omitted, defaults to current month.
	Month int `json:"month,omitempty"`
	// Year e.g. 2025. If omitted, defaults to current year.
	Year int `json:"year,omitempty"`
}

type getMovementsArgs struct {
	Month int `json:"month,omitempty"`
	Year  int `json:"year,omitempty"`
	// Limit max movements to return (default 50, max 100).
	Limit int `json:"limit,omitempty"`
}

func (g *ADKAgentGateway) buildFinancialTools(ctx context.Context) ([]tool.Tool, error) {
	overviewTool, err := functiontool.New(functiontool.Config{
		Name: "get_financial_overview",
		Description: "Retorna o resumo financeiro do período (mês/ano): receitas totais, despesas totais, saldo líquido e saldo atual de cada carteira. " +
			"Use para responder perguntas como 'quanto recebi/gastei este mês?' ou 'qual meu saldo?'. " +
			"Os parâmetros month e year são opcionais — se omitidos, usa o mês e ano atuais.",
	}, func(_ tool.Context, args financialPeriodArgs) (domain.AgentFinancialOverview, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "get_financial_overview"), log.Int("month", args.Month), log.Int("year", args.Year))
		return g.financialRepo.GetFinancialOverview(ctx, args.Month, args.Year)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create get_financial_overview tool: %w", err)
	}

	breakdownTool, err := functiontool.New(functiontool.Config{
		Name: "get_spending_breakdown",
		Description: "Retorna gastos e receitas detalhados por categoria para o período, com valor e percentual. " +
			"Use para responder 'onde estou gastando mais?' ou 'qual minha maior despesa?'.",
	}, func(_ tool.Context, args financialPeriodArgs) (domain.AgentSpendingBreakdown, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "get_spending_breakdown"), log.Int("month", args.Month), log.Int("year", args.Year))
		return g.financialRepo.GetSpendingBreakdown(ctx, args.Month, args.Year)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create get_spending_breakdown tool: %w", err)
	}

	creditCardsTool, err := functiontool.New(functiontool.Config{
		Name: "get_credit_cards",
		Description: "Retorna todos os cartões de crédito do usuário com limite total, limite disponível, " +
			"valor e data de vencimento da próxima fatura, e total de faturas em aberto. " +
			"Use para responder 'qual minha fatura?' ou 'quanto posso gastar no cartão?'.",
	}, func(_ tool.Context, _ struct{}) (domain.AgentCreditCardsSummary, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "get_credit_cards"))
		return g.financialRepo.GetCreditCardsSummary(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create get_credit_cards tool: %w", err)
	}

	movementsTool, err := functiontool.New(functiontool.Config{
		Name: "get_movements",
		Description: "Lista as transações (movimentações) do período com data, descrição, valor, categoria e carteira. " +
			"Use para detalhar transações específicas ou quando o usuário quiser ver o extrato. " +
			"O parâmetro limit controla quantos registros retornar (padrão 50, máximo 100).",
	}, func(_ tool.Context, args getMovementsArgs) (domain.AgentMovementsList, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "get_movements"), log.Int("month", args.Month), log.Int("year", args.Year), log.Int("limit", args.Limit))
		return g.financialRepo.GetMovements(ctx, args.Month, args.Year, args.Limit)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create get_movements tool: %w", err)
	}

	recurringTool, err := functiontool.New(functiontool.Config{
		Name: "get_recurring_expenses",
		Description: "Lista todos os compromissos financeiros recorrentes ativos (assinaturas, contas fixas, salários) " +
			"com valor, categoria e dia do mês, além do impacto total mensal. " +
			"Use para responder 'quais são minhas despesas fixas?' ou 'quanto gasto em assinaturas?'.",
	}, func(_ tool.Context, _ struct{}) (domain.AgentRecurringSummary, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "get_recurring_expenses"))
		return g.financialRepo.GetRecurringSummary(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create get_recurring_expenses tool: %w", err)
	}

	budgetTool, err := functiontool.New(functiontool.Config{
		Name: "get_budget_status",
		Description: "Compara o orçamento planejado (estimativas) com o realizado por categoria para o período. " +
			"Retorna variação absoluta e percentual. " +
			"Use para responder 'estou dentro do orçamento?' ou 'ultrapassei o limite de alguma categoria?'.",
	}, func(_ tool.Context, args financialPeriodArgs) (domain.AgentBudgetStatus, error) {
		log.InfoContext(ctx, "agent tool called", log.String("tool", "get_budget_status"), log.Int("month", args.Month), log.Int("year", args.Year))
		return g.financialRepo.GetBudgetStatus(ctx, args.Month, args.Year)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create get_budget_status tool: %w", err)
	}

	return []tool.Tool{overviewTool, breakdownTool, creditCardsTool, movementsTool, recurringTool, budgetTool}, nil
}
