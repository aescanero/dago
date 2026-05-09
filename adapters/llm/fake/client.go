package fake

import (
	"context"

	"github.com/aescanero/dago/libs/ports"
)

// FakeLLMClient implements ports.LLMClient for tests.
// Returns responses from Responses in FIFO order.
// When the queue is exhausted, returns a default response.
type FakeLLMClient struct { //nolint:revive
	Responses []ports.LLMResponse
	Calls     []ports.LLMRequest
}

var _ ports.LLMClient = &FakeLLMClient{}

// Complete returns the next queued response or the default one.
// Always records the call in Calls. Never returns error.
func (f *FakeLLMClient) Complete(_ context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
	f.Calls = append(f.Calls, req)
	if len(f.Responses) > 0 {
		resp := f.Responses[0]
		f.Responses = f.Responses[1:]
		return resp, nil
	}
	return ports.LLMResponse{Content: "fake response", StopReason: "end_turn"}, nil
}
