package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	havenCrypto "haven/client/crypto"
	"haven/shared"
)

const (
	requestTimeout = 15 * time.Second
	sendBufferSize = 256
)

// SessionState holds the authenticated session state for a connection.
type SessionState struct {
	Token         string
	UserID        string
	EncryptionKey []byte // nil if wss://, set if ws://
	NonceCounter  uint64 // for app-layer encryption frame counter
}

// VoiceState tracks the current voice session on this connection.
type VoiceState struct {
	ChannelID string
	RoomID    string
}

// ServerConnection manages a single WebSocket connection to a server.
type ServerConnection struct {
	ServerID  int64
	Address   string
	Conn      *websocket.Conn
	Session   SessionState
	SendCh    chan []byte
	Connected bool

	// OnUnexpectedClose is called when the connection drops unexpectedly
	// (not from an intentional disconnect). Used for auto-reconnect.
	OnUnexpectedClose func(serverID int64)

	pendingReqs map[string]chan RawMessage
	voiceRoom   *VoiceState

	mu               sync.Mutex
	sendMu           sync.Mutex // serializes nonce increment + encrypt + channel send
	wailsCtx         context.Context
	emitFunc         func(ctx context.Context, eventName string, data ...interface{})
	sendNonce        atomic.Uint64
	recvNonce        atomic.Uint64
	cancelFunc       context.CancelFunc
	intentionalClose bool
}

// NewServerConnection creates a new server connection (not yet connected).
func NewServerConnection(serverID int64, address string, wailsCtx context.Context, emitFunc func(ctx context.Context, eventName string, data ...interface{})) *ServerConnection {
	return &ServerConnection{
		ServerID:    serverID,
		Address:     address,
		SendCh:      make(chan []byte, sendBufferSize),
		pendingReqs: make(map[string]chan RawMessage),
		wailsCtx:    wailsCtx,
		emitFunc:    emitFunc,
	}
}

// SetConn sets the underlying websocket connection and starts read/write goroutines.
func (sc *ServerConnection) SetConn(conn *websocket.Conn) {
	sc.mu.Lock()
	sc.Conn = conn
	sc.Connected = true
	sc.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	sc.cancelFunc = cancel

	go sc.readLoop(ctx)
	go sc.writeLoop(ctx)
}

// Request sends a request and waits for a typed response (blocking, with timeout).
func (sc *ServerConnection) Request(msgType string, payload interface{}) (RawMessage, error) {
	msgID := uuid.New().String()

	respCh := make(chan RawMessage, 1)
	sc.mu.Lock()
	sc.pendingReqs[msgID] = respCh
	sc.mu.Unlock()

	defer func() {
		sc.mu.Lock()
		delete(sc.pendingReqs, msgID)
		sc.mu.Unlock()
	}()

	if err := sc.sendRaw(msgType, msgID, payload); err != nil {
		return RawMessage{}, fmt.Errorf("request send: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp.Type == msgType+shared.SuffixError {
			var errPayload ErrorPayload
			if err := json.Unmarshal(resp.Payload, &errPayload); err != nil {
				return resp, fmt.Errorf("server error (unparseable)")
			}
			return resp, fmt.Errorf("server error %s: %s", errPayload.Code, errPayload.Message)
		}
		return resp, nil
	case <-time.After(requestTimeout):
		return RawMessage{}, fmt.Errorf("request timeout for %s", msgType)
	}
}

// Send sends a fire-and-forget message (no response expected).
func (sc *ServerConnection) Send(msgType string, payload interface{}) error {
	return sc.sendRaw(msgType, "", payload)
}

// SendRawBytes queues raw bytes to be sent over the websocket.
func (sc *ServerConnection) SendRawBytes(data []byte) error {
	sc.mu.Lock()
	connected := sc.Connected
	sc.mu.Unlock()

	if !connected {
		return fmt.Errorf("not connected")
	}

	select {
	case sc.SendCh <- data:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// MarkIntentionalClose marks this connection as being closed intentionally
// (user disconnect, leave server, etc.) so auto-reconnect is suppressed.
func (sc *ServerConnection) MarkIntentionalClose() {
	sc.mu.Lock()
	sc.intentionalClose = true
	sc.mu.Unlock()
}

// Close closes the connection and cleans up.
func (sc *ServerConnection) Close() {
	sc.mu.Lock()
	wasConnected := sc.Connected
	intentional := sc.intentionalClose
	sc.Connected = false
	conn := sc.Conn
	sc.Conn = nil

	for _, ch := range sc.pendingReqs {
		close(ch)
	}
	sc.pendingReqs = make(map[string]chan RawMessage)
	sc.mu.Unlock()

	if sc.cancelFunc != nil {
		sc.cancelFunc()
	}

	if conn != nil {
		conn.Close()
	}

	if wasConnected && sc.emitFunc != nil {
		sc.emitFunc(sc.wailsCtx, "server:disconnected", map[string]interface{}{
			"serverID": sc.ServerID,
		})
	}

	// Trigger auto-reconnect for unexpected disconnects.
	if wasConnected && !intentional && sc.OnUnexpectedClose != nil {
		go sc.OnUnexpectedClose(sc.ServerID)
	}
}

// GetVoiceRoom returns the current voice state, or nil if not in voice.
func (sc *ServerConnection) GetVoiceRoom() *VoiceState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.voiceRoom
}

// SetVoiceRoom sets the current voice state.
func (sc *ServerConnection) SetVoiceRoom(vs *VoiceState) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.voiceRoom = vs
}

func (sc *ServerConnection) sendRaw(msgType, msgID string, payload interface{}) error {
	sc.mu.Lock()
	connected := sc.Connected
	sc.mu.Unlock()

	if !connected {
		return fmt.Errorf("not connected to %s", sc.Address)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	msg := RawMessage{
		Type:    msgType,
		ID:      msgID,
		Payload: payloadBytes,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Lock sendMu to ensure nonce ordering: if goroutine A gets nonce N and
	// goroutine B gets nonce N+1, A's encrypted message must enter SendCh first.
	// Without this, the server would receive messages out of nonce order and
	// fail to decrypt (chacha20poly1305 authentication failed).
	sc.sendMu.Lock()
	if sc.Session.EncryptionKey != nil {
		nonce := sc.sendNonce.Add(1) - 1
		data, err = havenCrypto.Encrypt(sc.Session.EncryptionKey, nonce, data)
		if err != nil {
			sc.sendMu.Unlock()
			return fmt.Errorf("encrypt message: %w", err)
		}
	}

	select {
	case sc.SendCh <- data:
		sc.sendMu.Unlock()
		return nil
	default:
		sc.sendMu.Unlock()
		return fmt.Errorf("send buffer full")
	}
}

func (sc *ServerConnection) readLoop(ctx context.Context) {
	defer sc.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, data, err := sc.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("ws read error", "server", sc.Address, "error", err)
			}
			return
		}

		if sc.Session.EncryptionKey != nil {
			nonce := sc.recvNonce.Add(1) - 1
			data, err = havenCrypto.Decrypt(sc.Session.EncryptionKey, nonce, data)
			if err != nil {
				slog.Error("ws decrypt error", "server", sc.Address, "error", err)
				return
			}
		}

		var msg RawMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			slog.Warn("ws unmarshal error", "server", sc.Address, "error", err)
			continue
		}

		sc.handleMessage(msg)
	}
}

func (sc *ServerConnection) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-sc.SendCh:
			if !ok {
				return
			}

			var msgType int
			if sc.Session.EncryptionKey != nil {
				msgType = websocket.BinaryMessage
			} else {
				msgType = websocket.TextMessage
			}

			if err := sc.Conn.WriteMessage(msgType, data); err != nil {
				slog.Error("ws write error", "server", sc.Address, "error", err)
				sc.Close()
				return
			}
		}
	}
}

func (sc *ServerConnection) handleMessage(msg RawMessage) {
	// If this is a response to a pending request, dispatch it.
	if msg.ID != "" {
		sc.mu.Lock()
		ch, ok := sc.pendingReqs[msg.ID]
		sc.mu.Unlock()

		if ok {
			select {
			case ch <- msg:
			default:
			}
			return
		}
	}

	// Event messages: forward to Wails frontend.
	if len(msg.Type) > 6 && msg.Type[:6] == "event." {
		sc.forwardEvent(msg)
		return
	}

	slog.Debug("unhandled ws message", "type", msg.Type, "server", sc.Address)
}

func (sc *ServerConnection) forwardEvent(msg RawMessage) {
	if sc.emitFunc == nil {
		return
	}

	eventName := mapEventToWails(msg.Type)

	var payload interface{}
	if msg.Payload != nil {
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			slog.Warn("event unmarshal error", "event", msg.Type, "error", err)
			return
		}
	}

	// Inject serverID into payload for frontend routing.
	if payloadMap, ok := payload.(map[string]interface{}); ok {
		payloadMap["serverID"] = sc.ServerID
		sc.emitFunc(sc.wailsCtx, eventName, payloadMap)
	} else {
		sc.emitFunc(sc.wailsCtx, eventName, map[string]interface{}{
			"serverID": sc.ServerID,
			"data":     payload,
		})
	}
}

// mapEventToWails converts a WS event type to a Wails event name.
// "event.message.new" -> "message:new"
func mapEventToWails(wsType string) string {
	// Strip "event." prefix and replace dots with colons.
	if len(wsType) <= 6 {
		return wsType
	}
	stripped := wsType[6:]
	result := make([]byte, len(stripped))
	for i, b := range stripped {
		if b == '.' {
			result[i] = ':'
		} else {
			result[i] = byte(b)
		}
	}
	return string(result)
}
