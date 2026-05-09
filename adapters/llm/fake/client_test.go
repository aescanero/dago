package fake_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/aescanero/dago/adapters/llm/fake"
	"github.com/aescanero/dago/libs/ports"
)

func TestFakeLLMClientQueue(t *testing.T) {
	resp1 := ports.LLMResponse{Content: "first", StopReason: "end_turn", InputTokens: 1, OutputTokens: 1}
	resp2 := ports.LLMResponse{Content: "second", StopReason: "end_turn", InputTokens: 2, OutputTokens: 2}

	f := &fake.FakeLLMClient{Responses: []ports.LLMResponse{resp1, resp2}}

	req1 := ports.LLMRequest{Model: "m1", Messages: []ports.Message{{Role: "user", Content: "q1"}}}
	req2 := ports.LLMRequest{Model: "m2", Messages: []ports.Message{{Role: "user", Content: "q2"}}}
	req3 := ports.LLMRequest{Model: "m3", Messages: []ports.Message{{Role: "user", Content: "q3"}}}

	got1, err := f.Complete(context.Background(), req1)
	if err != nil {
		t.Fatalf("call 1: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got1, resp1) {
		t.Errorf("call 1: got %+v, want %+v", got1, resp1)
	}

	got2, err := f.Complete(context.Background(), req2)
	if err != nil {
		t.Fatalf("call 2: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got2, resp2) {
		t.Errorf("call 2: got %+v, want %+v", got2, resp2)
	}

	got3, err := f.Complete(context.Background(), req3)
	if err != nil {
		t.Fatalf("call 3: unexpected error: %v", err)
	}
	wantDefault := ports.LLMResponse{Content: "fake response", StopReason: "end_turn"}
	if !reflect.DeepEqual(got3, wantDefault) {
		t.Errorf("call 3 (default): got %+v, want %+v", got3, wantDefault)
	}

	if len(f.Calls) != 3 {
		t.Fatalf("len(Calls) = %d, want 3", len(f.Calls))
	}
	if !reflect.DeepEqual(f.Calls[0], req1) {
		t.Errorf("Calls[0] = %+v, want %+v", f.Calls[0], req1)
	}
	if !reflect.DeepEqual(f.Calls[1], req2) {
		t.Errorf("Calls[1] = %+v, want %+v", f.Calls[1], req2)
	}
	if !reflect.DeepEqual(f.Calls[2], req3) {
		t.Errorf("Calls[2] = %+v, want %+v", f.Calls[2], req3)
	}
}
