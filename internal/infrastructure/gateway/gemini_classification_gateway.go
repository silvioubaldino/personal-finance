package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"personal-finance/internal/domain"

	"github.com/google/uuid"
	"google.golang.org/genai"
)

type GeminiClassificationGateway struct {
	projectID string
	location  string
	modelName string
}

func NewGeminiClassificationGateway() *GeminiClassificationGateway {
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = defaultLocation
	}

	modelName := os.Getenv("VERTEX_MODEL")
	if modelName == "" {
		modelName = defaultModel
	}

	return &GeminiClassificationGateway{
		projectID: os.Getenv("GOOGLE_PROJECT_ID"),
		location:  location,
		modelName: modelName,
	}
}

type classificationRequest struct {
	Index       int     `json:"index"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

type classificationResponse struct {
	Index         int        `json:"index"`
	CategoryID    *string    `json:"category_id"`
	SubCategoryID *string    `json:"subcategory_id"`
	Confidence    float64    `json:"confidence"`
}

func (g *GeminiClassificationGateway) ClassifyMovements(
	ctx context.Context,
	movements []domain.ExtractedMovement,
	categories []domain.Category,
) ([]domain.CategorySuggestion, error) {
	if len(movements) == 0 {
		return nil, nil
	}

	categoriesJSON, err := buildCategoriesJSON(categories)
	if err != nil {
		return nil, fmt.Errorf("failed to build categories JSON: %w", err)
	}

	requests := make([]classificationRequest, len(movements))
	for i, m := range movements {
		requests[i] = classificationRequest{
			Index:       i,
			Description: m.Description,
			Amount:      m.Amount,
		}
	}

	movementsJSON, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal movements for classification: %w", err)
	}

	prompt := buildClassificationPrompt(categoriesJSON, string(movementsJSON))

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  g.projectID,
		Location: g.location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	resp, err := client.Models.GenerateContent(ctx, g.modelName, []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{genai.NewPartFromText(prompt)}, "user"),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini classification call failed: %w", err)
	}

	recordTokenUsage(ctx, "statement_classify", g.modelName, resp)

	var responseText string
	if resp != nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				responseText += part.Text
			}
		}
	}

	if responseText == "" {
		return buildFallbackSuggestions(movements), nil
	}

	responseText = cleanJSONResponse(responseText)

	var rawResults []classificationResponse
	if err := json.Unmarshal([]byte(responseText), &rawResults); err != nil {
		return buildFallbackSuggestions(movements), nil
	}

	return mapClassificationResults(movements, rawResults), nil
}

func buildCategoriesJSON(categories []domain.Category) (string, error) {
	type subCatJSON struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	type catJSON struct {
		ID            string       `json:"id"`
		Name          string       `json:"name"`
		SubCategories []subCatJSON `json:"subcategories"`
	}

	cats := make([]catJSON, 0, len(categories))
	for _, c := range categories {
		if c.ID == nil {
			continue
		}
		subs := make([]subCatJSON, 0, len(c.SubCategories))
		for _, sc := range c.SubCategories {
			if sc.ID == nil {
				continue
			}
			subs = append(subs, subCatJSON{ID: sc.ID.String(), Name: sc.Description})
		}
		cats = append(cats, catJSON{
			ID:            c.ID.String(),
			Name:          c.Description,
			SubCategories: subs,
		})
	}

	b, err := json.Marshal(cats)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func buildClassificationPrompt(categoriesJSON, movementsJSON string) string {
	return fmt.Sprintf(`Você é um categorizador de transações financeiras.

Categorias disponíveis (use APENAS os IDs exatos abaixo, não invente IDs):
%s

Classifique cada transação abaixo. Retorne um array JSON com um objeto por transação, na mesma ordem, com os campos:
- "index": o índice original da transação (começa em 0)
- "category_id": UUID da categoria ou null se não souber
- "subcategory_id": UUID da subcategoria ou null (só inclua se tiver certeza)
- "confidence": número de 0.0 a 1.0 indicando sua confiança

Transações a classificar:
%s

Retorne APENAS o array JSON, sem texto adicional, sem markdown.
Exemplo: [{"index":0,"category_id":"uuid","subcategory_id":null,"confidence":0.9}]`, categoriesJSON, movementsJSON)
}

func mapClassificationResults(
	movements []domain.ExtractedMovement,
	rawResults []classificationResponse,
) []domain.CategorySuggestion {
	resultByIndex := make(map[int]classificationResponse, len(rawResults))
	for _, r := range rawResults {
		resultByIndex[r.Index] = r
	}

	suggestions := make([]domain.CategorySuggestion, len(movements))
	for i, m := range movements {
		suggestion := domain.CategorySuggestion{
			Description: m.Description,
			Source:      "ai",
		}

		if r, ok := resultByIndex[i]; ok {
			suggestion.Confidence = r.Confidence
			if r.CategoryID != nil && *r.CategoryID != "" {
				parsed := parseUUID(*r.CategoryID)
				suggestion.CategoryID = parsed
			}
			if r.SubCategoryID != nil && *r.SubCategoryID != "" {
				parsed := parseUUID(*r.SubCategoryID)
				suggestion.SubCategoryID = parsed
			}
		}

		suggestions[i] = suggestion
	}

	return suggestions
}

func buildFallbackSuggestions(movements []domain.ExtractedMovement) []domain.CategorySuggestion {
	suggestions := make([]domain.CategorySuggestion, len(movements))
	for i, m := range movements {
		suggestions[i] = domain.CategorySuggestion{
			Description: m.Description,
			Source:      "ai",
			Confidence:  0,
		}
	}
	return suggestions
}

func parseUUID(s string) *uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}
