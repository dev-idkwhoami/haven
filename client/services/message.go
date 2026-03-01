package services

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"haven/client/connection"
	havenCrypto "haven/client/crypto"
	"haven/shared"
)

// Message is the frontend-facing message data.
type Message struct {
	ID           string   `json:"id"`
	ChannelID    string   `json:"channelId"`
	AuthorPubKey string   `json:"authorPubKey"`
	Content      string   `json:"content"`
	FileIDs      []string `json:"fileIds"`
	EditedAt     string   `json:"editedAt"`
	CreatedAt    string   `json:"createdAt"`
}

// MessageSearchParams are Discord-style search filters.
type MessageSearchParams struct {
	Text       string `json:"text"`
	ChannelID  string `json:"channelId"`
	FromPubKey string `json:"fromPubKey"`
	Has        string `json:"has"`
	Before     string `json:"before"`
	After      string `json:"after"`
}

// MessagePage is a paginated list of messages.
type MessagePage struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}

// SearchResult is a search result with total count.
type SearchResult struct {
	Messages   []Message `json:"messages"`
	TotalCount int       `json:"totalCount"`
}

// MessageService manages message operations with internal signing.
type MessageService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey
}

// NewMessageService creates a new MessageService.
func NewMessageService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *MessageService {
	return &MessageService{
		db:      db,
		manager: manager,
		privKey: privKey,
		pubKey:  privKey.Public().(ed25519.PublicKey),
	}
}

// SetContext is called by Wails during startup.
func (m *MessageService) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// Send sends a signed message to a channel.
func (m *MessageService) Send(serverID int64, channelID string, content string, fileIDs []string) (Message, error) {
	conn, err := m.manager.Get(serverID)
	if err != nil {
		return Message{}, fmt.Errorf("get connection: %w", err)
	}

	// Generate nonce and sign the message internally.
	nonce, err := havenCrypto.RandomNonce(32)
	if err != nil {
		return Message{}, fmt.Errorf("generate nonce: %w", err)
	}

	timestamp := time.Now().Unix()
	sig := havenCrypto.SignMessage(m.privKey, content, channelID, timestamp, nonce)

	payload := map[string]interface{}{
		"channel_id": channelID,
		"content":    content,
		"signature":  havenCrypto.HexEncode(sig),
		"nonce":      havenCrypto.HexEncode(nonce),
	}
	if len(fileIDs) > 0 {
		payload["file_ids"] = fileIDs
	}

	resp, err := conn.Request(shared.TypeMessageSend, payload)
	if err != nil {
		return Message{}, fmt.Errorf("message.send: %w", err)
	}

	var result struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
		Version   int64  `json:"version"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return Message{}, fmt.Errorf("unmarshal send result: %w", err)
	}

	fids := fileIDs
	if fids == nil {
		fids = []string{}
	}

	return Message{
		ID:           result.ID,
		ChannelID:    channelID,
		AuthorPubKey: havenCrypto.HexEncode(m.pubKey),
		Content:      content,
		FileIDs:      fids,
		CreatedAt:    result.CreatedAt,
	}, nil
}

// Edit edits an existing message with a new signature.
func (m *MessageService) Edit(serverID int64, messageID string, content string) error {
	conn, err := m.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	nonce, err := havenCrypto.RandomNonce(32)
	if err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	// For edits we sign with empty channelID since the server already knows the channel.
	// The server uses the message's existing channel_id for verification.
	timestamp := time.Now().Unix()
	sig := havenCrypto.SignMessage(m.privKey, content, "", timestamp, nonce)

	_, err = conn.Request(shared.TypeMessageEdit, map[string]interface{}{
		"id":        messageID,
		"content":   content,
		"signature": havenCrypto.HexEncode(sig),
		"nonce":     havenCrypto.HexEncode(nonce),
	})
	if err != nil {
		return fmt.Errorf("message.edit: %w", err)
	}
	return nil
}

// Delete deletes a message.
func (m *MessageService) Delete(serverID int64, messageID string) error {
	conn, err := m.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeMessageDelete, map[string]interface{}{
		"id": messageID,
	})
	if err != nil {
		return fmt.Errorf("message.delete: %w", err)
	}
	return nil
}

// GetHistory fetches message history with cursor-based pagination.
func (m *MessageService) GetHistory(serverID int64, channelID string, beforeID string, limit int) (MessagePage, error) {
	conn, err := m.manager.Get(serverID)
	if err != nil {
		return MessagePage{}, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{
		"channel_id": channelID,
	}
	if beforeID != "" {
		payload["before"] = beforeID
	}
	if limit > 0 {
		payload["limit"] = limit
	}

	resp, err := conn.Request(shared.TypeMessageHistory, payload)
	if err != nil {
		return MessagePage{}, fmt.Errorf("message.history: %w", err)
	}

	var result struct {
		Messages []messageWS `json:"messages"`
		HasMore  bool        `json:"has_more"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return MessagePage{}, fmt.Errorf("unmarshal history: %w", err)
	}

	messages := make([]Message, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = msg.toMessage()
	}

	return MessagePage{
		Messages: messages,
		HasMore:  result.HasMore,
	}, nil
}

// Search searches messages with Discord-style filters.
func (m *MessageService) Search(serverID int64, params MessageSearchParams) (SearchResult, error) {
	conn, err := m.manager.Get(serverID)
	if err != nil {
		return SearchResult{}, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{}
	if params.Text != "" {
		payload["text"] = params.Text
	}
	if params.ChannelID != "" {
		payload["channel_id"] = params.ChannelID
	}
	if params.FromPubKey != "" {
		payload["from_pubkey"] = params.FromPubKey
	}
	if params.Has != "" {
		payload["has"] = params.Has
	}
	if params.Before != "" {
		payload["before"] = params.Before
	}
	if params.After != "" {
		payload["after"] = params.After
	}

	resp, err := conn.Request(shared.TypeMessageSearch, payload)
	if err != nil {
		return SearchResult{}, fmt.Errorf("message.search: %w", err)
	}

	var result struct {
		Messages   []messageWS `json:"messages"`
		TotalCount int         `json:"total_count"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return SearchResult{}, fmt.Errorf("unmarshal search: %w", err)
	}

	messages := make([]Message, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = msg.toMessage()
	}

	return SearchResult{
		Messages:   messages,
		TotalCount: result.TotalCount,
	}, nil
}

// messageWS is the wire format from the server.
type messageWS struct {
	ID           string   `json:"id"`
	ChannelID    string   `json:"channel_id"`
	AuthorPubKey string   `json:"author_pubkey"`
	Content      string   `json:"content"`
	Signature    string   `json:"signature"`
	Nonce        string   `json:"nonce"`
	FileIDs      []string `json:"file_ids"`
	EditedAt     string   `json:"edited_at"`
	CreatedAt    string   `json:"created_at"`
	Version      int64    `json:"version"`
}

func (msg messageWS) toMessage() Message {
	fileIDs := msg.FileIDs
	if fileIDs == nil {
		fileIDs = []string{}
	}
	return Message{
		ID:           msg.ID,
		ChannelID:    msg.ChannelID,
		AuthorPubKey: msg.AuthorPubKey,
		Content:      msg.Content,
		FileIDs:      fileIDs,
		EditedAt:     msg.EditedAt,
		CreatedAt:    msg.CreatedAt,
	}
}
