package ollama_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aescanero/dago/adapters/llm/ollama"
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

func TestOllamaCompleteText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponse("hello ollama", "stop")))
	}))
	defer ts.Close()

	client := ollama.NewOllamaClient(ollama.Config{BaseURL: ts.URL, DefaultModel: "mixtral"})

	resp, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "mixtral",
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "hello ollama" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello ollama")
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

func TestOllamaCompleteWithTools(t *testing.T) {
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

	client := ollama.NewOllamaClient(ollama.Config{BaseURL: ts.URL, DefaultModel: "mixtral"})

	schema := json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`)
	resp, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "mixtral",
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
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
}

func TestOllamaCompleteServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer ts.Close()

	client := ollama.NewOllamaClient(ollama.Config{BaseURL: ts.URL, DefaultModel: "mixtral"})

	_, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "mixtral",
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

func TestOllamaCompleteContextTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponse("too late", "stop")))
	}))
	defer ts.Close()

	client := ollama.NewOllamaClient(ollama.Config{BaseURL: ts.URL, DefaultModel: "mixtral"})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.Complete(ctx, ports.LLMRequest{
		Model:     "mixtral",
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

func TestOllamaCompleteModelDefault(t *testing.T) {
	var receivedModel string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.Unmarshal(body["model"], &receivedModel)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponse("ok", "stop")))
	}))
	defer ts.Close()

	client := ollama.NewOllamaClient(ollama.Config{BaseURL: ts.URL})

	_, err := client.Complete(context.Background(), ports.LLMRequest{
		Model:     "",
		Messages:  []ports.Message{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if receivedModel != "mixtral" {
		t.Errorf("model sent to server = %q, want %q", receivedModel, "mixtral")
	}
}

func TestOllamaConvertFinishReason(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"stop", "end_turn"},
		{"tool_calls", "tool_use"},
		{"length", "max_tokens"},
		{"", "end_turn"},
		{"unknown", "end_turn"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := ollama.ConvertFinishReason(tc.input)
			if got != tc.want {
				t.Errorf("ConvertFinishReason(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
