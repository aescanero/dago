package ollama

import (
	"encoding/json"

	openai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/ports"
)

// toOpenAIMessages converts []ports.Message to []openai.ChatCompletionMessage.
func toOpenAIMessages(messages []ports.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		msg := openai.ChatCompletionMessage{Content: m.Content}
		switch m.Role {
		case "assistant":
			msg.Role = openai.ChatMessageRoleAssistant
		case "tool_result":
			msg.Role = openai.ChatMessageRoleTool
			msg.ToolCallID = m.ToolUseID
		default:
			msg.Role = openai.ChatMessageRoleUser
		}
		result = append(result, msg)
	}
	return result
}

// toOpenAITools converts []ports.ToolDefinition to []openai.Tool.
func toOpenAITools(tools []ports.ToolDefinition) []openai.Tool {
	result := make([]openai.Tool, 0, len(tools))
	for _, t := range tools {
		result = append(result, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(t.InputSchema),
			},
		})
	}
	return result
}

// fromOpenAIResponse converts openai.ChatCompletionResponse to ports.LLMResponse.
func fromOpenAIResponse(resp openai.ChatCompletionResponse) ports.LLMResponse {
	result := ports.LLMResponse{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}
	if len(resp.Choices) == 0 {
		return result
	}
	choice := resp.Choices[0]
	result.StopReason = ConvertFinishReason(string(choice.FinishReason))
	result.Content = choice.Message.Content
	for _, tc := range choice.Message.ToolCalls {
		result.ToolUses = append(result.ToolUses, ports.ToolUse{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}
	if len(result.ToolUses) > 0 {
		result.Content = ""
	}
	return result
}

// ConvertFinishReason maps OpenAI finish_reason to domain StopReason.
func ConvertFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	default:
		return "end_turn"
	}
}
