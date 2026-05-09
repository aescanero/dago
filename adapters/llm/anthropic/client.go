package anthropic

import (
	"context"
	"errors"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"

	"github.com/aescanero/dago/libs/ports"
)

const defaultModel = "claude-sonnet-4-6"

// Config holds configuration for the Anthropic client.
type Config struct {
	APIKey     string // ANTHROPIC_API_KEY (required)
	BaseURL    string // ANTHROPIC_BASE_URL (optional)
	MaxRetries int    // ANTHROPIC_MAX_RETRIES (default 2)
}

// AnthropicClient implements ports.LLMClient using the official Anthropic SDK.
type AnthropicClient struct { //nolint:revive
	client       *anthropicsdk.Client
	defaultModel string
}

var _ ports.LLMClient = &AnthropicClient{}

// NewAnthropicClient builds an AnthropicClient. Returns error if APIKey is empty.
func NewAnthropicClient(cfg Config) (*AnthropicClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("anthropic: APIKey is required")
	}
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithMaxRetries(cfg.MaxRetries),
		option.WithoutEnvironmentDefaults(),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	c := anthropicsdk.NewClient(opts...)
	return &AnthropicClient{client: &c, defaultModel: defaultModel}, nil
}

// Complete sends the request to the Anthropic API and returns the domain response.
func (c *AnthropicClient) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	p := anthropicsdk.MessageNewParams{
		Model:     model,
		MaxTokens: int64(req.MaxTokens),
		Messages:  toAnthropicMessages(req.Messages),
	}
	if req.System != "" {
		p.System = []anthropicsdk.TextBlockParam{{Text: req.System}}
	}
	if len(req.Tools) > 0 {
		p.Tools = toAnthropicTools(req.Tools)
	}
	if req.Temperature != 0 {
		p.Temperature = param.NewOpt(req.Temperature)
	}
	msg, err := c.client.Messages.New(ctx, p)
	if err != nil {
		return ports.LLMResponse{}, mapAnthropicError("anthropic complete", err)
	}
	return fromAnthropicResponse(msg), nil
}
