package ws

import (
	"encoding/json"
	"log/slog"

	"haven/shared"
)

// WSMessage is the unified WebSocket message envelope.
type WSMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// HandlerFunc is the signature for a WS message handler.
type HandlerFunc func(client *Client, msg *WSMessage)

// Router dispatches incoming WebSocket messages to registered handlers by type.
type Router struct {
	handlers map[string]HandlerFunc
}

// NewRouter creates a new Router instance.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]HandlerFunc),
	}
}

// Register adds a handler for a specific message type.
func (r *Router) Register(msgType string, handler HandlerFunc) {
	r.handlers[msgType] = handler
}

// Route parses a raw WebSocket message and dispatches it to the appropriate handler.
func (r *Router) Route(client *Client, raw []byte) {
	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		slog.Debug("ws invalid json", "pubkey", client.PubKeyHex, "error", err)
		SendError(client, "unknown", "", shared.ErrBadRequest, "invalid JSON")
		return
	}

	if msg.Type == "" {
		SendError(client, "unknown", msg.ID, shared.ErrBadRequest, "missing message type")
		return
	}

	handler, ok := r.handlers[msg.Type]
	if !ok {
		slog.Debug("ws unknown message type", "pubkey", client.PubKeyHex, "type", msg.Type)
		SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "unknown message type")
		return
	}

	handler(client, &msg)
}

// SendOK sends a success response to a client.
func SendOK(client *Client, msgType, id string, payload any) {
	resp := map[string]any{
		"type": msgType + shared.SuffixOK,
	}
	if id != "" {
		resp["id"] = id
	}
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			slog.Error("marshal ok payload", "type", msgType, "error", err)
			return
		}
		resp["payload"] = json.RawMessage(data)
	}
	send(client, resp)
}

// SendError sends an error response to a client.
func SendError(client *Client, msgType, id, code, message string) {
	resp := map[string]any{
		"type": msgType + shared.SuffixError,
		"payload": map[string]string{
			"code":    code,
			"message": message,
		},
	}
	if id != "" {
		resp["id"] = id
	}
	send(client, resp)
}

// SendEvent sends a fire-and-forget event to a client.
func SendEvent(client *Client, eventType string, payload any) {
	resp := map[string]any{
		"type": eventType,
	}
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			slog.Error("marshal event payload", "type", eventType, "error", err)
			return
		}
		resp["payload"] = json.RawMessage(data)
	}
	send(client, resp)
}

// MarshalEvent marshals an event message into bytes for broadcasting.
func MarshalEvent(eventType string, payload any) ([]byte, error) {
	resp := map[string]any{
		"type": eventType,
	}
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		resp["payload"] = json.RawMessage(data)
	}
	return json.Marshal(resp)
}

func send(client *Client, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("ws marshal response", "error", err)
		return
	}
	client.Send(data)
}
