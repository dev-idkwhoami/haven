package handlers

import (
	"encoding/hex"
	"strings"
	"time"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterMessageHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeMessageSend, handleMessageSend(d))
	router.Register(shared.TypeMessageEdit, handleMessageEdit(d))
	router.Register(shared.TypeMessageDelete, handleMessageDelete(d))
	router.Register(shared.TypeMessageHistory, handleMessageHistory(d))
	router.Register(shared.TypeMessageSearch, handleMessageSearch(d))
	router.Register(shared.TypeMessageTyping, handleMessageTyping(d))
	router.Register(shared.TypeMessageRead, handleMessageRead(d))
}

func handleMessageSend(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermSendMessages) {
			return
		}

		if !d.RateLimiter.AllowMessage(client.PubKeyHex) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrRateLimited, "rate limited")
			return
		}

		var req struct {
			ChannelID string   `json:"channel_id"`
			Content   string   `json:"content"`
			Signature string   `json:"signature"`
			Nonce     string   `json:"nonce"`
			FileIDs   []string `json:"file_ids"`
		}
		if !parsePayload(msg, &req) || req.ChannelID == "" || req.Content == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "channel_id and content are required")
			return
		}

		// Verify channel access
		if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, req.ChannelID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "no access to channel")
			return
		}

		// Check attach files permission if file_ids present
		if len(req.FileIDs) > 0 {
			if !checkPerm(d, client, msg.Type, msg.ID, shared.PermAttachFiles) {
				return
			}
		}

		sig, _ := hex.DecodeString(req.Signature)
		nonce, _ := hex.DecodeString(req.Nonce)

		message := models.Message{
			ID:        newULID(),
			ChannelID: req.ChannelID,
			AuthorID:  client.UserID,
			Content:   req.Content,
			Signature: sig,
			Nonce:     nonce,
			Version:   1,
		}
		if err := d.DB.Create(&message).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create message")
			return
		}

		// Link file attachments
		for _, fileID := range req.FileIDs {
			d.DB.Create(&models.MessageFile{
				MessageID: message.ID,
				FileID:    fileID,
			})
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"id":         message.ID,
			"created_at": message.CreatedAt,
			"version":    message.Version,
		})

		// Broadcast to channel
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventMessageNew, map[string]any{
			"channel_id": req.ChannelID,
			"message":    messageToMap(&message, client.PubKey, req.FileIDs),
		})
		d.Hub.BroadcastToChannel(req.ChannelID, eventBytes)
	}
}

func handleMessageEdit(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ID        string `json:"id"`
			Content   string `json:"content"`
			Signature string `json:"signature"`
			Nonce     string `json:"nonce"`
		}
		if !parsePayload(msg, &req) || req.ID == "" || req.Content == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id and content are required")
			return
		}

		var message models.Message
		if err := d.DB.First(&message, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "message not found")
			return
		}

		// Only author can edit
		if message.AuthorID != client.UserID {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "can only edit own messages")
			return
		}

		sig, _ := hex.DecodeString(req.Signature)
		nonce, _ := hex.DecodeString(req.Nonce)
		now := time.Now()

		message.Version++
		if err := d.DB.Model(&message).Updates(map[string]any{
			"content":   req.Content,
			"signature": sig,
			"nonce":     nonce,
			"edited_at": now,
			"version":   message.Version,
		}).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to edit message")
			return
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"version":   message.Version,
			"edited_at": now,
		})

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventMessageEdited, map[string]any{
			"channel_id": message.ChannelID,
			"id":         message.ID,
			"content":    req.Content,
			"signature":  req.Signature,
			"nonce":      req.Nonce,
			"edited_at":  now,
			"version":    message.Version,
		})
		d.Hub.BroadcastToChannel(message.ChannelID, eventBytes)
	}
}

func handleMessageDelete(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ID string `json:"id"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var message models.Message
		if err := d.DB.First(&message, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "message not found")
			return
		}

		// Author can delete own; ManageMessages can delete anyone's
		isAuthor := message.AuthorID == client.UserID
		if !isAuthor {
			if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageMessages) {
				return
			}
			// Mod deleting another's message — audit log
			auditLog(d.DB, client.UserID, shared.AuditMessageDelete, shared.TargetTypeMessage, message.ID, map[string]any{
				"channel_id": message.ChannelID,
				"author_id":  message.AuthorID,
			})
		}

		// Delete file links
		d.DB.Where("message_id = ?", message.ID).Delete(&models.MessageFile{})
		d.DB.Delete(&message)

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventMessageDeleted, map[string]any{
			"channel_id": message.ChannelID,
			"id":         message.ID,
		})
		d.Hub.BroadcastToChannel(message.ChannelID, eventBytes)
	}
}

func handleMessageHistory(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ChannelID string `json:"channel_id"`
			Before    string `json:"before"`
			Limit     int    `json:"limit"`
		}
		if !parsePayload(msg, &req) || req.ChannelID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "channel_id is required")
			return
		}

		if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, req.ChannelID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "no access to channel")
			return
		}

		if req.Limit <= 0 || req.Limit > 100 {
			req.Limit = 50
		}

		query := d.DB.Where("channel_id = ?", req.ChannelID).Order("id DESC").Limit(req.Limit + 1)
		if req.Before != "" {
			query = query.Where("id < ?", req.Before)
		}

		var messages []models.Message
		query.Find(&messages)

		hasMore := len(messages) > req.Limit
		if hasMore {
			messages = messages[:req.Limit]
		}

		out := make([]map[string]any, len(messages))
		for i, m := range messages {
			fileIDs := getMessageFileIDs(d, m.ID)
			authorPubKey := getAuthorPubKey(d, m.AuthorID)
			out[i] = messageToMap(&m, authorPubKey, fileIDs)
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"messages": out,
			"has_more": hasMore,
		})
	}
}

func handleMessageSearch(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			Text       *string  `json:"text"`
			ChannelID  *string  `json:"channel_id"`
			FromPubKey *string  `json:"from_pubkey"`
			Has        []string `json:"has"`
			Before     *string  `json:"before"`
			After      *string  `json:"after"`
			Limit      int      `json:"limit"`
		}
		parsePayload(msg, &req)

		if req.Limit <= 0 || req.Limit > 50 {
			req.Limit = 25
		}

		query := d.DB.Model(&models.Message{})

		if req.Text != nil && *req.Text != "" {
			query = query.Where("content LIKE ?", "%"+*req.Text+"%")
		}
		if req.ChannelID != nil && *req.ChannelID != "" {
			// Check access
			if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, *req.ChannelID) {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "no access to channel")
				return
			}
			query = query.Where("channel_id = ?", *req.ChannelID)
		}
		if req.FromPubKey != nil && *req.FromPubKey != "" {
			user, err := getUserByPubKeyHex(d.DB, *req.FromPubKey)
			if err == nil {
				query = query.Where("author_id = ?", user.ID)
			}
		}
		if req.Before != nil && *req.Before != "" {
			t, err := time.Parse(time.RFC3339, *req.Before)
			if err == nil {
				query = query.Where("created_at < ?", t)
			}
		}
		if req.After != nil && *req.After != "" {
			t, err := time.Parse(time.RFC3339, *req.After)
			if err == nil {
				query = query.Where("created_at > ?", t)
			}
		}

		// Filter by "has" (file, image, link)
		for _, h := range req.Has {
			switch strings.ToLower(h) {
			case "file":
				query = query.Where("id IN (SELECT message_id FROM message_files)")
			case "image":
				query = query.Where("id IN (SELECT mf.message_id FROM message_files mf JOIN files f ON mf.file_id = f.id WHERE f.mime_type LIKE 'image/%')")
			case "link":
				query = query.Where("content LIKE '%http://%' OR content LIKE '%https://%'")
			}
		}

		var totalCount int64
		query.Count(&totalCount)

		var messages []models.Message
		query.Order("id DESC").Limit(req.Limit).Find(&messages)

		out := make([]map[string]any, len(messages))
		for i, m := range messages {
			fileIDs := getMessageFileIDs(d, m.ID)
			authorPubKey := getAuthorPubKey(d, m.AuthorID)
			out[i] = messageToMap(&m, authorPubKey, fileIDs)
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"messages":    out,
			"total_count": totalCount,
		})
	}
}

func handleMessageTyping(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		// Fire-and-forget — no response
		var req struct {
			ChannelID string `json:"channel_id"`
		}
		if !parsePayload(msg, &req) || req.ChannelID == "" {
			return
		}

		if !d.RateLimiter.AllowMessage("typing:" + client.PubKeyHex) {
			return // silently dropped
		}

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventMessageTyping, map[string]any{
			"channel_id": req.ChannelID,
			"pubkey":     client.PubKeyHex,
		})
		d.Hub.BroadcastToChannel(req.ChannelID, eventBytes)
	}
}

func handleMessageRead(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		// No response — watermark-based
		var req struct {
			ChannelID  string `json:"channel_id"`
			LastReadID string `json:"last_read_id"`
		}
		if !parsePayload(msg, &req) || req.ChannelID == "" || req.LastReadID == "" {
			return
		}

		// Broadcast read receipt to channel
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventMessageRead, map[string]any{
			"channel_id":   req.ChannelID,
			"pubkey":       client.PubKeyHex,
			"last_read_id": req.LastReadID,
		})
		d.Hub.BroadcastToChannel(req.ChannelID, eventBytes)
	}
}

func messageToMap(m *models.Message, authorPubKey []byte, fileIDs []string) map[string]any {
	result := map[string]any{
		"id":            m.ID,
		"channel_id":    m.ChannelID,
		"author_pubkey": hex.EncodeToString(authorPubKey),
		"content":       m.Content,
		"signature":     hex.EncodeToString(m.Signature),
		"nonce":         hex.EncodeToString(m.Nonce),
		"file_ids":      fileIDs,
		"created_at":    m.CreatedAt,
		"version":       m.Version,
	}
	if m.EditedAt != nil {
		result["edited_at"] = *m.EditedAt
	}
	return result
}

func getMessageFileIDs(d *Deps, messageID string) []string {
	var mfs []models.MessageFile
	d.DB.Where("message_id = ?", messageID).Find(&mfs)
	ids := make([]string, len(mfs))
	for i, mf := range mfs {
		ids[i] = mf.FileID
	}
	return ids
}

func getAuthorPubKey(d *Deps, authorID string) []byte {
	var user models.User
	if err := d.DB.Select("public_key").First(&user, "id = ?", authorID).Error; err != nil {
		return make([]byte, 32) // fallback to zero key (sentinel)
	}
	return user.PublicKey
}
