package openai

import (
	"context"
	"errors"
	"fmt"

	goai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/ports"
)

const defaultModel = "gpt-4o"
const defaultBaseURL = "https://api.openai.com/v1"

// Config holds configuration for the OpenAI adapter.
type Config struct {
	APIKey  string // OPENAI_API_KEY (required)
	BaseURL string // OPENAI_BASE_URL (optional; default https://api.openai.com/v1)
	Model   string // OPENAI_MODEL   (optional; default gpt-4o)
}

// OpenAIClient implements ports.LLMClient using the OpenAI API.
type OpenAIClient struct {
	client *goai.Client
	model  string
}

var _ ports.LLMClient = &OpenAIClient{}

// NewOpenAIClient creates an OpenAIClient from cfg.
// Returns an error if APIKey is empty.
func NewOpenAIClient(cfg Config) (*OpenAIClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("openai: APIKey is required")
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
	return &OpenAIClient{
		client: goai.NewClientWithConfig(ocfg),
		model:  model,
	}, nil
}

// Complete sends req to the OpenAI chat completions API and returns the response.
func (c *OpenAIClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
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
		return ports.LLMResponse{}, fmt.Errorf("openai complete: %w", mapError(err))
	}
	return fromOpenAIResponse(resp), nil
}
