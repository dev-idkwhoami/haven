package connection

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Manager manages multiple simultaneous server connections.
type Manager struct {
	mu          sync.RWMutex
	connections map[int64]*ServerConnection // keyed by TrustedServer.ID
	wailsCtx    context.Context
	emitFunc    func(ctx context.Context, eventName string, data ...interface{})
}

// NewManager creates a new connection manager.
func NewManager() *Manager {
	return &Manager{
		connections: make(map[int64]*ServerConnection),
	}
}

// SetWailsContext sets the Wails runtime context and emit function.
// Called during app startup after Wails context is available.
func (m *Manager) SetWailsContext(ctx context.Context, emitFunc func(ctx context.Context, eventName string, data ...interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wailsCtx = ctx
	m.emitFunc = emitFunc
}

// Get returns an existing connection by server ID.
func (m *Manager) Get(serverID int64) (*ServerConnection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[serverID]
	if !ok {
		return nil, fmt.Errorf("no connection for server %d", serverID)
	}

	if !conn.Connected {
		return nil, fmt.Errorf("server %d is disconnected", serverID)
	}

	return conn, nil
}

// GetOrNil returns the connection for serverID, or nil if not found.
func (m *Manager) GetOrNil(serverID int64) *ServerConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[serverID]
}

// Register adds a new ServerConnection to the manager.
func (m *Manager) Register(sc *ServerConnection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, ok := m.connections[sc.ServerID]; ok {
		old.MarkIntentionalClose()
		old.Close()
	}
	m.connections[sc.ServerID] = sc
}

// CreateConnection creates a new ServerConnection and registers it.
func (m *Manager) CreateConnection(serverID int64, address string) *ServerConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, ok := m.connections[serverID]; ok {
		old.MarkIntentionalClose()
		old.Close()
	}

	sc := NewServerConnection(serverID, address, m.wailsCtx, m.emitFunc)
	m.connections[serverID] = sc
	return sc
}

// Disconnect closes and removes a specific server connection.
func (m *Manager) Disconnect(serverID int64) error {
	m.mu.Lock()
	conn, ok := m.connections[serverID]
	if ok {
		delete(m.connections, serverID)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("no connection for server %d", serverID)
	}

	conn.MarkIntentionalClose()
	conn.Close()
	slog.Info("disconnected from server", "serverID", serverID, "address", conn.Address)
	return nil
}

// DisconnectAll closes all server connections.
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	conns := make(map[int64]*ServerConnection, len(m.connections))
	for k, v := range m.connections {
		conns[k] = v
	}
	m.connections = make(map[int64]*ServerConnection)
	m.mu.Unlock()

	for _, conn := range conns {
		conn.MarkIntentionalClose()
		conn.Close()
	}

	slog.Info("disconnected from all servers", "count", len(conns))
}

// ConnectedCount returns the number of active connections.
func (m *Manager) ConnectedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, conn := range m.connections {
		if conn.Connected {
			count++
		}
	}
	return count
}

// AllConnected returns all connected server IDs.
func (m *Manager) AllConnected() []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var ids []int64
	for id, conn := range m.connections {
		if conn.Connected {
			ids = append(ids, id)
		}
	}
	return ids
}
