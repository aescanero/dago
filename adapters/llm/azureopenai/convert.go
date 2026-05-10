package azureopenai

import (
	"encoding/json"

	goai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/ports"
)

func toOpenAIMessages(messages []ports.Message) []goai.ChatCompletionMessage {
	result := make([]goai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		msg := goai.ChatCompletionMessage{Content: m.Content}
		switch m.Role {
		case "assistant":
			msg.Role = goai.ChatMessageRoleAssistant
		case "tool_result":
			msg.Role = goai.ChatMessageRoleTool
			msg.ToolCallID = m.ToolUseID
		default:
			msg.Role = goai.ChatMessageRoleUser
		}
		result = append(result, msg)
	}
	return result
}

func toOpenAITools(tools []ports.ToolDefinition) []goai.Tool {
	result := make([]goai.Tool, 0, len(tools))
	for _, t := range tools {
		result = append(result, goai.Tool{
			Type: goai.ToolTypeFunction,
			Function: &goai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(t.InputSchema),
			},
		})
	}
	return result
}

func fromOpenAIResponse(resp goai.ChatCompletionResponse) ports.LLMResponse {
	result := ports.LLMResponse{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}
	if len(resp.Choices) == 0 {
		return result
	}
	choice := resp.Choices[0]
	result.StopReason = convertFinishReason(string(choice.FinishReason))
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

func convertFinishReason(reason string) string {
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
