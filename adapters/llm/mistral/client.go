package mistral

import (
	"context"
	"errors"
	"fmt"

	goai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/ports"
)

const defaultModel = "mistral-large-latest"
const defaultBaseURL = "https://api.mistral.ai/v1"

// Config holds configuration for the Mistral adapter.
type Config struct {
	APIKey  string // MISTRAL_API_KEY  (required)
	BaseURL string // MISTRAL_BASE_URL (optional; default https://api.mistral.ai/v1)
	Model   string // MISTRAL_MODEL    (optional; default mistral-large-latest)
}

// MistralClient implements ports.LLMClient using Mistral's OpenAI-compatible API.
type MistralClient struct {
	client *goai.Client
	model  string
}

var _ ports.LLMClient = &MistralClient{}

// NewMistralClient creates a MistralClient from cfg.
// Returns an error if APIKey is empty.
func NewMistralClient(cfg Config) (*MistralClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("mistral: APIKey is required")
	}
	ocfg := goai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		ocfg.BaseURL = cfg.BaseURL
	} else {
		ocfg.BaseURL = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	return &MistralClient{
		client: goai.NewClientWithConfig(ocfg),
		model:  model,
	}, nil
}

// Complete sends req to Mistral's chat completions API and returns the response.
func (c *MistralClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	chatReq := goai.ChatCompletionRequest{
		Model:     model,
		Messages:  toOpenAIMessages(req.Messages),
		MaxTokens: req.MaxTokens,
	}
	if len(req.Tools) > 0 {
		chatReq.Tools = toOpenAITools(req.Tools)
	}
	resp, err := c.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return ports.LLMResponse{}, fmt.Errorf("mistral complete: %w", mapError(err))
	}
	return fromOpenAIResponse(resp), nil
}
