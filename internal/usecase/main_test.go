package usecase

import (
	"os"
	"testing"

	"personal-finance/pkg/log"
)

// TestMain inicializa o logger global: os usecases logam pelos helpers
// context-aware (log.InfoContext/WarnContext), que — ao contrário de log.Info —
// não protegem contra logger nulo e entrariam em panic sem isto.
func TestMain(m *testing.M) {
	log.Initialize()
	os.Exit(m.Run())
}
