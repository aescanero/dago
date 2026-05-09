package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aescanero/dago/adapters/llm/fake"
	"github.com/aescanero/dago/libs/domain"
	"github.com/aescanero/dago/libs/ports"
)

// fakePublisher is a test-private EventPublisher that records published events.
type fakePublisher struct {
	published []publishedCall
}

type publishedCall struct {
	event domain.Event
	opts  ports.PublishOptions
}

func (f *fakePublisher) Publish(_ context.Context, event domain.Event, opts ports.PublishOptions) error {
	f.published = append(f.published, publishedCall{event, opts})
	return nil
}

func (f *fakePublisher) Close() error { return nil }

// makeData builds a minimal NodeExecuteRequestedData for tests.
func makeData(t *testing.T, configJSON string) NodeExecuteRequestedData {
	t.Helper()
	return NodeExecuteRequestedData{
		ExecutionID: "exec-1",
		GraphID:     "graph-1",
		NodeID:      "node-1",
		NodeKey:     "llm_node",
		Pattern:     "llm_call",
		Config:      json.RawMessage(configJSON),
		Variables:   map[string]any{},
		Messages:    []ports.Message{{Role: "user", Content: "hello"}},
	}
}

func TestLLMCallHandler_Success(t *testing.T) {
	llm := &fake.FakeLLMClient{
		Responses: []ports.LLMResponse{{Content: "ok response", StopReason: "end_turn"}},
	}
	pub := &fakePublisher{}
	h := NewLLMCallHandler(llm, pub)
	data := makeData(t, `{"model":"claude-sonnet-4-6","max_tokens":100}`)

	err := h.Handle(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.published))
	}
	if pub.published[0].opts.Stream != domain.StreamNodeExecuted {
		t.Errorf("stream = %q, want %q", pub.published[0].opts.Stream, domain.StreamNodeExecuted)
	}
	var payload NodeExecutedData
	if err := json.Unmarshal(pub.published[0].event.Data, &payload); err != nil {
		t.Fatalf("unmarshal NodeExecutedData: %v", err)
	}
	var update map[string]any
	if err := json.Unmarshal(payload.VariablesUpdate, &update); err != nil {
		t.Fatalf("unmarshal variables_update: %v", err)
	}
	if update["response"] != "ok response" {
		t.Errorf("variables_update.response = %v, want 'ok response'", update["response"])
	}
	if payload.DurationMs < 0 {
		t.Errorf("duration_ms = %d, want >= 0", payload.DurationMs)
	}
}

func TestLLMCallHandler_WithInputMapping(t *testing.T) {
	llm := &fake.FakeLLMClient{
		Responses: []ports.LLMResponse{{Content: "resp", StopReason: "end_turn"}},
	}
	pub := &fakePublisher{}
	h := NewLLMCallHandler(llm, pub)
	data := makeData(t, `{"model":"claude-sonnet-4-6","input_mapping":{"user_message":"state.variables.query"}}`)
	data.Variables = map[string]any{"query": "explain dago"}
	data.Messages = nil

	if err := h.Handle(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(llm.Calls) != 1 {
		t.Fatalf("expected 1 LLM call, got %d", len(llm.Calls))
	}
	if len(llm.Calls[0].Messages) == 0 || llm.Calls[0].Messages[0].Content != "explain dago" {
		t.Errorf("LLM message content = %v, want 'explain dago'", llm.Calls[0].Messages)
	}
}

func TestLLMCallHandler_WithOutputMapping(t *testing.T) {
	llm := &fake.FakeLLMClient{
		Responses: []ports.LLMResponse{{Content: "mapped response", StopReason: "end_turn"}},
	}
	pub := &fakePublisher{}
	h := NewLLMCallHandler(llm, pub)
	data := makeData(t, `{"model":"claude-sonnet-4-6","output_mapping":{"state.variables.answer":"output.content"}}`)

	if err := h.Handle(context.Background(), data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload NodeExecutedData
	if err := json.Unmarshal(pub.published[0].event.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var update map[string]any
	if err := json.Unmarshal(payload.VariablesUpdate, &update); err != nil {
		t.Fatalf("unmarshal variables_update: %v", err)
	}
	if update["answer"] != "mapped response" {
		t.Errorf("variables_update.answer = %v, want 'mapped response'", update["answer"])
	}
}

func TestLLMCallHandler_RateLimited(t *testing.T) {
	llm := &fake.FakeLLMClient{
		Errors: []error{fmt.Errorf("test: %w", domain.ErrRateLimited)},
	}
	pub := &fakePublisher{}
	h := NewLLMCallHandler(llm, pub)
	data := makeData(t, `{"model":"claude-sonnet-4-6"}`)

	err := h.Handle(context.Background(), data)
	if err == nil {
		t.Fatal("expected handler to return error, got nil")
	}
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.published))
	}
	if pub.published[0].opts.Stream != domain.StreamNodeExecuteFailed {
		t.Errorf("stream = %q, want %q", pub.published[0].opts.Stream, domain.StreamNodeExecuteFailed)
	}
	var payload NodeExecuteFailedData
	if err := json.Unmarshal(pub.published[0].event.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ErrorCode != "rate_limited" {
		t.Errorf("error_code = %q, want 'rate_limited'", payload.ErrorCode)
	}
	if !payload.Retryable {
		t.Error("retryable = false, want true")
	}
}

func TestLLMCallHandler_ProviderUnavailable(t *testing.T) {
	llm := &fake.FakeLLMClient{
		Errors: []error{domain.ErrProviderUnavailable},
	}
	pub := &fakePublisher{}
	h := NewLLMCallHandler(llm, pub)
	data := makeData(t, `{"model":"claude-sonnet-4-6"}`)

	err := h.Handle(context.Background(), data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var payload NodeExecuteFailedData
	if err := json.Unmarshal(pub.published[0].event.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ErrorCode != "provider_unavailable" {
		t.Errorf("error_code = %q, want 'provider_unavailable'", payload.ErrorCode)
	}
	if !payload.Retryable {
		t.Error("retryable = false, want true")
	}
}

func TestLLMCallHandler_Unauthorized(t *testing.T) {
	llm := &fake.FakeLLMClient{
		Errors: []error{domain.ErrUnauthorized},
	}
	pub := &fakePublisher{}
	h := NewLLMCallHandler(llm, pub)
	data := makeData(t, `{"model":"claude-sonnet-4-6"}`)

	err := h.Handle(context.Background(), data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var payload NodeExecuteFailedData
	if err := json.Unmarshal(pub.published[0].event.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ErrorCode != "unauthorized" {
		t.Errorf("error_code = %q, want 'unauthorized'", payload.ErrorCode)
	}
	if payload.Retryable {
		t.Error("retryable = true, want false")
	}
}

func TestLLMCallHandler_ExecutionError(t *testing.T) {
	llm := &fake.FakeLLMClient{
		Errors: []error{fmt.Errorf("internal error")},
	}
	pub := &fakePublisher{}
	h := NewLLMCallHandler(llm, pub)
	data := makeData(t, `{"model":"claude-sonnet-4-6"}`)

	err := h.Handle(context.Background(), data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var payload NodeExecuteFailedData
	if err := json.Unmarshal(pub.published[0].event.Data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ErrorCode != "execution_error" {
		t.Errorf("error_code = %q, want 'execution_error'", payload.ErrorCode)
	}
	if payload.Retryable {
		t.Error("retryable = true, want false")
	}
}
