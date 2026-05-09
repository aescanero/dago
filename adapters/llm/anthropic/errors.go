package anthropic

import (
	"context"
	"errors"
	"fmt"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"

	"github.com/aescanero/dago/libs/domain"
)

// mapAnthropicError converts an SDK error into a domain error.
func mapAnthropicError(op string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	var apiErr *anthropicsdk.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return fmt.Errorf("%s: %w", op, domain.ErrUnauthorized)
		case 429:
			return fmt.Errorf("%s: %w", op, domain.ErrRateLimited)
		case 500, 529:
			return fmt.Errorf("%s: %w", op, domain.ErrProviderUnavailable)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
