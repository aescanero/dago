package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
	"github.com/aescanero/dago/services/executor/internal/mapping"
)

// LLMCallConfig holds llm_call node configuration.
type LLMCallConfig struct {
	Model         string            `json:"model"`
	SystemPrompt  string            `json:"system_prompt"`
	Temperature   float64           `json:"temperature"`
	MaxTokens     int               `json:"max_tokens"`
	InputMapping  map[string]string `json:"input_mapping"`
	OutputMapping map[string]string `json:"output_mapping"`
}

// LLMCallHandler handles the llm_call node pattern.
type LLMCallHandler struct {
	llmClient ports.LLMClient
	publisher ports.EventPublisher
}

// NewLLMCallHandler creates an LLMCallHandler.
func NewLLMCallHandler(client ports.LLMClient, publisher ports.EventPublisher) *LLMCallHandler {
	return &LLMCallHandler{llmClient: client, publisher: publisher}
}

// Handle processes a node.execute.requested event for the llm_call pattern.
func (h *LLMCallHandler) Handle(ctx context.Context, data NodeExecuteRequestedData) error {
	var cfg LLMCallConfig
	if err := json.Unmarshal(data.Config, &cfg); err != nil {
		return h.publishFailure(ctx, data, "execution_error", false, err)
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 2048
	}

	msgs, err := mapping.ApplyInputMapping(cfg.InputMapping, data.Variables, data.Messages)
	if err != nil {
		return h.publishFailure(ctx, data, "execution_error", false, err)
	}

	req := ports.LLMRequest{
		Model:       cfg.Model,
		System:      cfg.SystemPrompt,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Messages:    msgs,
	}

	start := time.Now()
	resp, err := h.llmClient.Complete(ctx, req)
	if err != nil {
		code, retryable := classifyLLMError(err)
		return h.publishFailure(ctx, data, code, retryable, err)
	}

	update, err := mapping.ApplyOutputMapping(cfg.OutputMapping, resp)
	if err != nil {
		return h.publishFailure(ctx, data, "execution_error", false, err)
	}

	return h.publishSuccess(ctx, data, resp, update, time.Since(start).Milliseconds())
}

func classifyLLMError(err error) (code string, retryable bool) {
	switch {
	case errors.Is(err, domain.ErrRateLimited):
		return "rate_limited", true
	case errors.Is(err, domain.ErrProviderUnavailable):
		return "provider_unavailable", true
	case errors.Is(err, domain.ErrUnauthorized):
		return "unauthorized", false
	default:
		return "execution_error", false
	}
}

func (h *LLMCallHandler) publishSuccess(
	ctx context.Context,
	data NodeExecuteRequestedData,
	resp ports.LLMResponse,
	update map[string]any,
	durationMs int64,
) error {
	outputRaw, _ := json.Marshal(resp)
	updateRaw, _ := json.Marshal(update)
	payload := NodeExecutedData{
		ExecutionID:     data.ExecutionID,
		GraphID:         data.GraphID,
		NodeID:          data.NodeID,
		NodeKey:         data.NodeKey,
		Output:          outputRaw,
		VariablesUpdate: updateRaw,
		DurationMs:      durationMs,
	}
	return h.publishEvent(ctx, domain.EventTypeNodeExecuted, domain.StreamNodeExecuted, payload)
}

func (h *LLMCallHandler) publishFailure(
	ctx context.Context,
	data NodeExecuteRequestedData,
	code string,
	retryable bool,
	cause error,
) error {
	payload := NodeExecuteFailedData{
		ExecutionID: data.ExecutionID,
		GraphID:     data.GraphID,
		NodeID:      data.NodeID,
		NodeKey:     data.NodeKey,
		Error:       cause.Error(),
		ErrorCode:   code,
		Retryable:   retryable,
	}
	_ = h.publishEvent(ctx, domain.EventTypeNodeExecuteFailed, domain.StreamNodeExecuteFailed, payload)
	return cause
}

func (h *LLMCallHandler) publishEvent(ctx context.Context, evtType, stream string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("handler: marshal payload: %w", err)
	}
	evt := domain.Event{
		ID:              uuid.NewString(),
		Type:            evtType,
		Source:          "executor",
		SpecVersion:     "1.0",
		Time:            time.Now(),
		DataContentType: "application/json",
		Data:            json.RawMessage(raw),
	}
	if err := h.publisher.Publish(ctx, evt, ports.PublishOptions{Stream: stream}); err != nil {
		return fmt.Errorf("handler: publish %s: %w", stream, err)
	}
	return nil
}
