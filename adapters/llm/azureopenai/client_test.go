package azureopenai_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	azureopenaillm "github.com/aescanero/dago/adapters/llm/azureopenai"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

func chatResponse(content, finishReason string) string {
	return `{
		"id":"chatcmpl-01","object":"chat.completion",
		"choices":[{"message":{"role":"assistant","content":"` + content + `"},"finish_reason":"` + finishReason + `","index":0}],
		"usage":{"prompt_tokens":10,"completion_tokens":5}
	}`
}

func TestAzureOpenAICompleteText(t *testing.T) {
	const deployment = "gpt-4o-deployment"
	var receivedModel string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.Unmarshal(body["model"], &receivedModel)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponse("hello azure", "stop")))
	}))
	defer ts.Close()

	client, err := azureopenaillm.NewAzureOpenAIClient(azureopenaillm.Config{
		APIKey:     "test-key",
		Endpoint:   ts.URL,
		Deployment: deployment,
		APIVersion: "2024-08-01-preview",
	})
	if err != nil {
		t.Fatalf("NewAzureOpenAIClient: %v", err)
	}

	resp, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "some-other-model", // should be overridden by Deployment
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "hello azure" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello azure")
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "end_turn")
	}
	if receivedModel != deployment {
		t.Errorf("model sent to server = %q, want deployment %q", receivedModel, deployment)
	}
}

func TestAzureOpenAICompleteWithTools(t *testing.T) {
	toolResp := `{
		"id":"chatcmpl-02","object":"chat.completion",
		"choices":[{
			"message":{"role":"assistant","content":"","tool_calls":[{"id":"tc_01","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"NYC\"}"}}]},
			"finish_reason":"tool_calls","index":0
		}],
		"usage":{"prompt_tokens":15,"completion_tokens":8}
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toolResp))
	}))
	defer ts.Close()

	client, err := azureopenaillm.NewAzureOpenAIClient(azureopenaillm.Config{
		APIKey:     "test-key",
		Endpoint:   ts.URL,
		Deployment: "gpt-4o-deployment",
		APIVersion: "2024-08-01-preview",
	})
	if err != nil {
		t.Fatalf("NewAzureOpenAIClient: %v", err)
	}

	schema := json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`)
	resp, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "gpt-4o",
		Messages:  []ports.Message{{Role: "user", Content: "Weather?"}},
		Tools:     []ports.ToolDefinition{{Name: "get_weather", Description: "Get weather", InputSchema: schema}},
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
}

func TestAzureOpenAICompleteRateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded","type":"requests","code":"rate_limit_exceeded"}}`))
	}))
	defer ts.Close()

	client, err := azureopenaillm.NewAzureOpenAIClient(azureopenaillm.Config{
		APIKey:     "test-key",
		Endpoint:   ts.URL,
		Deployment: "gpt-4o-deployment",
		APIVersion: "2024-08-01-preview",
	})
	if err != nil {
		t.Fatalf("NewAzureOpenAIClient: %v", err)
	}

	_, err = client.Complete(context.Background(), ports.LLMRequest{
		Model:     "gpt-4o",
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

func TestAzureOpenAICompleteServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal server error","type":"server_error"}}`))
	}))
	defer ts.Close()

	client, err := azureopenaillm.NewAzureOpenAIClient(azureopenaillm.Config{
		APIKey:     "test-key",
		Endpoint:   ts.URL,
		Deployment: "gpt-4o-deployment",
		APIVersion: "2024-08-01-preview",
	})
	if err != nil {
		t.Fatalf("NewAzureOpenAIClient: %v", err)
	}

	_, err = client.Complete(context.Background(), ports.LLMRequest{
		Model:     "gpt-4o",
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

func TestAzureOpenAICompleteUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	defer ts.Close()

	client, err := azureopenaillm.NewAzureOpenAIClient(azureopenaillm.Config{
		APIKey:     "invalid-key",
		Endpoint:   ts.URL,
		Deployment: "gpt-4o-deployment",
		APIVersion: "2024-08-01-preview",
	})
	if err != nil {
		t.Fatalf("NewAzureOpenAIClient: %v", err)
	}

	_, err = client.Complete(context.Background(), ports.LLMRequest{
		Model:     "gpt-4o",
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

func TestAzureOpenAICompleteContextTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponse("too late", "stop")))
	}))
	defer ts.Close()

	client, err := azureopenaillm.NewAzureOpenAIClient(azureopenaillm.Config{
		APIKey:     "test-key",
		Endpoint:   ts.URL,
		Deployment: "gpt-4o-deployment",
		APIVersion: "2024-08-01-preview",
	})
	if err != nil {
		t.Fatalf("NewAzureOpenAIClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = client.Complete(ctx, ports.LLMRequest{
		Model:     "gpt-4o",
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
