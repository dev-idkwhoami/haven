package ws

import (
	"encoding/hex"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a single connected WebSocket client.
type Client struct {
	Conn         *websocket.Conn
	SendCh       chan []byte
	PubKey       []byte
	PubKeyHex    string
	UserID       string
	SessionToken string

	// EncKey is the ChaCha20-Poly1305 key for ws:// app-layer encryption.
	// Nil when using wss:// (TLS handles encryption).
	EncKey []byte

	// SendNonce and RecvNonce are incrementing counters for app-layer encryption.
	SendNonce uint64
	RecvNonce uint64

	Hub  *Hub
	once sync.Once
}

// Close cleanly shuts down the client connection and unregisters from the hub.
func (c *Client) Close() {
	c.once.Do(func() {
		if c.Hub != nil {
			c.Hub.Unregister <- c
		}
		close(c.SendCh)
		c.Conn.Close()
	})
}

// Send enqueues a message for sending to this client. Returns false if the send channel is full.
func (c *Client) Send(msg []byte) bool {
	select {
	case c.SendCh <- msg:
		return true
	default:
		return false
	}
}

// Hub maintains the set of active clients and coordinates broadcasts.
type Hub struct {
	// Clients maps public key hex -> *Client
	Clients   map[string]*Client
	clientsMu sync.RWMutex

	// UserClients maps userID -> *Client for quick lookup by user ID
	UserClients   map[string]*Client
	userClientsMu sync.RWMutex

	Register   chan *Client
	Unregister chan *Client

	// Router is the message router used for dispatching messages from clients.
	Router *Router

	// ChannelAccess is a callback that checks if a client has access to a channel.
	// Set by the server after hub creation. Returns true if the client can see the channel.
	ChannelAccess func(client *Client, channelID string) bool
}

// NewHub creates and returns a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		Clients:     make(map[string]*Client),
		UserClients: make(map[string]*Client),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
	}
}

// Run starts the hub's event loop, processing register/unregister events.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.clientsMu.Lock()
			h.Clients[client.PubKeyHex] = client
			h.clientsMu.Unlock()

			h.userClientsMu.Lock()
			h.UserClients[client.UserID] = client
			h.userClientsMu.Unlock()

			slog.Info("client registered", "pubkey", client.PubKeyHex, "user_id", client.UserID)

		case client := <-h.Unregister:
			h.clientsMu.Lock()
			if _, ok := h.Clients[client.PubKeyHex]; ok {
				delete(h.Clients, client.PubKeyHex)
			}
			h.clientsMu.Unlock()

			h.userClientsMu.Lock()
			if existing, ok := h.UserClients[client.UserID]; ok && existing == client {
				delete(h.UserClients, client.UserID)
			}
			h.userClientsMu.Unlock()

			slog.Info("client unregistered", "pubkey", client.PubKeyHex, "user_id", client.UserID)
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg []byte) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	for _, client := range h.Clients {
		client.Send(msg)
	}
}

// BroadcastToChannel sends a message to all clients with access to a channel.
func (h *Hub) BroadcastToChannel(channelID string, msg []byte) {
	if h.ChannelAccess == nil {
		h.Broadcast(msg)
		return
	}
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	for _, client := range h.Clients {
		if h.ChannelAccess(client, channelID) {
			client.Send(msg)
		}
	}
}

// SendTo sends a message to a specific client identified by public key hex.
func (h *Hub) SendTo(pubKeyHex string, msg []byte) bool {
	h.clientsMu.RLock()
	client, ok := h.Clients[pubKeyHex]
	h.clientsMu.RUnlock()
	if !ok {
		return false
	}
	return client.Send(msg)
}

// SendToUser sends a message to a specific client identified by user ID.
func (h *Hub) SendToUser(userID string, msg []byte) bool {
	h.userClientsMu.RLock()
	client, ok := h.UserClients[userID]
	h.userClientsMu.RUnlock()
	if !ok {
		return false
	}
	return client.Send(msg)
}

// GetClient returns the client for a given public key hex, or nil.
func (h *Hub) GetClient(pubKeyHex string) *Client {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return h.Clients[pubKeyHex]
}

// GetClientByUserID returns the client for a given user ID, or nil.
func (h *Hub) GetClientByUserID(userID string) *Client {
	h.userClientsMu.RLock()
	defer h.userClientsMu.RUnlock()
	return h.UserClients[userID]
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.Clients)
}

// ForEachClient calls fn for each connected client. The callback must not block.
func (h *Hub) ForEachClient(fn func(c *Client)) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	for _, c := range h.Clients {
		fn(c)
	}
}

// PubKeyHex returns the hex-encoded public key string for a raw public key.
func PubKeyHex(pubKey []byte) string {
	return hex.EncodeToString(pubKey)
}
