package anthropic

import (
	"context"
	"fmt"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/ports"
	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"go.uber.org/zap"
)

// Client implements the LLMClient interface for Anthropic Claude
type Client struct {
	client anthropicsdk.Client
	logger *zap.Logger
}

// NewClient creates a new Anthropic client
func NewClient(apiKey string, logger *zap.Logger) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := anthropicsdk.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

// Complete performs a standard text completion (ports.LLMClient interface)
func (c *Client) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// CompleteWithTools performs a completion with tool calling support (ports.LLMClient interface)
func (c *Client) CompleteWithTools(ctx context.Context, req ports.CompletionRequest, tools []ports.Tool) (*ports.CompletionResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// CompleteStructured performs a completion with guaranteed JSON schema conformance (ports.LLMClient interface)
func (c *Client) CompleteStructured(ctx context.Context, req ports.CompletionRequest, schema ports.JSONSchema) (*ports.StructuredResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateCompletion generates a completion using domain.LLMRequest (compatibility method)
func (c *Client) GenerateCompletion(ctx context.Context, req interface{}) (interface{}, error) {
	// Type assert the request
	llmReq, ok := req.(*domain.LLMRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type")
	}

	c.logger.Debug("generating completion",
		zap.String("model", llmReq.Model),
		zap.Int("message_count", len(llmReq.Messages)))

	// Convert messages to Anthropic format
	messages := make([]anthropicsdk.MessageParam, 0, len(llmReq.Messages))
	for _, msg := range llmReq.Messages {
		// Create text block for message content
		if msg.Role == "user" {
			messages = append(messages, anthropicsdk.NewUserMessage(
				anthropicsdk.NewTextBlock(msg.Content),
			))
		} else if msg.Role == "assistant" {
			messages = append(messages, anthropicsdk.NewAssistantMessage(
				anthropicsdk.NewTextBlock(msg.Content),
			))
		}
	}

	// Build request parameters
	maxTokens := int64(llmReq.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 1024
	}

	params := anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.Model(llmReq.Model),
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	if llmReq.System != "" {
		params.System = []anthropicsdk.TextBlockParam{
			{Text: llmReq.System},
		}
	}

	if llmReq.Temperature > 0 {
		params.Temperature = param.NewOpt(llmReq.Temperature)
	}

	// Call API
	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		c.logger.Error("API call failed", zap.Error(err))
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	// Convert response
	llmResp := &domain.LLMResponse{
		Content: extractContent(resp),
		Model:   string(resp.Model),
		Usage: domain.Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
		},
	}

	c.logger.Debug("completion generated",
		zap.Int("input_tokens", llmResp.Usage.InputTokens),
		zap.Int("output_tokens", llmResp.Usage.OutputTokens))

	return llmResp, nil
}

// extractContent extracts text content from response
func extractContent(resp *anthropicsdk.Message) string {
	if len(resp.Content) == 0 {
		return ""
	}

	// For MVP, just return first text block
	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text
		}
	}

	return ""
}
