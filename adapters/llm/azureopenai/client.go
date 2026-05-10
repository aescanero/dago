package azureopenai

import (
	"context"
	"errors"
	"fmt"

	goai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/ports"
)

// Config holds configuration for the Azure OpenAI adapter.
type Config struct {
	APIKey     string // AZURE_OPENAI_API_KEY    (required)
	Endpoint   string // AZURE_OPENAI_ENDPOINT    (required)
	Deployment string // AZURE_OPENAI_DEPLOYMENT  (required)
	APIVersion string // AZURE_OPENAI_API_VERSION (required)
}

// AzureOpenAIClient implements ports.LLMClient using the Azure OpenAI API.
type AzureOpenAIClient struct {
	client     *goai.Client
	deployment string
}

var _ ports.LLMClient = &AzureOpenAIClient{}

// NewAzureOpenAIClient creates an AzureOpenAIClient from cfg.
// Returns an error if any required field is empty.
func NewAzureOpenAIClient(cfg Config) (*AzureOpenAIClient, error) {
	switch {
	case cfg.APIKey == "":
		return nil, errors.New("azureopenai: APIKey is required")
	case cfg.Endpoint == "":
		return nil, errors.New("azureopenai: Endpoint is required")
	case cfg.Deployment == "":
		return nil, errors.New("azureopenai: Deployment is required")
	case cfg.APIVersion == "":
		return nil, errors.New("azureopenai: APIVersion is required")
	}
	ocfg := goai.DefaultAzureConfig(cfg.APIKey, cfg.Endpoint)
	ocfg.APIVersion = cfg.APIVersion
	return &AzureOpenAIClient{
		client:     goai.NewClientWithConfig(ocfg),
		deployment: cfg.Deployment,
	}, nil
}

// Complete sends req to the Azure OpenAI chat completions API and returns the response.
// The Deployment always overrides req.Model — Azure routes by deployment name, not model name.
func (c *AzureOpenAIClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
	chatReq := goai.ChatCompletionRequest{
		Model:     c.deployment,
		Messages:  toOpenAIMessages(req.Messages),
		MaxTokens: req.MaxTokens,
	}
	if len(req.Tools) > 0 {
		chatReq.Tools = toOpenAITools(req.Tools)
	}
	resp, err := c.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return ports.LLMResponse{}, fmt.Errorf("azureopenai complete: %w", mapError(err))
	}
	return fromOpenAIResponse(resp), nil
}
