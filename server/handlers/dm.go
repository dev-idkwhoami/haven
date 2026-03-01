package handlers

import (
	"encoding/hex"
	"time"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterDMHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeDMCreate, handleDMCreate(d))
	router.Register(shared.TypeDMList, handleDMList(d))
	router.Register(shared.TypeDMSend, handleDMSend(d))
	router.Register(shared.TypeDMHistory, handleDMHistory(d))
	router.Register(shared.TypeDMAddMember, handleDMAddMember(d))
	router.Register(shared.TypeDMRemoveMember, handleDMRemoveMember(d))
	router.Register(shared.TypeDMLeave, handleDMLeave(d))
	router.Register(shared.TypeDMKeyDistrib, handleDMKeyDistribute(d))
}

func handleDMCreate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			Participants []string `json:"participants"`
			Name         *string  `json:"name"`
		}
		if !parsePayload(msg, &req) || len(req.Participants) == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "participants are required")
			return
		}

		// Resolve all participant user IDs
		participantUserIDs := []string{client.UserID}
		for _, pkHex := range req.Participants {
			user, err := getUserByPubKeyHex(d.DB, pkHex)
			if err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "participant not found: "+pkHex)
				return
			}
			participantUserIDs = append(participantUserIDs, user.ID)
		}

		isGroup := len(req.Participants) > 1

		// For 1:1 DMs, check if a conversation already exists
		if !isGroup {
			existingID := findExisting1on1(d, client.UserID, participantUserIDs[1])
			if existingID != "" {
				ws.SendOK(client, msg.Type, msg.ID, map[string]any{
					"conversation_id": existingID,
				})
				return
			}
		}

		now := time.Now()
		conv := models.DMConversation{
			ID:        newULID(),
			IsGroup:   isGroup,
			Name:      req.Name,
			CreatedBy: &client.UserID,
		}
		if err := d.DB.Create(&conv).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create conversation")
			return
		}

		// Add all participants
		for i, uid := range participantUserIDs {
			p := models.DMParticipant{
				ConversationID: conv.ID,
				UserID:         uid,
				IsKeyManager:   i == 0, // creator is key manager
				JoinedAt:       now,
			}
			d.DB.Create(&p)
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"conversation_id": conv.ID,
		})
	}
}

func handleDMList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		// Find all conversations where this user is an active participant
		var participations []models.DMParticipant
		d.DB.Where("user_id = ? AND left_at IS NULL", client.UserID).Find(&participations)

		convIDs := make([]string, len(participations))
		for i, p := range participations {
			convIDs[i] = p.ConversationID
		}

		if len(convIDs) == 0 {
			ws.SendOK(client, msg.Type, msg.ID, map[string]any{
				"conversations": []any{},
			})
			return
		}

		var conversations []models.DMConversation
		d.DB.Where("id IN ?", convIDs).Find(&conversations)

		out := make([]map[string]any, len(conversations))
		for i, conv := range conversations {
			// Get active participants
			var parts []models.DMParticipant
			d.DB.Where("conversation_id = ? AND left_at IS NULL", conv.ID).Find(&parts)

			participantList := make([]map[string]any, 0, len(parts))
			for _, p := range parts {
				var user models.User
				if err := d.DB.First(&user, "id = ?", p.UserID).Error; err != nil {
					continue
				}
				participantList = append(participantList, map[string]any{
					"pubkey":         hex.EncodeToString(user.PublicKey),
					"display_name":   user.DisplayName,
					"is_key_manager": p.IsKeyManager,
				})
			}

			entry := map[string]any{
				"id":           conv.ID,
				"is_group":     conv.IsGroup,
				"participants": participantList,
			}
			if conv.Name != nil {
				entry["name"] = *conv.Name
			}

			// Get last message timestamp
			var lastMsg models.DMMessage
			if err := d.DB.Where("conversation_id = ?", conv.ID).Order("id DESC").First(&lastMsg).Error; err == nil {
				entry["last_message_at"] = lastMsg.CreatedAt
			}

			out[i] = entry
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"conversations": out,
		})
	}
}

func handleDMSend(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID   string `json:"conversation_id"`
			EncryptedPayload string `json:"encrypted_payload"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" || req.EncryptedPayload == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id and encrypted_payload are required")
			return
		}

		// Verify sender is a participant
		if !isDMParticipant(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "not a participant")
			return
		}

		payload, err := hex.DecodeString(req.EncryptedPayload)
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid encrypted_payload hex")
			return
		}

		dmMsg := models.DMMessage{
			ID:               newULID(),
			ConversationID:   req.ConversationID,
			SenderID:         client.UserID,
			EncryptedPayload: payload,
		}
		if err := d.DB.Create(&dmMsg).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to send message")
			return
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"id":         dmMsg.ID,
			"created_at": dmMsg.CreatedAt,
		})

		// Broadcast to all online participants
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMNew, map[string]any{
			"conversation_id":   req.ConversationID,
			"id":                dmMsg.ID,
			"sender_pubkey":     client.PubKeyHex,
			"encrypted_payload": req.EncryptedPayload,
			"created_at":        dmMsg.CreatedAt,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)
	}
}

func handleDMHistory(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
			Before         string `json:"before"`
			Limit          int    `json:"limit"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id is required")
			return
		}

		if !isDMParticipant(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "not a participant")
			return
		}

		if req.Limit <= 0 || req.Limit > 100 {
			req.Limit = 50
		}

		query := d.DB.Where("conversation_id = ?", req.ConversationID).Order("id DESC").Limit(req.Limit + 1)
		if req.Before != "" {
			query = query.Where("id < ?", req.Before)
		}

		var messages []models.DMMessage
		query.Find(&messages)

		hasMore := len(messages) > req.Limit
		if hasMore {
			messages = messages[:req.Limit]
		}

		out := make([]map[string]any, len(messages))
		for i, m := range messages {
			senderPubKey := getAuthorPubKey(d, m.SenderID)
			out[i] = map[string]any{
				"id":                m.ID,
				"sender_pubkey":     hex.EncodeToString(senderPubKey),
				"encrypted_payload": hex.EncodeToString(m.EncryptedPayload),
				"created_at":        m.CreatedAt,
			}
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"messages": out,
			"has_more": hasMore,
		})
	}
}

func handleDMAddMember(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
			PubKey         string `json:"pubkey"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" || req.PubKey == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id and pubkey are required")
			return
		}

		// Verify caller is key manager
		if !isDMKeyManager(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "only key manager can add members")
			return
		}

		target, err := getUserByPubKeyHex(d.DB, req.PubKey)
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "user not found")
			return
		}

		// Check if already a participant
		var existing models.DMParticipant
		if err := d.DB.Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", req.ConversationID, target.ID).First(&existing).Error; err == nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "user is already a participant")
			return
		}

		p := models.DMParticipant{
			ConversationID: req.ConversationID,
			UserID:         target.ID,
			JoinedAt:       time.Now(),
		}
		d.DB.Create(&p)

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMMemberAdded, map[string]any{
			"conversation_id": req.ConversationID,
			"pubkey":          req.PubKey,
			"display_name":    target.DisplayName,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)
	}
}

func handleDMRemoveMember(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
			PubKey         string `json:"pubkey"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" || req.PubKey == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id and pubkey are required")
			return
		}

		// Verify caller is key manager
		if !isDMKeyManager(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "only key manager can remove members")
			return
		}

		target, err := getUserByPubKeyHex(d.DB, req.PubKey)
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "user not found")
			return
		}

		now := time.Now()
		d.DB.Model(&models.DMParticipant{}).
			Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", req.ConversationID, target.ID).
			Update("left_at", now)

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMMemberRemoved, map[string]any{
			"conversation_id": req.ConversationID,
			"pubkey":          req.PubKey,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)
	}
}

func handleDMLeave(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id is required")
			return
		}

		if !isDMParticipant(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "not a participant")
			return
		}

		now := time.Now()
		d.DB.Model(&models.DMParticipant{}).
			Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", req.ConversationID, client.UserID).
			Update("left_at", now)

		ws.SendOK(client, msg.Type, msg.ID, nil)

		eventBytes, _ := ws.MarshalEvent(shared.TypeEventDMMemberRemoved, map[string]any{
			"conversation_id": req.ConversationID,
			"pubkey":          client.PubKeyHex,
		})
		broadcastToDMParticipants(d, req.ConversationID, eventBytes)
	}
}

func handleDMKeyDistribute(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			ConversationID string `json:"conversation_id"`
			Recipients     []struct {
				PubKey       string `json:"pubkey"`
				EncryptedKey string `json:"encrypted_key"`
			} `json:"recipients"`
		}
		if !parsePayload(msg, &req) || req.ConversationID == "" || len(req.Recipients) == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "conversation_id and recipients are required")
			return
		}

		if !isDMParticipant(d, client.UserID, req.ConversationID) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "not a participant")
			return
		}

		// Relay encrypted keys to each recipient
		for _, r := range req.Recipients {
			eventBytes, _ := ws.MarshalEvent("event.dm.key.distributed", map[string]any{
				"conversation_id": req.ConversationID,
				"from_pubkey":     client.PubKeyHex,
				"encrypted_key":   r.EncryptedKey,
			})
			d.Hub.SendTo(r.PubKey, eventBytes)
		}

		ws.SendOK(client, msg.Type, msg.ID, nil)
	}
}

// isDMParticipant checks if a user is an active participant in a DM conversation.
func isDMParticipant(d *Deps, userID, conversationID string) bool {
	var count int64
	d.DB.Model(&models.DMParticipant{}).
		Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", conversationID, userID).
		Count(&count)
	return count > 0
}

// isDMKeyManager checks if a user is the key manager for a DM conversation.
func isDMKeyManager(d *Deps, userID, conversationID string) bool {
	var p models.DMParticipant
	err := d.DB.Where("conversation_id = ? AND user_id = ? AND left_at IS NULL AND is_key_manager = ?",
		conversationID, userID, true).First(&p).Error
	return err == nil
}

// findExisting1on1 checks if a 1:1 DM already exists between two users.
func findExisting1on1(d *Deps, userID1, userID2 string) string {
	// Find conversations where both users are active participants and it's not a group
	var convIDs []string
	d.DB.Model(&models.DMParticipant{}).
		Select("conversation_id").
		Where("user_id = ? AND left_at IS NULL", userID1).
		Pluck("conversation_id", &convIDs)

	for _, cid := range convIDs {
		var conv models.DMConversation
		if err := d.DB.First(&conv, "id = ? AND is_group = ?", cid, false).Error; err != nil {
			continue
		}
		var count int64
		d.DB.Model(&models.DMParticipant{}).
			Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", cid, userID2).
			Count(&count)
		if count > 0 {
			return cid
		}
	}
	return ""
}

// broadcastToDMParticipants sends a message to all online participants of a DM conversation.
func broadcastToDMParticipants(d *Deps, conversationID string, eventBytes []byte) {
	var parts []models.DMParticipant
	d.DB.Where("conversation_id = ? AND left_at IS NULL", conversationID).Find(&parts)

	for _, p := range parts {
		d.Hub.SendToUser(p.UserID, eventBytes)
	}
}
