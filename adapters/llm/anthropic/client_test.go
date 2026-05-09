package anthropic_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aescanero/dago/adapters/llm/anthropic"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

func textResponse(text string) string {
	return `{
		"id":"msg_01","type":"message","role":"assistant",
		"content":[{"type":"text","text":"` + text + `"}],
		"model":"claude-sonnet-4-6-20251001",
		"stop_reason":"end_turn",
		"usage":{"input_tokens":10,"output_tokens":5}
	}`
}

func TestCompleteText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(textResponse("hello world")))
	}))
	defer ts.Close()

	client, err := anthropic.NewAnthropicClient(anthropic.Config{APIKey: "test-key", BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}

	resp, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "claude-sonnet-4-6",
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "hello world" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello world")
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "end_turn")
	}
	if resp.InputTokens <= 0 {
		t.Errorf("InputTokens = %d, want > 0", resp.InputTokens)
	}
	if resp.OutputTokens <= 0 {
		t.Errorf("OutputTokens = %d, want > 0", resp.OutputTokens)
	}
}

func TestCompleteWithTools(t *testing.T) {
	toolResp := `{
		"id":"msg_02","type":"message","role":"assistant",
		"content":[{"type":"tool_use","id":"tu_01","name":"get_weather","input":{"location":"NYC"}}],
		"model":"claude-sonnet-4-6-20251001",
		"stop_reason":"tool_use",
		"usage":{"input_tokens":15,"output_tokens":8}
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toolResp))
	}))
	defer ts.Close()

	client, _ := anthropic.NewAnthropicClient(anthropic.Config{APIKey: "test-key", BaseURL: ts.URL})

	schema := json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`)
	resp, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []ports.Message{{Role: "user", Content: "What's the weather?"}},
		Tools: []ports.ToolDefinition{
			{Name: "get_weather", Description: "Get weather", InputSchema: schema},
		},
		MaxTokens: 200,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "tool_use")
	}
	if len(resp.ToolUses) != 1 {
		t.Fatalf("len(ToolUses) = %d, want 1", len(resp.ToolUses))
	}
	if resp.ToolUses[0].Name != "get_weather" {
		t.Errorf("ToolUses[0].Name = %q, want %q", resp.ToolUses[0].Name, "get_weather")
	}
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
}

func TestCompleteToolResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(textResponse("The weather is sunny.")))
	}))
	defer ts.Close()

	client, _ := anthropic.NewAnthropicClient(anthropic.Config{APIKey: "test-key", BaseURL: ts.URL})

	resp, err := client.Complete(context.Background(), ports.LLMRequest{
		Model: "claude-sonnet-4-6",
		Messages: []ports.Message{
			{Role: "user", Content: "What's the weather?"},
			{Role: "assistant", Content: ""},
			{Role: "tool_result", Content: "sunny", ToolUseID: "tu_01"},
		},
		MaxTokens: 200,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "end_turn")
	}
	if resp.Content == "" {
		t.Error("Content is empty, want non-empty")
	}
}

func TestCompleteRateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"rate limited"}}`))
	}))
	defer ts.Close()

	client, _ := anthropic.NewAnthropicClient(anthropic.Config{APIKey: "test-key", BaseURL: ts.URL, MaxRetries: 0})

	_, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "claude-sonnet-4-6",
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got: %v", err)
	}
}

func TestCompleteServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"type":"overloaded_error","message":"server error"}}`))
	}))
	defer ts.Close()

	client, _ := anthropic.NewAnthropicClient(anthropic.Config{APIKey: "test-key", BaseURL: ts.URL, MaxRetries: 0})

	_, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "claude-sonnet-4-6",
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got: %v", err)
	}
}

func TestCompleteUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"authentication_error","message":"unauthorized"}}`))
	}))
	defer ts.Close()

	client, _ := anthropic.NewAnthropicClient(anthropic.Config{APIKey: "test-key", BaseURL: ts.URL, MaxRetries: 0})

	_, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "claude-sonnet-4-6",
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestCompleteContextTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(textResponse("too late")))
	}))
	defer ts.Close()

	client, _ := anthropic.NewAnthropicClient(anthropic.Config{APIKey: "test-key", BaseURL: ts.URL, MaxRetries: 0})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.Complete(ctx, ports.LLMRequest{
		Model:     "claude-sonnet-4-6",
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got: %v", err)
	}
}
