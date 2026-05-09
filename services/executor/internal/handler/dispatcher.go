package handler

import (
	"context"
	"fmt"
)

// Dispatcher routes a NodeExecuteRequestedData to the correct NodeHandler by pattern.
type Dispatcher struct {
	handlers map[string]NodeHandler
}

// NewDispatcher creates a Dispatcher with the given handler map.
func NewDispatcher(handlers map[string]NodeHandler) *Dispatcher {
	return &Dispatcher{handlers: handlers}
}

// Dispatch delegates to the handler registered for data.Pattern.
// Returns an error wrapping "unsupported pattern" if no handler is registered.
func (d *Dispatcher) Dispatch(ctx context.Context, data NodeExecuteRequestedData) error {
	h, ok := d.handlers[data.Pattern]
	if !ok {
		return fmt.Errorf("dispatcher: unsupported pattern %q", data.Pattern)
	}
	if err := h.Handle(ctx, data); err != nil {
		return fmt.Errorf("dispatcher: handle %s: %w", data.Pattern, err)
	}
	return nil
}
