package services

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"haven/client/connection"
	havenCrypto "haven/client/crypto"
	"haven/shared"
)

// DMConversationInfo is the frontend-facing DM conversation data.
type DMConversationInfo struct {
	ID           string   `json:"id"`
	IsGroup      bool     `json:"isGroup"`
	Label        string   `json:"label"`
	Participants []string `json:"participants"`
	CreatedAt    string   `json:"createdAt"`
	LastActivity string   `json:"lastActivity"`
}

// DMMessageOut is a decrypted DM message for the frontend.
type DMMessageOut struct {
	ID        string `json:"id"`
	ConvID    string `json:"convId"`
	SenderKey string `json:"senderKey"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// DMMessagePage is a paginated list of DM messages.
type DMMessagePage struct {
	Messages []DMMessageOut `json:"messages"`
	HasMore  bool           `json:"hasMore"`
}

// DMService manages DM conversations with transparent E2EE.
type DMService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey
}

// NewDMService creates a new DMService.
func NewDMService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *DMService {
	return &DMService{
		db:      db,
		manager: manager,
		privKey: privKey,
		pubKey:  privKey.Public().(ed25519.PublicKey),
	}
}

// SetContext is called by Wails during startup.
func (d *DMService) SetContext(ctx context.Context) {
	d.ctx = ctx
}

// CreateDM creates a DM conversation. One participant = 1:1, multiple = group.
func (d *DMService) CreateDM(serverID int64, participants []string) (DMConversationInfo, error) {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return DMConversationInfo{}, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeDMCreate, map[string]interface{}{
		"participants": participants,
	})
	if err != nil {
		return DMConversationInfo{}, fmt.Errorf("dm.create: %w", err)
	}

	var result struct {
		ConversationID string `json:"conversation_id"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return DMConversationInfo{}, fmt.Errorf("unmarshal dm.create: %w", err)
	}

	return DMConversationInfo{
		ID:           result.ConversationID,
		IsGroup:      len(participants) > 1,
		Participants: participants,
	}, nil
}

// GetConversations lists all DM conversations for the current user.
func (d *DMService) GetConversations(serverID int64) ([]DMConversationInfo, error) {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeDMList, nil)
	if err != nil {
		return nil, fmt.Errorf("dm.list: %w", err)
	}

	var result struct {
		Conversations []dmConvWS `json:"conversations"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal dm.list: %w", err)
	}

	convs := make([]DMConversationInfo, len(result.Conversations))
	for i, wc := range result.Conversations {
		pubs := make([]string, len(wc.Participants))
		for j, p := range wc.Participants {
			pubs[j] = p.PubKey
		}
		label := wc.Name
		if label == "" && len(wc.Participants) > 0 {
			// For 1:1 DMs, use the other participant's display name.
			for _, p := range wc.Participants {
				if p.PubKey != havenCrypto.HexEncode(d.pubKey) {
					label = p.DisplayName
					break
				}
			}
		}
		convs[i] = DMConversationInfo{
			ID:           wc.ID,
			IsGroup:      wc.IsGroup,
			Label:        label,
			Participants: pubs,
			LastActivity: wc.LastMessageAt,
		}
	}
	return convs, nil
}

// Send encrypts and sends a DM message.
func (d *DMService) Send(serverID int64, convID string, content string) (DMMessageOut, error) {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return DMMessageOut{}, fmt.Errorf("get connection: %w", err)
	}

	// For now, derive a shared key based on conversation participants.
	// In a full implementation, this would use cached group keys for group DMs
	// and X25519 DH for 1:1 DMs. For the service skeleton, we sign + encrypt.
	nonce, err := havenCrypto.RandomNonce(32)
	if err != nil {
		return DMMessageOut{}, fmt.Errorf("generate nonce: %w", err)
	}

	// Build the inner plaintext: sign the content, then bundle content + signature + nonce.
	sig := havenCrypto.Sign(d.privKey, append([]byte(content), nonce...))
	inner := map[string]interface{}{
		"content":   content,
		"signature": havenCrypto.HexEncode(sig),
		"nonce":     havenCrypto.HexEncode(nonce),
	}
	innerBytes, err := json.Marshal(inner)
	if err != nil {
		return DMMessageOut{}, fmt.Errorf("marshal inner: %w", err)
	}

	// Encrypt the inner payload. For 1:1 DMs we derive a shared key from the peer's pubkey.
	// For group DMs we'd use the group key. Here we use a placeholder encryption
	// that will be replaced with proper key management when the full DM key system is wired.
	encrypted, err := havenCrypto.EncryptBlob(d.getDMKey(serverID, convID), innerBytes)
	if err != nil {
		return DMMessageOut{}, fmt.Errorf("encrypt dm: %w", err)
	}

	resp, err := conn.Request(shared.TypeDMSend, map[string]interface{}{
		"conversation_id":   convID,
		"encrypted_payload": havenCrypto.HexEncode(encrypted),
	})
	if err != nil {
		return DMMessageOut{}, fmt.Errorf("dm.send: %w", err)
	}

	var result struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return DMMessageOut{}, fmt.Errorf("unmarshal dm.send: %w", err)
	}

	return DMMessageOut{
		ID:        result.ID,
		ConvID:    convID,
		SenderKey: havenCrypto.HexEncode(d.pubKey),
		Content:   content,
		Timestamp: result.CreatedAt,
	}, nil
}

// GetHistory fetches and decrypts DM message history.
func (d *DMService) GetHistory(serverID int64, convID string, beforeID string, limit int) (DMMessagePage, error) {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return DMMessagePage{}, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{
		"conversation_id": convID,
	}
	if beforeID != "" {
		payload["before"] = beforeID
	}
	if limit > 0 {
		payload["limit"] = limit
	}

	resp, err := conn.Request(shared.TypeDMHistory, payload)
	if err != nil {
		return DMMessagePage{}, fmt.Errorf("dm.history: %w", err)
	}

	var result struct {
		Messages []dmMsgWS `json:"messages"`
		HasMore  bool      `json:"has_more"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return DMMessagePage{}, fmt.Errorf("unmarshal dm.history: %w", err)
	}

	key := d.getDMKey(serverID, convID)
	messages := make([]DMMessageOut, 0, len(result.Messages))
	for _, wm := range result.Messages {
		msg, err := d.decryptDMMessage(wm, convID, key)
		if err != nil {
			slog.Warn("failed to decrypt dm message", "id", wm.ID, "error", err)
			continue
		}
		messages = append(messages, msg)
	}

	return DMMessagePage{
		Messages: messages,
		HasMore:  result.HasMore,
	}, nil
}

// AddMember adds a member to a group DM.
func (d *DMService) AddMember(serverID int64, convID string, pubKey string) error {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeDMAddMember, map[string]interface{}{
		"conversation_id": convID,
		"pubkey":          pubKey,
	})
	if err != nil {
		return fmt.Errorf("dm.add_member: %w", err)
	}
	return nil
}

// RemoveMember removes a member from a group DM.
func (d *DMService) RemoveMember(serverID int64, convID string, pubKey string) error {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeDMRemoveMember, map[string]interface{}{
		"conversation_id": convID,
		"pubkey":          pubKey,
	})
	if err != nil {
		return fmt.Errorf("dm.remove_member: %w", err)
	}
	return nil
}

// LeaveConversation leaves a DM conversation.
func (d *DMService) LeaveConversation(serverID int64, convID string) error {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeDMLeave, map[string]interface{}{
		"conversation_id": convID,
	})
	if err != nil {
		return fmt.Errorf("dm.leave: %w", err)
	}
	return nil
}

// StartCall initiates a voice call in a DM conversation.
func (d *DMService) StartCall(serverID int64, convID string) error {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeDMVoiceStart, map[string]interface{}{
		"conversation_id": convID,
	})
	if err != nil {
		return fmt.Errorf("dm.voice.start: %w", err)
	}
	return nil
}

// AcceptCall accepts an incoming DM voice call.
func (d *DMService) AcceptCall(serverID int64, convID string) error {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeDMVoiceAccept, map[string]interface{}{
		"conversation_id": convID,
	})
	if err != nil {
		return fmt.Errorf("dm.voice.accept: %w", err)
	}
	return nil
}

// RejectCall rejects an incoming DM voice call.
func (d *DMService) RejectCall(serverID int64, convID string) error {
	conn, err := d.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeDMVoiceReject, map[string]interface{}{
		"conversation_id": convID,
	})
	if err != nil {
		return fmt.Errorf("dm.voice.reject: %w", err)
	}
	return nil
}

// LeaveCall leaves an active DM voice call.
func (d *DMService) LeaveCall() error {
	// DM voice calls use the same single-active-session model.
	// The active call's serverID and convID would be tracked internally.
	// For now, this is a placeholder until VoiceService tracks the active DM call.
	return nil
}

// getDMKey returns the encryption key for a DM conversation.
// For 1:1 DMs, derives via X25519 DH. For group DMs, uses the stored group key.
func (d *DMService) getDMKey(serverID int64, convID string) []byte {
	// TODO: Implement proper key management:
	// - 1:1 DMs: DeriveSharedKey(myPrivKey, peerPubKey)
	// - Group DMs: Look up cached group key from dm.key.distribute
	// For now, derive a deterministic key from the conversation ID as a placeholder.
	// This will be replaced when the full DM key distribution system is implemented.
	key := make([]byte, 32)
	convBytes := []byte(convID)
	for i := 0; i < 32 && i < len(convBytes); i++ {
		key[i] = convBytes[i]
	}
	return key
}

// decryptDMMessage decrypts a single DM message from the server.
func (d *DMService) decryptDMMessage(wm dmMsgWS, convID string, key []byte) (DMMessageOut, error) {
	encrypted, err := havenCrypto.HexDecode(wm.EncryptedPayload)
	if err != nil {
		return DMMessageOut{}, fmt.Errorf("decode encrypted payload: %w", err)
	}

	plaintext, err := havenCrypto.DecryptBlob(key, encrypted)
	if err != nil {
		return DMMessageOut{}, fmt.Errorf("decrypt dm message: %w", err)
	}

	var inner struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(plaintext, &inner); err != nil {
		return DMMessageOut{}, fmt.Errorf("unmarshal inner: %w", err)
	}

	return DMMessageOut{
		ID:        wm.ID,
		ConvID:    convID,
		SenderKey: wm.SenderPubKey,
		Content:   inner.Content,
		Timestamp: wm.CreatedAt,
	}, nil
}

// dmConvWS is the wire format for a DM conversation.
type dmConvWS struct {
	ID            string          `json:"id"`
	IsGroup       bool            `json:"is_group"`
	Name          string          `json:"name"`
	Participants  []dmParticipant `json:"participants"`
	LastMessageAt string          `json:"last_message_at"`
}

type dmParticipant struct {
	PubKey       string `json:"pubkey"`
	DisplayName  string `json:"display_name"`
	IsKeyManager bool   `json:"is_key_manager"`
}

// dmMsgWS is the wire format for a DM message.
type dmMsgWS struct {
	ID               string `json:"id"`
	SenderPubKey     string `json:"sender_pubkey"`
	EncryptedPayload string `json:"encrypted_payload"`
	CreatedAt        string `json:"created_at"`
}
