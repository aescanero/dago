package mapping

import (
	"testing"

	"github.com/aescanero/dago/libs/ports"
)

func TestApplyOutputMapping_NoMapping(t *testing.T) {
	resp := ports.LLMResponse{Content: "generated response", StopReason: "end_turn"}
	got, err := ApplyOutputMapping(nil, resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["response"] != "generated response" {
		t.Errorf("got %v, want response='generated response'", got)
	}
}

func TestApplyOutputMapping_ContentToVariable(t *testing.T) {
	resp := ports.LLMResponse{Content: "summary", StopReason: "end_turn"}
	mapping := map[string]string{"state.variables.summary": "output.content"}
	got, err := ApplyOutputMapping(mapping, resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["summary"] != "summary" {
		t.Errorf("got %v, want summary='summary'", got)
	}
}

func TestApplyOutputMapping_StopReasonToVariable(t *testing.T) {
	resp := ports.LLMResponse{Content: "", StopReason: "max_tokens"}
	mapping := map[string]string{"state.variables.reason": "output.stop_reason"}
	got, err := ApplyOutputMapping(mapping, resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["reason"] != "max_tokens" {
		t.Errorf("got %v, want reason='max_tokens'", got)
	}
}

func TestApplyOutputMapping_MultipleTargets(t *testing.T) {
	resp := ports.LLMResponse{Content: "text", StopReason: "end_turn"}
	mapping := map[string]string{
		"state.variables.text": "output.content",
		"state.variables.stop": "output.stop_reason",
	}
	got, err := ApplyOutputMapping(mapping, resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["text"] != "text" {
		t.Errorf("got text=%v, want 'text'", got["text"])
	}
	if got["stop"] != "end_turn" {
		t.Errorf("got stop=%v, want 'end_turn'", got["stop"])
	}
}

func TestApplyOutputMapping_UnknownSourcePath(t *testing.T) {
	resp := ports.LLMResponse{Content: "x"}
	mapping := map[string]string{"state.variables.x": "output.unknown_field"}
	_, err := ApplyOutputMapping(mapping, resp)
	if err == nil {
		t.Fatal("expected error for unsupported output field, got nil")
	}
}

func TestApplyOutputMapping_InvalidTargetPath(t *testing.T) {
	resp := ports.LLMResponse{Content: "x"}
	mapping := map[string]string{"not.a.state.variable": "output.content"}
	_, err := ApplyOutputMapping(mapping, resp)
	if err == nil {
		t.Fatal("expected error for invalid target path, got nil")
	}
}
