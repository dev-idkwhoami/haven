package connection

import "encoding/json"

// RawMessage is the unified WebSocket message envelope.
type RawMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ErrorPayload is the standard error payload from the server.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
