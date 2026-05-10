package openai

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	goai "github.com/sashabaranov/go-openai"

	"github.com/aescanero/dago/libs/domain"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	var apiErr *goai.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.HTTPStatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: %s", domain.ErrUnauthorized, apiErr.Message)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: %s", domain.ErrRateLimited, apiErr.Message)
		case http.StatusInternalServerError, http.StatusServiceUnavailable:
			return fmt.Errorf("%w: %s", domain.ErrProviderUnavailable, apiErr.Message)
		}
	}
	return fmt.Errorf("openai: %w", err)
}
