package mapping

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aescanero/dago/libs/ports"
)

// ErrUnsupportedPath is returned when a path expression is not recognised.
var ErrUnsupportedPath = errors.New("unsupported path")

// ApplyInputMapping builds []ports.Message for LLMRequest.
// If inputMapping is nil or empty, returns messages as-is.
func ApplyInputMapping(
	inputMapping map[string]string,
	variables map[string]any,
	messages []ports.Message,
) ([]ports.Message, error) {
	if len(inputMapping) == 0 {
		return messages, nil
	}
	result := make([]ports.Message, 0, len(inputMapping))
	for _, path := range inputMapping {
		content, err := resolveInputPath(path, variables, messages)
		if err != nil {
			return nil, err
		}
		result = append(result, ports.Message{Role: "user", Content: content})
	}
	return result, nil
}

func resolveInputPath(path string, variables map[string]any, messages []ports.Message) (string, error) {
	switch {
	case path == "state.messages[-1].content":
		if len(messages) == 0 {
			return "", fmt.Errorf("mapping/input: no messages available for path %q", path)
		}
		return messages[len(messages)-1].Content, nil
	case strings.HasPrefix(path, "state.variables."):
		name := strings.TrimPrefix(path, "state.variables.")
		val, ok := variables[name]
		if !ok {
			return "", fmt.Errorf("mapping/input: variable %q not found", name)
		}
		return fmt.Sprintf("%v", val), nil
	default:
		return "", fmt.Errorf("mapping/input: unsupported path %q: %w", path, ErrUnsupportedPath)
	}
}
