package valkey

import (
	"encoding/json"
	"fmt"

	"github.com/aescanero/dago/libs/domain"
)

const envelopeField = "envelope"

// marshalEnvelope serializes a domain.Event to the map format expected by XADD.
func marshalEnvelope(evt domain.Event) (map[string]any, error) {
	b, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("eventbus marshal envelope: %w", err)
	}
	return map[string]any{envelopeField: string(b)}, nil
}

// unmarshalEnvelope deserializes a Valkey Streams entry back to a domain.Event.
func unmarshalEnvelope(fields map[string]string) (domain.Event, error) {
	raw, ok := fields[envelopeField]
	if !ok {
		return domain.Event{}, fmt.Errorf("eventbus unmarshal envelope: missing %q field", envelopeField)
	}
	var evt domain.Event
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		return domain.Event{}, fmt.Errorf("eventbus unmarshal envelope: %w", err)
	}
	return evt, nil
}
