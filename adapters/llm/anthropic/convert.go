package anthropic

import (
	"encoding/json"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"

	"github.com/aescanero/dago/libs/ports"
)

// toAnthropicMessages converts []ports.Message → []anthropicsdk.MessageParam.
func toAnthropicMessages(messages []ports.Message) []anthropicsdk.MessageParam {
	params := make([]anthropicsdk.MessageParam, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "assistant":
			params = append(params, anthropicsdk.NewAssistantMessage(anthropicsdk.NewTextBlock(m.Content)))
		case "tool_result":
			params = append(params, anthropicsdk.NewUserMessage(
				anthropicsdk.NewToolResultBlock(m.ToolUseID, m.Content, false),
			))
		default: // "user"
			params = append(params, anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(m.Content)))
		}
	}
	return params
}

// toAnthropicTools converts []ports.ToolDefinition → []anthropicsdk.ToolUnionParam.
func toAnthropicTools(tools []ports.ToolDefinition) []anthropicsdk.ToolUnionParam {
	result := make([]anthropicsdk.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		inputSchema := param.Override[anthropicsdk.ToolInputSchemaParam](json.RawMessage(t.InputSchema))
		tp := anthropicsdk.ToolParam{
			Name:        t.Name,
			Description: param.NewOpt(t.Description),
			InputSchema: inputSchema,
		}
		result = append(result, anthropicsdk.ToolUnionParam{OfTool: &tp})
	}
	return result
}

// fromAnthropicResponse converts anthropicsdk.Message → ports.LLMResponse.
func fromAnthropicResponse(msg *anthropicsdk.Message) ports.LLMResponse {
	resp := ports.LLMResponse{
		StopReason:   string(msg.StopReason),
		InputTokens:  int(msg.Usage.InputTokens),
		OutputTokens: int(msg.Usage.OutputTokens),
	}
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			resp.Content = block.Text
		case "tool_use":
			resp.ToolUses = append(resp.ToolUses, ports.ToolUse{
				ID:    block.ID,
				Name:  block.Name,
				Input: json.RawMessage(block.Input),
			})
		}
	}
	return resp
}
