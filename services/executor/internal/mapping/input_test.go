package mapping

import (
	"testing"

	"github.com/aescanero/dago/libs/ports"
)

func TestApplyInputMapping_NoMapping(t *testing.T) {
	msgs := []ports.Message{{Role: "user", Content: "hello"}}
	got, err := ApplyInputMapping(nil, map[string]any{}, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Content != "hello" {
		t.Errorf("got %+v, want original messages", got)
	}
}

func TestApplyInputMapping_EmptyMapping(t *testing.T) {
	msgs := []ports.Message{{Role: "user", Content: "test"}}
	got, err := ApplyInputMapping(map[string]string{}, map[string]any{}, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Content != "test" {
		t.Errorf("got %+v, want original messages", got)
	}
}

func TestApplyInputMapping_StateMessagesLast(t *testing.T) {
	msgs := []ports.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "resp"},
		{Role: "user", Content: "second"},
	}
	mapping := map[string]string{"user_message": "state.messages[-1].content"}
	got, err := ApplyInputMapping(mapping, map[string]any{}, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Content != "second" {
		t.Errorf("got %+v, want content='second'", got)
	}
}

func TestApplyInputMapping_StateVariables(t *testing.T) {
	mapping := map[string]string{"user_message": "state.variables.query"}
	vars := map[string]any{"query": "what is dago?"}
	got, err := ApplyInputMapping(mapping, vars, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Content != "what is dago?" {
		t.Errorf("got %+v, want content='what is dago?'", got)
	}
	if got[0].Role != "user" {
		t.Errorf("role = %q, want 'user'", got[0].Role)
	}
}

func TestApplyInputMapping_StateVariables_Missing(t *testing.T) {
	mapping := map[string]string{"user_message": "state.variables.nonexistent"}
	_, err := ApplyInputMapping(mapping, map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for missing variable, got nil")
	}
}

func TestApplyInputMapping_UnknownPath(t *testing.T) {
	mapping := map[string]string{"x": "state.unknown.path"}
	_, err := ApplyInputMapping(mapping, map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported path, got nil")
	}
}
