package llm

import (
	"fmt"

	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/aescanero/dago/pkg/adapters/llm/anthropic"
	"go.uber.org/zap"
)

// Config holds LLM client configuration
type Config struct {
	Provider string
	APIKey   string
	Logger   *zap.Logger
}

// NewClient creates a new LLM client based on provider
func NewClient(cfg *Config) (ports.LLMClient, error) {
	switch cfg.Provider {
	case "anthropic":
		return anthropic.NewClient(cfg.APIKey, cfg.Logger)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
