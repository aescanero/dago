package mapping

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aescanero/dago/libs/ports"
)

// ErrInvalidTarget is returned when an output mapping target is not state.variables.<name>.
var ErrInvalidTarget = errors.New("invalid target path")

// ApplyOutputMapping builds map[string]any of variable updates.
// If outputMapping is nil returns {"response": response.Content}.
func ApplyOutputMapping(
	outputMapping map[string]string,
	response ports.LLMResponse,
) (map[string]any, error) {
	if len(outputMapping) == 0 {
		return map[string]any{"response": response.Content}, nil
	}
	result := make(map[string]any, len(outputMapping))
	for target, src := range outputMapping {
		if !strings.HasPrefix(target, "state.variables.") {
			return nil, fmt.Errorf("mapping/output: %w: %q", ErrInvalidTarget, target)
		}
		name := strings.TrimPrefix(target, "state.variables.")
		val, err := resolveOutputSource(src, response)
		if err != nil {
			return nil, err
		}
		result[name] = val
	}
	return result, nil
}

func resolveOutputSource(src string, resp ports.LLMResponse) (string, error) {
	switch src {
	case "output.content":
		return resp.Content, nil
	case "output.stop_reason":
		return resp.StopReason, nil
	default:
		return "", fmt.Errorf("mapping/output: unsupported output field %q: %w", src, ErrUnsupportedPath)
	}
}
