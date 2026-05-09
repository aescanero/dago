package ports

import (
	"context"
	"encoding/json"
)

// Message represents a conversation turn.
type Message struct {
	Role      string // "user" | "assistant" | "tool_result"
	Content   string
	ToolUseID string // only for Role == "tool_result"
}

// ToolDefinition describes a tool available to the LLM.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage // JSON Schema of the tool input
}

// LLMRequest encapsulates a call to the LLM.
type LLMRequest struct {
	Model       string
	System      string
	Messages    []Message
	Tools       []ToolDefinition
	MaxTokens   int
	Temperature float64
}

// ToolUse represents a tool invocation requested by the LLM.
type ToolUse struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// LLMResponse encapsulates the LLM response.
type LLMResponse struct {
	Content      string    // generated text; empty if StopReason == "tool_use"
	StopReason   string    // "end_turn" | "tool_use" | "max_tokens"
	ToolUses     []ToolUse // populated if StopReason == "tool_use"
	InputTokens  int
	OutputTokens int
}

// LLMClient is the output port for calls to language models.
type LLMClient interface {
	Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
