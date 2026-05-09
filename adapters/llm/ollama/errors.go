package ollama

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	openai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/domain"
)

// mapOllamaError converts a go-openai error to a domain error.
func mapOllamaError(op string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusInternalServerError {
		return fmt.Errorf("%s: %w", op, domain.ErrProviderUnavailable)
	}
	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) && reqErr.HTTPStatusCode == http.StatusInternalServerError {
		return fmt.Errorf("%s: %w", op, domain.ErrProviderUnavailable)
	}
	return fmt.Errorf("%s: %w", op, err)
}
