package protocol

import (
	"encoding/json"
	"testing"
)

func TestFrameSerialization(t *testing.T) {
	req := RequestPayload{
		ID:     "req_123",
		Method: "GET",
		Path:   "/api/health",
	}

	payloadBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request payload: %v", err)
	}

	frame := Frame{
		Type:    MsgRequest,
		ID:      req.ID,
		Payload: payloadBytes,
	}

	frameBytes, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("failed to marshal frame: %v", err)
	}

	var unmarshaledFrame Frame
	if err := json.Unmarshal(frameBytes, &unmarshaledFrame); err != nil {
		t.Fatalf("failed to unmarshal frame: %v", err)
	}

	if unmarshaledFrame.Type != MsgRequest || unmarshaledFrame.ID != "req_123" {
		t.Errorf("unmarshaled frame mismatch: %+v", unmarshaledFrame)
	}
}
