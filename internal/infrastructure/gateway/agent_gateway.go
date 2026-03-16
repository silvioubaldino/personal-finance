package gateway

import (
	"context"
	"fmt"

	"personal-finance/internal/domain"
)

// PlaceholderAgentGateway is a temporary implementation of the AgentGateway interface.
// It will be replaced by the real ADK + Vertex AI gateway implementation in a future task.
//
// This placeholder allows the entire agent feature (memories, conversations, audit)
// to be developed and tested without an actual LLM connection.
type PlaceholderAgentGateway struct {
	// Future fields:
	// adkRunner    *runner.Runner
	// geminiModel  genai.Model
	// memoryTools  []tool.Tool
	// financialTools []tool.Tool  // TODO: implement in next task
}

func NewPlaceholderAgentGateway() *PlaceholderAgentGateway {
	return &PlaceholderAgentGateway{}
}

// Chat sends a message to the LLM and returns a response.
//
// In the real implementation, this will:
// 1. Build system prompt from context
// 2. Register ADK tools (save_memory, delete_memory, search_memories)
// 3. Register financial analysis tools (get_overview, get_spending_by_category, etc.) — TODO: next task
// 4. Run the ADK agentic loop with Vertex AI (Gemini Flash, southamerica-east1)
// 5. Return the final response with metadata
//
// ADK Tools that will be registered:
//
// Memory Tools (implemented in this task):
//   - save_memory(memory_type, content, metadata?) → saves user facts, goals, constraints, etc.
//   - delete_memory(memory_id)                     → removes outdated memories
//   - search_memories(query?)                      → searches existing memories by content
//   - update_memory(memory_id, content, metadata?) → updates an existing memory
//
// Financial Analysis Tools (TODO: next task — placeholder only):
//   - get_overview(month, year)                              → wallet balances, totals
//   - get_spending_by_category(month, year, category_name?)  → grouped spending
//   - analyze_trend(category_name, months_back)              → multi-month series
//   - simulate_impact(expense_value, installment_count)      → budget impact projection
func (g *PlaceholderAgentGateway) Chat(
	_ context.Context,
	systemPrompt string,
	userMessage string,
	history []domain.AgentMessage,
) (domain.AgentGatewayResponse, error) {

	// Placeholder: echo back a helpful message indicating the gateway is not yet connected.
	response := fmt.Sprintf(
		"[Placeholder] O gateway ADK ainda não está conectado ao Vertex AI. "+
			"Recebi sua mensagem: \"%s\". "+
			"Histórico contém %d mensagens anteriores. "+
			"System prompt tem %d caracteres.",
		userMessage, len(history), len(systemPrompt),
	)

	return domain.AgentGatewayResponse{
		Content:      response,
		ToolsCalled:  []string{},
		InputTokens:  0,
		OutputTokens: 0,
	}, nil
}
