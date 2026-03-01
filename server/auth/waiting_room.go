package auth

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Waiter represents a client waiting for access approval.
type Waiter struct {
	Conn        *websocket.Conn
	PubKeyHex   string
	PubKey      []byte
	DisplayName string
	RequestID   string
	DoneCh      chan struct{}
	Result      string // "approved" | "rejected"
}

// WaitingRoom manages clients waiting for access approval in allowlist mode.
type WaitingRoom struct {
	mu      sync.RWMutex
	waiters map[string]*Waiter // pubkey hex → waiter
}

// NewWaitingRoom creates a new WaitingRoom.
func NewWaitingRoom() *WaitingRoom {
	return &WaitingRoom{
		waiters: make(map[string]*Waiter),
	}
}

// Add registers a waiter. If a waiter with the same pubkey already exists,
// the old one is closed (duplicate connection).
func (wr *WaitingRoom) Add(w *Waiter) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	if existing, ok := wr.waiters[w.PubKeyHex]; ok {
		existing.Result = "rejected"
		close(existing.DoneCh)
	}
	wr.waiters[w.PubKeyHex] = w
}

// Remove removes a waiter by pubkey hex.
func (wr *WaitingRoom) Remove(hex string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	delete(wr.waiters, hex)
}

// Get returns a waiter by pubkey hex, or nil if not found.
func (wr *WaitingRoom) Get(hex string) *Waiter {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return wr.waiters[hex]
}

// Approve approves a waiter and signals their DoneCh. Returns true if found.
func (wr *WaitingRoom) Approve(hex string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	w, ok := wr.waiters[hex]
	if !ok {
		return false
	}
	w.Result = "approved"
	close(w.DoneCh)
	delete(wr.waiters, hex)
	return true
}

// Reject rejects a waiter and signals their DoneCh. Returns true if found.
func (wr *WaitingRoom) Reject(hex string) bool {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	w, ok := wr.waiters[hex]
	if !ok {
		return false
	}
	w.Result = "rejected"
	close(w.DoneCh)
	delete(wr.waiters, hex)
	return true
}

// Count returns the number of waiting clients.
func (wr *WaitingRoom) Count() int {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return len(wr.waiters)
}

// IsOnline checks if a pubkey hex is currently in the waiting room.
func (wr *WaitingRoom) IsOnline(hex string) bool {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	_, ok := wr.waiters[hex]
	return ok
}
