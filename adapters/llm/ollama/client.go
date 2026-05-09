package ollama

import (
	"context"

	openai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/ports"
)

const (
	defaultBaseURL     = "http://localhost:11434"
	defaultOllamaModel = "mixtral"
)

// Config holds configuration for the Ollama client.
type Config struct {
	BaseURL      string // OLLAMA_BASE_URL (default: http://localhost:11434)
	DefaultModel string // OLLAMA_DEFAULT_MODEL (default: mixtral)
}

// OllamaClient implements ports.LLMClient using Ollama's OpenAI-compatible API.
type OllamaClient struct { //nolint:revive
	client       *openai.Client
	defaultModel string
}

var _ ports.LLMClient = &OllamaClient{}

// NewOllamaClient builds an OllamaClient.
// If BaseURL is empty, uses "http://localhost:11434".
// If DefaultModel is empty, uses "mixtral".
func NewOllamaClient(cfg Config) *OllamaClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := cfg.DefaultModel
	if model == "" {
		model = defaultOllamaModel
	}
	oaiCfg := openai.DefaultConfig("")
	oaiCfg.BaseURL = baseURL + "/v1"
	return &OllamaClient{
		client:       openai.NewClientWithConfig(oaiCfg),
		defaultModel: model,
	}
}

// Complete sends the request to Ollama and returns the domain response.
func (c *OllamaClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	chatReq := openai.ChatCompletionRequest{
		Model:     model,
		Messages:  toOpenAIMessages(req.Messages),
		MaxTokens: req.MaxTokens,
	}
	if len(req.Tools) > 0 {
		chatReq.Tools = toOpenAITools(req.Tools)
	}
	if req.Temperature != 0 {
		chatReq.Temperature = float32(req.Temperature)
	}
	resp, err := c.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return ports.LLMResponse{}, mapOllamaError("ollama complete", err)
	}
	return fromOpenAIResponse(resp), nil
}
