package services

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"haven/client/connection"
	havenCrypto "haven/client/crypto"
	"haven/client/models"
	"haven/shared"
)

// ErrWaitingRoom is returned by TrustAndAuth when the server requires an access request.
var ErrWaitingRoom = errors.New("waiting_room")

// ServerEntry is the frontend-facing server list item.
type ServerEntry struct {
	ID              int64  `json:"id"`
	Address         string `json:"address"`
	Name            string `json:"name"`
	IconHash        string `json:"iconHash"`
	IsRelayOnly     bool   `json:"isRelayOnly"`
	IsOwner         bool   `json:"isOwner"`
	Connected       bool   `json:"connected"`
	LastConnectedAt string `json:"lastConnectedAt"`
}

// ServerHello is returned during the connect flow so the frontend knows what to render.
type ServerHello struct {
	ServerPubKey string `json:"serverPubKey"`
	ServerName   string `json:"serverName"`
	AccessMode   string `json:"accessMode"`
	TrustStatus  string `json:"trustStatus"`  // "new" | "trusted" | "mismatch"
	StoredPubKey string `json:"storedPubKey"` // hex, only if mismatch
}

// ServerInfo is the detailed server metadata.
type ServerInfo struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IconID            string `json:"iconId"`
	IconHash          string `json:"iconHash"`
	AccessMode        string `json:"accessMode"`
	MemberCount       int    `json:"memberCount"`
	MaxFileSize       int64  `json:"maxFileSize"`
	TotalStorageLimit int64  `json:"totalStorageLimit"`
}

// authHelloPayload is the server's auth.hello payload.
type authHelloPayload struct {
	ServerPubKey   string `json:"server_pubkey"`
	ServerNonce    string `json:"server_nonce"`
	ServerSig      string `json:"server_signature"`
	ChallengeNonce string `json:"challenge_nonce"`
	AccessMode     string `json:"access_mode"`
	ServerName     string `json:"server_name"`
	ServerVersion  string `json:"server_version"`
}

// authSuccessPayload is the server's auth.success payload.
type authSuccessPayload struct {
	SessionToken       string         `json:"session_token"`
	UserID             string         `json:"user_id"`
	IsOwner            bool           `json:"is_owner"`
	EncryptionRequired bool           `json:"encryption_required"`
	ServerInfo         authServerInfo `json:"server_info"`
}

type authServerInfo struct {
	MaxFileSize       int64          `json:"max_file_size"`
	TotalStorageLimit int64          `json:"total_storage_limit"`
	RateLimits        authRateLimits `json:"rate_limits"`
}

type authRateLimits struct {
	MessagesPerSecond int `json:"messages_per_second"`
	MessageBurst      int `json:"message_burst"`
	ConcurrentUploads int `json:"concurrent_uploads"`
}

// pendingAuth holds state for an in-progress connection handshake.
type pendingAuth struct {
	sc             *connection.ServerConnection
	hello          authHelloPayload
	challengeNonce []byte
	serverPubKey   []byte
	serverID       int64
	isNewServer    bool
}

// ServerService manages server connections, TOFU trust, and server metadata.
type ServerService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey

	pending           *pendingAuth
	pendingAccessConn *websocket.Conn

	reconnectMu      sync.Mutex
	reconnectCancels map[int64]context.CancelFunc
	reconnectWake    map[int64]chan struct{}

	focusedServer atomic.Int64
}

// NewServerService creates a new ServerService.
func NewServerService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *ServerService {
	return &ServerService{
		db:               db,
		manager:          manager,
		privKey:          privKey,
		pubKey:           privKey.Public().(ed25519.PublicKey),
		reconnectCancels: make(map[int64]context.CancelFunc),
		reconnectWake:    make(map[int64]chan struct{}),
	}
}

// SetContext is called by Wails during startup.
func (s *ServerService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetServers returns all non-relay trusted servers.
func (s *ServerService) GetServers() []ServerEntry {
	var servers []models.TrustedServer
	s.db.Where("is_relay_only = ?", false).Find(&servers)
	return s.toEntries(servers)
}

// GetRelayServers returns all relay-only trusted servers.
func (s *ServerService) GetRelayServers() []ServerEntry {
	var servers []models.TrustedServer
	s.db.Where("is_relay_only = ?", true).Find(&servers)
	return s.toEntries(servers)
}

// Connect initiates a connection to a server address and returns the hello info
// so the frontend can render trust prompts and access mode fields.
func (s *ServerService) Connect(address string) (ServerHello, error) {
	wsURL := buildWSURL(address)

	slog.Info("connecting to server", "address", address, "url", wsURL)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return ServerHello{}, fmt.Errorf("dial server: %w", err)
	}

	// Read auth.hello from server.
	_, data, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return ServerHello{}, fmt.Errorf("read auth.hello: %w", err)
	}

	var msg connection.RawMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		conn.Close()
		return ServerHello{}, fmt.Errorf("unmarshal auth.hello: %w", err)
	}

	if msg.Type != shared.TypeAuthHello {
		conn.Close()
		return ServerHello{}, fmt.Errorf("expected auth.hello, got %s", msg.Type)
	}

	var hello authHelloPayload
	if err := json.Unmarshal(msg.Payload, &hello); err != nil {
		conn.Close()
		return ServerHello{}, fmt.Errorf("unmarshal hello payload: %w", err)
	}

	// Verify server identity.
	serverPubKey, err := havenCrypto.HexDecode(hello.ServerPubKey)
	if err != nil {
		conn.Close()
		return ServerHello{}, fmt.Errorf("decode server pubkey: %w", err)
	}

	serverNonce, err := havenCrypto.HexDecode(hello.ServerNonce)
	if err != nil {
		conn.Close()
		return ServerHello{}, fmt.Errorf("decode server nonce: %w", err)
	}

	serverSig, err := havenCrypto.HexDecode(hello.ServerSig)
	if err != nil {
		conn.Close()
		return ServerHello{}, fmt.Errorf("decode server sig: %w", err)
	}

	if !havenCrypto.Verify(ed25519.PublicKey(serverPubKey), serverNonce, serverSig) {
		conn.Close()
		return ServerHello{}, fmt.Errorf("server signature verification failed")
	}

	challengeNonce, err := havenCrypto.HexDecode(hello.ChallengeNonce)
	if err != nil {
		conn.Close()
		return ServerHello{}, fmt.Errorf("decode challenge nonce: %w", err)
	}

	// TOFU check.
	var trusted models.TrustedServer
	trustStatus := "new"
	var serverID int64
	isNewServer := true

	err = s.db.Where("address = ?", address).First(&trusted).Error
	if err == nil {
		isNewServer = false
		serverID = trusted.ID
		if bytes.Equal(trusted.PublicKey, serverPubKey) {
			trustStatus = "trusted"
		} else {
			trustStatus = "mismatch"
		}
	}

	// Create a ServerConnection but don't start read/write loops yet.
	sc := connection.NewServerConnection(serverID, address, s.ctx, emitFunc)
	sc.Conn = conn

	s.pending = &pendingAuth{
		sc:             sc,
		hello:          hello,
		challengeNonce: challengeNonce,
		serverPubKey:   serverPubKey,
		serverID:       serverID,
		isNewServer:    isNewServer,
	}

	result := ServerHello{
		ServerPubKey: hello.ServerPubKey,
		ServerName:   hello.ServerName,
		AccessMode:   hello.AccessMode,
		TrustStatus:  trustStatus,
	}
	if trustStatus == "mismatch" {
		result.StoredPubKey = havenCrypto.HexEncode(trusted.PublicKey)
	}

	return result, nil
}

// TrustAndAuth completes the auth handshake after the user approves trust.
func (s *ServerService) TrustAndAuth(accessToken string) error {
	if s.pending == nil {
		return fmt.Errorf("no pending connection")
	}
	pa := s.pending
	s.pending = nil

	// Sign the challenge: sign(challenge_nonce || server_pubkey).
	sig := havenCrypto.SignChallenge(s.privKey, pa.challengeNonce, pa.serverPubKey)

	// Get local profile for the handshake.
	var lp models.LocalProfile
	s.db.First(&lp)

	bio := ""
	if lp.Bio != nil {
		bio = *lp.Bio
	}
	avatarHash := ""
	if lp.AvatarHash != nil {
		avatarHash = *lp.AvatarHash
	}

	// Build session token from stored server if reconnecting.
	var sessionToken string
	if !pa.isNewServer {
		var ts models.TrustedServer
		if err := s.db.First(&ts, pa.serverID).Error; err == nil && ts.SessionToken != nil {
			sessionToken = *ts.SessionToken
		}
	}

	// Send auth.respond.
	respondPayload := map[string]interface{}{
		"client_pubkey": havenCrypto.HexEncode(s.pubKey),
		"signature":     havenCrypto.HexEncode(sig),
		"access_token":  nilIfEmpty(accessToken),
		"session_token": nilIfEmpty(sessionToken),
		"profile": map[string]interface{}{
			"display_name": lp.DisplayName,
			"avatar_hash":  avatarHash,
			"bio":          bio,
		},
	}

	respondData, err := json.Marshal(connection.RawMessage{
		Type:    shared.TypeAuthRespond,
		Payload: mustMarshal(respondPayload),
	})
	if err != nil {
		pa.sc.Conn.Close()
		return fmt.Errorf("marshal auth.respond: %w", err)
	}

	if err := pa.sc.Conn.WriteMessage(websocket.TextMessage, respondData); err != nil {
		pa.sc.Conn.Close()
		return fmt.Errorf("send auth.respond: %w", err)
	}

	// Read server response.
	slog.Debug("TrustAndAuth: reading server response")
	_, data, err := pa.sc.Conn.ReadMessage()
	if err != nil {
		slog.Debug("TrustAndAuth: read failed", "error", err)
		pa.sc.Conn.Close()
		return fmt.Errorf("read auth response: %w", err)
	}

	var msg connection.RawMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		pa.sc.Conn.Close()
		return fmt.Errorf("unmarshal auth response: %w", err)
	}

	slog.Debug("TrustAndAuth: received message", "type", msg.Type)

	if msg.Type == shared.TypeAuthError {
		pa.sc.Conn.Close()
		var errPayload connection.ErrorPayload
		json.Unmarshal(msg.Payload, &errPayload)
		slog.Debug("TrustAndAuth: auth error", "code", errPayload.Code, "message", errPayload.Message)
		return fmt.Errorf("auth rejected: %s — %s", errPayload.Code, errPayload.Message)
	}

	if msg.Type == shared.TypeAuthWaitingRoom {
		slog.Debug("TrustAndAuth: received waiting room, returning ErrWaitingRoom")
		s.pendingAccessConn = pa.sc.Conn
		return ErrWaitingRoom
	}

	if msg.Type != shared.TypeAuthSuccess {
		pa.sc.Conn.Close()
		return fmt.Errorf("unexpected auth response: %s", msg.Type)
	}

	var success authSuccessPayload
	if err := json.Unmarshal(msg.Payload, &success); err != nil {
		pa.sc.Conn.Close()
		return fmt.Errorf("unmarshal auth.success: %w", err)
	}

	// Store/update trusted server in DB.
	now := time.Now()
	if pa.isNewServer {
		ts := models.TrustedServer{
			Address:         pa.sc.Address,
			PublicKey:       pa.serverPubKey,
			Name:            &pa.hello.ServerName,
			SessionToken:    &success.SessionToken,
			FirstTrustedAt:  now,
			LastConnectedAt: &now,
		}
		if err := s.db.Create(&ts).Error; err != nil {
			pa.sc.Conn.Close()
			return fmt.Errorf("store trusted server: %w", err)
		}
		pa.sc.ServerID = ts.ID
	} else {
		s.db.Model(&models.TrustedServer{}).Where("id = ?", pa.serverID).Updates(map[string]interface{}{
			"session_token":     success.SessionToken,
			"last_connected_at": now,
			"name":              pa.hello.ServerName,
			"public_key":        pa.serverPubKey,
		})
		pa.sc.ServerID = pa.serverID
	}

	// Set up session state.
	pa.sc.Session = connection.SessionState{
		Token:  success.SessionToken,
		UserID: success.UserID,
	}

	// App-layer encryption for ws:// connections.
	if success.EncryptionRequired {
		serverNonce, _ := havenCrypto.HexDecode(pa.hello.ServerNonce)
		key, err := havenCrypto.DeriveWSEncryptionKey(s.privKey, ed25519.PublicKey(pa.serverPubKey), serverNonce, pa.challengeNonce)
		if err != nil {
			pa.sc.Conn.Close()
			return fmt.Errorf("derive encryption key: %w", err)
		}
		pa.sc.Session.EncryptionKey = key
	}

	// Set up auto-reconnect on unexpected disconnect.
	serverID := pa.sc.ServerID
	pa.sc.OnUnexpectedClose = func(sid int64) {
		s.autoReconnect(sid)
	}

	// Cancel any in-progress reconnect for this server (we just connected).
	s.cancelReconnect(serverID)

	// Register and start the connection.
	s.manager.Register(pa.sc)
	pa.sc.SetConn(pa.sc.Conn)

	slog.Info("connected to server", "serverID", pa.sc.ServerID, "address", pa.sc.Address)

	// Emit connected event.
	entry := s.toEntry(pa.sc.ServerID, pa.sc.Address, pa.hello.ServerName, "", false, success.IsOwner, true, &now)
	emitEvent(s.ctx, "server:connected", entry)

	return nil
}

// RejectTrust aborts a pending connection (user cancelled trust or saw key mismatch).
func (s *ServerService) RejectTrust() {
	if s.pending != nil {
		s.pending.sc.Conn.Close()
		s.pending = nil
	}
}

// Disconnect disconnects from a specific server.
func (s *ServerService) Disconnect(serverID int64) error {
	s.cancelReconnect(serverID)
	return s.manager.Disconnect(serverID)
}

// Reconnect reconnects to a previously trusted server.
func (s *ServerService) Reconnect(serverID int64) error {
	slog.Debug("Reconnect: starting", "serverID", serverID)
	var ts models.TrustedServer
	if err := s.db.First(&ts, serverID).Error; err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	hello, err := s.Connect(ts.Address)
	if err != nil {
		return fmt.Errorf("reconnect: %w", err)
	}

	if hello.TrustStatus == "mismatch" {
		s.RejectTrust()
		return fmt.Errorf("server key mismatch on reconnect")
	}

	slog.Debug("Reconnect: trust ok, calling TrustAndAuth")

	// For trusted reconnect, auto-authenticate.
	return s.TrustAndAuth("")
}

// LeaveServer leaves a server with the specified departure mode.
func (s *ServerService) LeaveServer(serverID int64, mode string) error {
	s.cancelReconnect(serverID)
	conn, err := s.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeUserLeave, map[string]interface{}{
		"mode": mode,
	})
	if err != nil {
		return fmt.Errorf("leave server: %w", err)
	}

	// Clean up local data.
	s.db.Where("id = ?", serverID).Delete(&models.TrustedServer{})
	s.db.Where("server_id = ?", serverID).Delete(&models.PerServerConfig{})
	s.db.Where("server_id = ?", serverID).Delete(&models.CachedUser{})
	s.db.Where("server_id = ?", serverID).Delete(&models.CachedMessage{})
	s.db.Where("server_id = ?", serverID).Delete(&models.CachedCategory{})
	s.db.Where("server_id = ?", serverID).Delete(&models.CachedChannel{})

	s.manager.Disconnect(serverID)
	return nil
}

// RemoveRelay removes a relay-only server.
func (s *ServerService) RemoveRelay(serverID int64) error {
	s.db.Where("id = ? AND is_relay_only = ?", serverID, true).Delete(&models.TrustedServer{})
	s.manager.Disconnect(serverID)
	return nil
}

// GetServerInfo fetches server metadata via the WS connection.
func (s *ServerService) GetServerInfo(serverID int64) (ServerInfo, error) {
	conn, err := s.manager.Get(serverID)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeServerInfo, nil)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("server.info: %w", err)
	}

	var info ServerInfo
	if err := json.Unmarshal(resp.Payload, &info); err != nil {
		return ServerInfo{}, fmt.Errorf("unmarshal server info: %w", err)
	}

	return info, nil
}

func (s *ServerService) toEntries(servers []models.TrustedServer) []ServerEntry {
	entries := make([]ServerEntry, 0, len(servers))
	for _, ts := range servers {
		name := ""
		if ts.Name != nil {
			name = *ts.Name
		}
		iconHash := ""
		if ts.IconHash != nil {
			iconHash = *ts.IconHash
		}
		lastConn := ""
		if ts.LastConnectedAt != nil {
			lastConn = ts.LastConnectedAt.Format(time.RFC3339)
		}

		sc := s.manager.GetOrNil(ts.ID)
		connected := sc != nil && sc.Connected

		entries = append(entries, ServerEntry{
			ID:              ts.ID,
			Address:         ts.Address,
			Name:            name,
			IconHash:        iconHash,
			IsRelayOnly:     ts.IsRelayOnly,
			Connected:       connected,
			LastConnectedAt: lastConn,
		})
	}
	return entries
}

func (s *ServerService) toEntry(id int64, address, name, iconHash string, isRelay, isOwner, connected bool, lastConn *time.Time) ServerEntry {
	lc := ""
	if lastConn != nil {
		lc = lastConn.Format(time.RFC3339)
	}
	return ServerEntry{
		ID:              id,
		Address:         address,
		Name:            name,
		IconHash:        iconHash,
		IsRelayOnly:     isRelay,
		IsOwner:         isOwner,
		Connected:       connected,
		LastConnectedAt: lc,
	}
}

// buildWSURL converts a user-typed address to a WebSocket URL.
// Domain → wss://, IP → ws://, appends /ws.
func buildWSURL(address string) string {
	address = strings.TrimSpace(address)

	// Already has a scheme.
	if strings.HasPrefix(address, "ws://") || strings.HasPrefix(address, "wss://") {
		if !strings.HasSuffix(address, "/ws") {
			address += "/ws"
		}
		return address
	}

	// Determine scheme: IP addresses get ws://, domains get wss://.
	host := address
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		host = host[:idx]
	}

	scheme := "wss://"
	if isIPAddress(host) {
		scheme = "ws://"
	}

	result := scheme + address
	if !strings.HasSuffix(result, "/ws") {
		result += "/ws"
	}
	return result
}

func isIPAddress(host string) bool {
	// Simple heuristic: if it starts with a digit or contains only digits and dots, it's an IP.
	if len(host) == 0 {
		return false
	}
	if host == "localhost" {
		return true
	}
	if host[0] == '[' {
		return true // IPv6
	}
	for _, ch := range host {
		if ch != '.' && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

// autoReconnect attempts to reconnect to a server with exponential backoff.
// When the user is actively viewing the server, retries are more aggressive (2s cap)
// and switching to the server mid-sleep wakes the loop immediately.
func (s *ServerService) autoReconnect(serverID int64) {
	s.reconnectMu.Lock()
	// If there's already a reconnect in progress for this server, skip.
	if _, ok := s.reconnectCancels[serverID]; ok {
		s.reconnectMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.reconnectCancels[serverID] = cancel
	s.reconnectMu.Unlock()

	defer func() {
		s.reconnectMu.Lock()
		delete(s.reconnectCancels, serverID)
		delete(s.reconnectWake, serverID)
		s.reconnectMu.Unlock()
	}()

	backoff := time.Second
	const maxBackoff = 10 * time.Second
	const focusedMaxBackoff = 2 * time.Second

	for attempt := 1; ; attempt++ {
		// Determine actual delay: if user is watching this server, cap at 2s.
		focused := s.focusedServer.Load() == serverID
		delay := backoff
		if focused && delay > focusedMaxBackoff {
			delay = focusedMaxBackoff
		}

		slog.Info("auto-reconnecting", "serverID", serverID, "attempt", attempt, "delay", delay, "focused", focused)

		// Emit reconnecting event so the frontend can show status.
		if s.ctx != nil {
			emitEvent(s.ctx, "server:reconnecting", map[string]interface{}{
				"serverID": serverID,
				"attempt":  attempt,
			})
		}

		// Create a wake channel so SetFocusedServer can interrupt the sleep.
		wake := make(chan struct{})
		s.reconnectMu.Lock()
		s.reconnectWake[serverID] = wake
		s.reconnectMu.Unlock()

		select {
		case <-ctx.Done():
			slog.Info("auto-reconnect cancelled", "serverID", serverID)
			return
		case <-time.After(delay):
		case <-wake:
			// User just switched to this server — retry immediately.
		}

		if err := s.Reconnect(serverID); err != nil {
			if errors.Is(err, ErrWaitingRoom) {
				slog.Info("server requires access request, stopping auto-reconnect", "serverID", serverID)
				emitEvent(s.ctx, "server:access_required", map[string]interface{}{
					"serverID": serverID,
				})
				return
			}
			slog.Warn("reconnect failed", "serverID", serverID, "attempt", attempt, "error", err)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		slog.Info("reconnected successfully", "serverID", serverID, "attempt", attempt)
		return
	}
}

// cancelReconnect cancels any in-progress reconnect goroutine for a server.
func (s *ServerService) cancelReconnect(serverID int64) {
	s.reconnectMu.Lock()
	if cancel, ok := s.reconnectCancels[serverID]; ok {
		cancel()
		delete(s.reconnectCancels, serverID)
	}
	s.reconnectMu.Unlock()
}

// SetFocusedServer tells the reconnect system which server the user is actively viewing.
// If that server is currently reconnecting and sleeping, it wakes up immediately.
func (s *ServerService) SetFocusedServer(serverID int64) {
	s.focusedServer.Store(serverID)
	s.reconnectMu.Lock()
	if wake, ok := s.reconnectWake[serverID]; ok {
		close(wake)
		delete(s.reconnectWake, serverID)
	}
	s.reconnectMu.Unlock()
}

// SubmitAccessRequest sends an access request on the pending waiting room connection.
func (s *ServerService) SubmitAccessRequest(message string) error {
	if s.pendingAccessConn == nil {
		return fmt.Errorf("no pending access request connection")
	}

	payload := map[string]interface{}{}
	if message != "" {
		payload["message"] = message
	}

	submitData, err := json.Marshal(connection.RawMessage{
		Type:    shared.TypeAccessRequestSubmit,
		Payload: mustMarshal(payload),
	})
	if err != nil {
		return fmt.Errorf("marshal access_request.submit: %w", err)
	}

	if err := s.pendingAccessConn.WriteMessage(websocket.TextMessage, submitData); err != nil {
		s.pendingAccessConn.Close()
		s.pendingAccessConn = nil
		return fmt.Errorf("send access_request.submit: %w", err)
	}

	// Start goroutine to wait for the decision.
	go s.waitForAccessDecision()
	return nil
}

// waitForAccessDecision reads from the pending connection waiting for approval/rejection.
func (s *ServerService) waitForAccessDecision() {
	conn := s.pendingAccessConn
	if conn == nil {
		return
	}
	defer func() {
		conn.Close()
		s.pendingAccessConn = nil
	}()

	// Set a generous read deadline (server timeout + buffer).
	conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	_, data, err := conn.ReadMessage()
	if err != nil {
		emitEvent(s.ctx, "access_request:timeout", nil)
		return
	}

	var msg connection.RawMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		emitEvent(s.ctx, "access_request:timeout", nil)
		return
	}

	switch msg.Type {
	case shared.TypeAuthAccessGranted:
		emitEvent(s.ctx, "access_request:approved", nil)
	case shared.TypeAuthAccessDenied:
		emitEvent(s.ctx, "access_request:rejected", nil)
	case shared.TypeAuthError:
		emitEvent(s.ctx, "access_request:timeout", nil)
	default:
		emitEvent(s.ctx, "access_request:timeout", nil)
	}
}

// CancelAccessRequest cancels a pending access request by closing the connection.
func (s *ServerService) CancelAccessRequest() {
	if s.pendingAccessConn != nil {
		s.pendingAccessConn.Close()
		s.pendingAccessConn = nil
	}
}
