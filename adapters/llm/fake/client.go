package fake

import (
	"context"

	"github.com/aescanero/dago/libs/ports"
)

// FakeLLMClient implements ports.LLMClient for tests.
// Returns responses from Responses in FIFO order; errors from Errors in FIFO order.
// Errors are drained before Responses. When both are exhausted, returns a default response.
type FakeLLMClient struct { //nolint:revive
	Responses []ports.LLMResponse
	Errors    []error
	Calls     []ports.LLMRequest
}

var _ ports.LLMClient = &FakeLLMClient{}

// Complete returns the next queued error or response, recording the call in Calls.
func (f *FakeLLMClient) Complete(_ context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
	f.Calls = append(f.Calls, req)
	if len(f.Errors) > 0 {
		err := f.Errors[0]
		f.Errors = f.Errors[1:]
		return ports.LLMResponse{}, err
	}
	if len(f.Responses) > 0 {
		resp := f.Responses[0]
		f.Responses = f.Responses[1:]
		return resp, nil
	}
	return ports.LLMResponse{Content: "fake response", StopReason: "end_turn"}, nil
}
