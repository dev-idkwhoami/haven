package auth

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"haven/server/config"
	servercrypto "haven/server/crypto"
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

// authHelloPayload is the Step 2 server hello.
type authHelloPayload struct {
	ServerPubKey    string `json:"server_pubkey"`
	ServerNonce     string `json:"server_nonce"`
	ServerSignature string `json:"server_signature"`
	ChallengeNonce  string `json:"challenge_nonce"`
	AccessMode      string `json:"access_mode"`
	ServerName      string `json:"server_name"`
	ServerVersion   string `json:"server_version"`
}

// authRespondPayload is the Step 3 client response.
type authRespondPayload struct {
	ClientPubKey string         `json:"client_pubkey"`
	Signature    string         `json:"signature"`
	AccessToken  *string        `json:"access_token"`
	SessionToken *string        `json:"session_token"`
	Profile      *clientProfile `json:"profile"`
}

type clientProfile struct {
	DisplayName string `json:"display_name"`
	AvatarHash  string `json:"avatar_hash"`
	Bio         string `json:"bio"`
}

// authSuccessPayload is the Step 5a success response.
type authSuccessPayload struct {
	SessionToken       string      `json:"session_token"`
	UserID             string      `json:"user_id"`
	IsOwner            bool        `json:"is_owner"`
	EncryptionRequired bool        `json:"encryption_required"`
	ServerInfo         *serverInfo `json:"server_info"`
}

type serverInfo struct {
	MaxFileSize       int64       `json:"max_file_size"`
	TotalStorageLimit int64       `json:"total_storage_limit"`
	RateLimits        *rateLimits `json:"rate_limits"`
}

type rateLimits struct {
	MessagesPerSecond int `json:"messages_per_second"`
	MessageBurst      int `json:"message_burst"`
	ConcurrentUploads int `json:"concurrent_uploads"`
}

// HandleNewConnection performs the full 6-step authentication handshake for a new WebSocket connection.
func HandleNewConnection(
	conn *websocket.Conn,
	hub *ws.Hub,
	db *gorm.DB,
	cfg *config.Config,
	hot *config.HotConfig,
	serverKey ed25519.PrivateKey,
	isTLS bool,
	waitingRoom *WaitingRoom,
) {
	// Step 2: Send auth.hello
	serverNonce, err := servercrypto.GenerateNonce()
	if err != nil {
		slog.Error("generate server nonce", "error", err)
		sendAuthError(conn, shared.ErrInternal, "internal error")
		conn.Close()
		return
	}
	challengeNonce, err := servercrypto.GenerateNonce()
	if err != nil {
		slog.Error("generate challenge nonce", "error", err)
		sendAuthError(conn, shared.ErrInternal, "internal error")
		conn.Close()
		return
	}

	serverPub := serverKey.Public().(ed25519.PublicKey)
	serverSig := servercrypto.SignNonce(serverKey, serverNonce)

	// Fetch server metadata from DB
	var srv models.Server
	if err := db.First(&srv).Error; err != nil {
		slog.Error("load server record", "error", err)
		sendAuthError(conn, shared.ErrInternal, "internal error")
		conn.Close()
		return
	}

	// Resolve effective access mode: config override takes precedence over DB.
	accessMode := srv.AccessMode
	if override := hot.AccessMode(); override != "" {
		accessMode = override
	}

	hello := ws.WSMessage{
		Type: shared.TypeAuthHello,
	}
	helloPayload := authHelloPayload{
		ServerPubKey:    hex.EncodeToString(serverPub),
		ServerNonce:     hex.EncodeToString(serverNonce),
		ServerSignature: hex.EncodeToString(serverSig),
		ChallengeNonce:  hex.EncodeToString(challengeNonce),
		AccessMode:      accessMode,
		ServerName:      srv.Name,
		ServerVersion:   shared.Version,
	}
	payloadBytes, _ := json.Marshal(helloPayload)
	hello.Payload = payloadBytes

	helloBytes, _ := json.Marshal(hello)
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, helloBytes); err != nil {
		slog.Debug("send auth.hello failed", "error", err)
		conn.Close()
		return
	}

	// Step 3: Receive auth.respond
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		slog.Debug("read auth.respond failed", "error", err)
		conn.Close()
		return
	}

	var respondMsg ws.WSMessage
	if err := json.Unmarshal(raw, &respondMsg); err != nil {
		sendAuthError(conn, shared.ErrBadRequest, "invalid JSON")
		conn.Close()
		return
	}
	if respondMsg.Type != shared.TypeAuthRespond {
		sendAuthError(conn, shared.ErrBadRequest, "expected auth.respond")
		conn.Close()
		return
	}

	var respond authRespondPayload
	if err := json.Unmarshal(respondMsg.Payload, &respond); err != nil {
		sendAuthError(conn, shared.ErrBadRequest, "invalid auth.respond payload")
		conn.Close()
		return
	}

	// Step 4: Server verification
	clientPubKey, err := hex.DecodeString(respond.ClientPubKey)
	if err != nil || len(clientPubKey) != ed25519.PublicKeySize {
		sendAuthError(conn, shared.ErrInvalidSignature, "invalid client public key")
		conn.Close()
		return
	}

	clientPubKeyHex := hex.EncodeToString(clientPubKey)
	slog.Debug("auth: client identified", "pubkey", clientPubKeyHex, "access_mode", accessMode)

	// Fast path: session token reconnection
	if respond.SessionToken != nil && *respond.SessionToken != "" {
		userID, err := ValidateSession(db, *respond.SessionToken)
		if err != nil {
			sendAuthError(conn, shared.ErrSessionExpired, "session expired or invalid")
			conn.Close()
			return
		}

		// Verify the pubkey matches the session user
		var user models.User
		if err := db.First(&user, "id = ?", userID).Error; err != nil {
			sendAuthError(conn, shared.ErrInternal, "internal error")
			conn.Close()
			return
		}
		if !bytes.Equal(user.PublicKey, clientPubKey) {
			sendAuthError(conn, shared.ErrInvalidSignature, "public key mismatch")
			conn.Close()
			return
		}

		// Allowlist gate for session reconnect — allowlist is ongoing access control.
		slog.Debug("auth: session reconnect allowlist check",
			"pubkey", clientPubKeyHex,
			"access_mode", accessMode,
			"is_owner", hot.IsOwner(clientPubKey),
			"is_allowlisted", hot.IsAllowlisted(clientPubKey),
			"has_approved_request", hasApprovedAccessRequest(db, clientPubKey))
		if accessMode == shared.AccessModeAllowlist &&
			!hot.IsOwner(clientPubKey) && !hot.IsAllowlisted(clientPubKey) &&
			!hasApprovedAccessRequest(db, clientPubKey) {
			if hot.AllowAccessRequests() {
				handleWaitingRoom(conn, db, hot, hub, clientPubKey, &respond, waitingRoom)
				return
			}
			slog.Info("auth: rejecting session reconnect — not on allowlist", "pubkey", clientPubKeyHex)
			sendAuthError(conn, shared.ErrNotAllowlisted, "you are not on the allowlist")
			conn.Close()
			return
		}

		slog.Debug("auth: session reconnect allowed", "pubkey", clientPubKeyHex)

		// Update profile if provided
		if respond.Profile != nil {
			updateUserProfile(db, &user, respond.Profile)
		}

		// Send auth.success and set up client
		finishAuth(conn, hub, hot, serverKey, &user, *respond.SessionToken, serverNonce, challengeNonce, clientPubKey, &srv, isTLS)
		return
	}

	// Verify client signature: signs (challenge_nonce || server_pubkey)
	signedData := make([]byte, 0, len(challengeNonce)+len(serverPub))
	signedData = append(signedData, challengeNonce...)
	signedData = append(signedData, serverPub...)

	sig, err := hex.DecodeString(respond.Signature)
	if err != nil {
		sendAuthError(conn, shared.ErrInvalidSignature, "invalid signature encoding")
		conn.Close()
		return
	}
	if !servercrypto.VerifySignature(ed25519.PublicKey(clientPubKey), signedData, sig) {
		sendAuthError(conn, shared.ErrInvalidSignature, "signature verification failed")
		conn.Close()
		return
	}

	// Ban check (owners are exempt)
	if !hot.IsOwner(clientPubKey) {
		var ban models.Ban
		err := db.Where("public_key = ?", clientPubKey).First(&ban).Error
		if err == nil {
			// Ban found — check expiry
			if ban.ExpiresAt == nil || ban.ExpiresAt.After(time.Now()) {
				sendAuthError(conn, shared.ErrBanned, "you are banned from this server")
				conn.Close()
				return
			}
			// Ban expired, clean it up
			db.Delete(&ban)
		}
	}

	// Identity resolution: known or new user
	var user models.User
	err = db.Where("public_key = ?", clientPubKey).First(&user).Error
	isNewUser := err != nil

	slog.Debug("auth: identity resolution", "pubkey", clientPubKeyHex, "is_new_user", isNewUser)

	if isNewUser {
		// Access control gate for new users
		switch accessMode {
		case shared.AccessModeOpen:
			// Allow
		case shared.AccessModeInvite:
			if respond.AccessToken == nil || *respond.AccessToken == "" {
				sendAuthError(conn, shared.ErrInvalidInvite, "invite code required")
				conn.Close()
				return
			}
			if !validateAndConsumeInvite(db, *respond.AccessToken) {
				sendAuthError(conn, shared.ErrInvalidInvite, "invalid or expired invite code")
				conn.Close()
				return
			}
		case shared.AccessModePassword:
			if respond.AccessToken == nil || *respond.AccessToken == "" {
				sendAuthError(conn, shared.ErrInvalidPassword, "password required")
				conn.Close()
				return
			}
			if srv.AccessPassword == nil || bcrypt.CompareHashAndPassword([]byte(*srv.AccessPassword), []byte(*respond.AccessToken)) != nil {
				sendAuthError(conn, shared.ErrInvalidPassword, "incorrect password")
				conn.Close()
				return
			}
		case shared.AccessModeAllowlist:
			if !hot.IsAllowlisted(clientPubKey) && !hot.IsOwner(clientPubKey) &&
				!hasApprovedAccessRequest(db, clientPubKey) {
				if hot.AllowAccessRequests() {
					handleWaitingRoom(conn, db, hot, hub, clientPubKey, &respond, waitingRoom)
					return
				}
				sendAuthError(conn, shared.ErrNotAllowlisted, "you are not on the allowlist")
				conn.Close()
				return
			}
		}

		// Register new user
		displayName := "Anonymous"
		if respond.Profile != nil && respond.Profile.DisplayName != "" {
			displayName = respond.Profile.DisplayName
		}
		bio := ""
		if respond.Profile != nil {
			bio = respond.Profile.Bio
		}

		user = models.User{
			ID:          ulid.Make().String(),
			PublicKey:   clientPubKey,
			DisplayName: displayName,
			Bio:         &bio,
			Status:      shared.StatusOnline,
			Version:     1,
		}
		if respond.Profile != nil && respond.Profile.AvatarHash != "" {
			user.AvatarHash = respond.Profile.AvatarHash
		}
		if err := db.Create(&user).Error; err != nil {
			slog.Error("create user", "error", err)
			sendAuthError(conn, shared.ErrInternal, "internal error")
			conn.Close()
			return
		}

		// Assign default role
		var defaultRole models.Role
		if err := db.Where("is_default = ?", true).First(&defaultRole).Error; err == nil {
			db.Create(&models.UserRole{
				UserID: user.ID,
				RoleID: defaultRole.ID,
			})
		}
	} else {
		// Returning user — allowlist is ongoing access control, must be checked every time.
		slog.Debug("auth: returning user allowlist check",
			"pubkey", clientPubKeyHex,
			"access_mode", accessMode,
			"is_owner", hot.IsOwner(clientPubKey),
			"is_allowlisted", hot.IsAllowlisted(clientPubKey),
			"has_approved_request", hasApprovedAccessRequest(db, clientPubKey))
		if accessMode == shared.AccessModeAllowlist &&
			!hot.IsOwner(clientPubKey) && !hot.IsAllowlisted(clientPubKey) &&
			!hasApprovedAccessRequest(db, clientPubKey) {
			if hot.AllowAccessRequests() {
				handleWaitingRoom(conn, db, hot, hub, clientPubKey, &respond, waitingRoom)
				return
			}
			sendAuthError(conn, shared.ErrNotAllowlisted, "you are not on the allowlist")
			conn.Close()
			return
		}

		// Update profile if changed
		if respond.Profile != nil {
			updateUserProfile(db, &user, respond.Profile)
		}
		// Set online
		db.Model(&user).Updates(map[string]any{"status": shared.StatusOnline})
	}

	// Create session
	token, err := CreateSession(db, user.ID)
	if err != nil {
		slog.Error("create session", "error", err)
		sendAuthError(conn, shared.ErrInternal, "internal error")
		conn.Close()
		return
	}

	finishAuth(conn, hub, hot, serverKey, &user, token, serverNonce, challengeNonce, clientPubKey, &srv, isTLS)
}

func finishAuth(
	conn *websocket.Conn,
	hub *ws.Hub,
	hot *config.HotConfig,
	serverKey ed25519.PrivateKey,
	user *models.User,
	sessionToken string,
	serverNonce, challengeNonce, clientPubKey []byte,
	srv *models.Server,
	isTLS bool,
) {
	rl := hot.RateLimits()
	success := authSuccessPayload{
		SessionToken:       sessionToken,
		UserID:             user.ID,
		IsOwner:            hot.IsOwner(user.PublicKey),
		EncryptionRequired: !isTLS,
		ServerInfo: &serverInfo{
			MaxFileSize:       srv.MaxFileSize,
			TotalStorageLimit: srv.TotalStorageLimit,
			RateLimits: &rateLimits{
				MessagesPerSecond: rl.MessagesPerSecond,
				MessageBurst:      rl.MessageBurst,
				ConcurrentUploads: rl.ConcurrentUploads,
			},
		},
	}

	successMsg := ws.WSMessage{
		Type: shared.TypeAuthSuccess,
	}
	payloadBytes, _ := json.Marshal(success)
	successMsg.Payload = payloadBytes

	msgBytes, _ := json.Marshal(successMsg)
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		slog.Debug("send auth.success failed", "error", err)
		conn.Close()
		return
	}

	// Step 6: App-layer encryption setup for ws:// connections
	client := &ws.Client{
		Conn:         conn,
		SendCh:       make(chan []byte, 256),
		Hub:          hub,
		PubKey:       clientPubKey,
		PubKeyHex:    hex.EncodeToString(clientPubKey),
		UserID:       user.ID,
		SessionToken: sessionToken,
	}

	if !isTLS {
		encKey, err := servercrypto.DeriveSharedKey(serverKey, ed25519.PublicKey(clientPubKey), serverNonce, challengeNonce)
		if err != nil {
			slog.Error("derive encryption key", "error", err)
			conn.Close()
			return
		}
		client.EncKey = encKey
	}

	// Register with hub
	hub.Register <- client

	// Start read/write pumps
	router := hub.Router
	go ws.WritePump(client)
	go ws.ReadPump(client, router)
}

func updateUserProfile(db *gorm.DB, user *models.User, profile *clientProfile) {
	updates := map[string]any{}
	if profile.DisplayName != "" && profile.DisplayName != user.DisplayName {
		updates["display_name"] = profile.DisplayName
	}
	if profile.Bio != "" {
		currentBio := ""
		if user.Bio != nil {
			currentBio = *user.Bio
		}
		if profile.Bio != currentBio {
			updates["bio"] = profile.Bio
		}
	}
	if profile.AvatarHash != "" && profile.AvatarHash != user.AvatarHash {
		updates["avatar_hash"] = profile.AvatarHash
	}
	if len(updates) > 0 {
		updates["version"] = user.Version + 1
		db.Model(user).Updates(updates)
	}
}

func validateAndConsumeInvite(db *gorm.DB, code string) bool {
	code = strings.TrimSpace(code)
	var invite models.InviteCode
	err := db.Where("code = ?", code).First(&invite).Error
	if err != nil {
		return false
	}

	// Check expiry
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return false
	}

	// Check uses
	if invite.UsesLeft != nil && *invite.UsesLeft <= 0 {
		return false
	}

	// Decrement uses
	if invite.UsesLeft != nil {
		newUses := *invite.UsesLeft - 1
		db.Model(&invite).Update("uses_left", newUses)
	}
	return true
}

// hasApprovedAccessRequest checks if there's an approved access request for this public key.
func hasApprovedAccessRequest(db *gorm.DB, pubKey []byte) bool {
	var count int64
	db.Model(&models.AccessRequest{}).Where("public_key = ? AND status = ?", pubKey, "approved").Count(&count)
	return count > 0
}

// handleWaitingRoom manages the waiting room flow for non-allowlisted users.
func handleWaitingRoom(
	conn *websocket.Conn,
	db *gorm.DB,
	hot *config.HotConfig,
	hub *ws.Hub,
	clientPubKey []byte,
	respond *authRespondPayload,
	waitingRoom *WaitingRoom,
) {
	pubKeyHex := hex.EncodeToString(clientPubKey)

	// Send auth.waiting_room to tell client they can request access.
	wrPayload := map[string]interface{}{
		"allow_message": true,
	}
	payloadBytes, _ := json.Marshal(wrPayload)
	wrMsg := ws.WSMessage{
		Type:    shared.TypeAuthWaitingRoom,
		Payload: payloadBytes,
	}
	msgBytes, _ := json.Marshal(wrMsg)
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		slog.Debug("send auth.waiting_room failed", "error", err)
		conn.Close()
		return
	}

	// Read access_request.submit from client.
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		slog.Debug("read access_request.submit failed", "error", err)
		conn.Close()
		return
	}

	var submitMsg ws.WSMessage
	if err := json.Unmarshal(raw, &submitMsg); err != nil || submitMsg.Type != shared.TypeAccessRequestSubmit {
		sendAuthError(conn, shared.ErrBadRequest, "expected access_request.submit")
		conn.Close()
		return
	}

	var submitPayload struct {
		Message *string `json:"message"`
	}
	json.Unmarshal(submitMsg.Payload, &submitPayload)

	// Extract display name from profile.
	displayName := "Anonymous"
	if respond.Profile != nil && respond.Profile.DisplayName != "" {
		displayName = respond.Profile.DisplayName
	}

	// Check for duplicate pending request or create new one.
	var existing models.AccessRequest
	err = db.Where("public_key = ? AND status = ?", clientPubKey, "pending").First(&existing).Error
	requestID := ""
	if err == nil {
		// Existing pending request — reuse it, update display name/message.
		requestID = existing.ID
		updates := map[string]interface{}{"display_name": displayName}
		if submitPayload.Message != nil {
			updates["message"] = *submitPayload.Message
		}
		db.Model(&existing).Updates(updates)
	} else {
		// Create new request.
		requestID = ulid.Make().String()
		ar := models.AccessRequest{
			ID:          requestID,
			PublicKey:   clientPubKey,
			DisplayName: displayName,
			Message:     submitPayload.Message,
			Status:      "pending",
		}
		if err := db.Create(&ar).Error; err != nil {
			slog.Error("create access request", "error", err)
			sendAuthError(conn, shared.ErrInternal, "internal error")
			conn.Close()
			return
		}

		// Broadcast event.access_request.new to admins.
		eventPayload, _ := json.Marshal(map[string]interface{}{
			"id":           requestID,
			"pubkey":       pubKeyHex,
			"display_name": displayName,
			"message":      submitPayload.Message,
			"is_online":    true,
		})
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventAccessRequestNew, json.RawMessage(eventPayload))
		hub.Broadcast(eventBytes)
	}

	// Add waiter to the waiting room.
	waiter := &Waiter{
		Conn:        conn,
		PubKeyHex:   pubKeyHex,
		PubKey:      clientPubKey,
		DisplayName: displayName,
		RequestID:   requestID,
		DoneCh:      make(chan struct{}),
	}
	waitingRoom.Add(waiter)
	defer waitingRoom.Remove(pubKeyHex)

	// Block until approved, rejected, or timeout.
	timeout := hot.RequestTimeout()
	select {
	case <-waiter.DoneCh:
		switch waiter.Result {
		case "approved":
			// Send auth.access_granted.
			grantPayload, _ := json.Marshal(map[string]string{"status": "approved"})
			grantMsg := ws.WSMessage{
				Type:    shared.TypeAuthAccessGranted,
				Payload: grantPayload,
			}
			gBytes, _ := json.Marshal(grantMsg)
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			conn.WriteMessage(websocket.TextMessage, gBytes)
			conn.Close()
		case "rejected":
			// Send auth.access_denied.
			denyPayload, _ := json.Marshal(map[string]string{"status": "rejected"})
			denyMsg := ws.WSMessage{
				Type:    shared.TypeAuthAccessDenied,
				Payload: denyPayload,
			}
			dBytes, _ := json.Marshal(denyMsg)
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			conn.WriteMessage(websocket.TextMessage, dBytes)
			conn.Close()
		}
	case <-time.After(timeout):
		sendAuthError(conn, shared.ErrNotAllowlisted, "access request timed out")
		conn.Close()
	}
}

func sendAuthError(conn *websocket.Conn, code, message string) {
	payload := map[string]string{
		"code":    code,
		"message": message,
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := ws.WSMessage{
		Type:    shared.TypeAuthError,
		Payload: payloadBytes,
	}
	msgBytes, _ := json.Marshal(msg)

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	conn.WriteMessage(websocket.TextMessage, msgBytes)
}
