//go:build acceptance

package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"personal-finance/internal/domain"
	"personal-finance/internal/plataform/authentication"
)

const FixturePrefix = "pf-fixture:"

func FixturesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "fixtures", "import")
}

// FixtureVisionGateway devolve a leitura gabarito do documento, identificado pelo /Subject do PDF.
type FixtureVisionGateway struct {
	mu       sync.RWMutex
	readings map[string]domain.StatementExtractResult
}

func NewFixtureVisionGateway() *FixtureVisionGateway {
	return &FixtureVisionGateway{readings: map[string]domain.StatementExtractResult{}}
}

func (g *FixtureVisionGateway) Register(id string, reading domain.StatementExtractResult) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.readings[id] = reading
}

func (g *FixtureVisionGateway) LoadFixtures() error {
	entries, err := os.ReadDir(FixturesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(FixturesDir(), e.Name(), "extraction.json")
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		var reading domain.StatementExtractResult
		if err := json.Unmarshal(raw, &reading); err != nil {
			return fmt.Errorf("fixture %s: %w", path, err)
		}
		g.Register(e.Name(), reading)
	}
	return nil
}

func (g *FixtureVisionGateway) ExtractMovements(_ context.Context, fileBytes []byte, _, _ string) (domain.StatementExtractResult, error) {
	id, err := FixtureID(fileBytes)
	if err != nil {
		return domain.StatementExtractResult{}, err
	}
	g.mu.RLock()
	reading, ok := g.readings[id]
	g.mu.RUnlock()
	if !ok {
		return domain.StatementExtractResult{}, fmt.Errorf("fixture vision gateway: no reading registered for %q", id)
	}
	return cloneReading(reading)
}

func FixtureID(pdf []byte) (string, error) {
	ctx, err := api.ReadContext(bytes.NewReader(pdf), model.NewDefaultConfiguration())
	if err != nil {
		return "", fmt.Errorf("fixture vision gateway: reading pdf: %w", err)
	}
	if err := api.ValidateContext(ctx); err != nil {
		return "", fmt.Errorf("fixture vision gateway: validating pdf: %w", err)
	}
	subject := ctx.XRefTable.Subject
	if !strings.HasPrefix(subject, FixturePrefix) {
		return "", fmt.Errorf("fixture vision gateway: pdf has no %q subject (got %q)", FixturePrefix, subject)
	}
	return strings.TrimPrefix(subject, FixturePrefix), nil
}

// cloneReading: o usecase altera o resultado, e a leitura é reusada entre cenários.
func cloneReading(r domain.StatementExtractResult) (domain.StatementExtractResult, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return domain.StatementExtractResult{}, err
	}
	var out domain.StatementExtractResult
	err = json.Unmarshal(raw, &out)
	return out, err
}

// FixtureClassificationGateway não sugere nada, salvo o que o cenário declarar (por usuário).
type FixtureClassificationGateway struct {
	mu          sync.RWMutex
	suggestions map[string]map[string]domain.CategorySuggestion // userID → description → sugestão
}

func NewFixtureClassificationGateway() *FixtureClassificationGateway {
	return &FixtureClassificationGateway{suggestions: map[string]map[string]domain.CategorySuggestion{}}
}

func (g *FixtureClassificationGateway) Suggest(userID, description string, s domain.CategorySuggestion) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.suggestions[userID] == nil {
		g.suggestions[userID] = map[string]domain.CategorySuggestion{}
	}
	s.Description = description
	s.Source = "ai"
	g.suggestions[userID][description] = s
}

func (g *FixtureClassificationGateway) ClassifyMovements(ctx context.Context, movements []domain.ExtractedMovement, _ []domain.Category) ([]domain.CategorySuggestion, error) {
	userID := authentication.UserIDFromContext(ctx)
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]domain.CategorySuggestion, len(movements))
	for i, m := range movements {
		if s, ok := g.suggestions[userID][m.Description]; ok {
			out[i] = s
			continue
		}
		out[i] = domain.CategorySuggestion{Description: m.Description}
	}
	return out, nil
}
