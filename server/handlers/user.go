package handlers

import (
	"encoding/hex"
	"log/slog"
	"os"
	"time"

	"haven/server/auth"
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"

	"gorm.io/gorm"
)

func RegisterUserHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeUserProfile, handleUserProfile(d))
	router.Register(shared.TypeUserUpdate, handleUserUpdate(d))
	router.Register(shared.TypeUserList, handleUserList(d))
	router.Register(shared.TypeUserKick, handleUserKick(d))
	router.Register(shared.TypeUserLeave, handleUserLeave(d))
}

func handleUserProfile(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			PubKey *string `json:"pubkey"`
		}
		parsePayload(msg, &req)

		var user models.User
		if req.PubKey != nil && *req.PubKey != "" {
			pubKey, err := hex.DecodeString(*req.PubKey)
			if err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid pubkey")
				return
			}
			if err := d.DB.Where("public_key = ?", pubKey).First(&user).Error; err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "user not found")
				return
			}
		} else {
			if err := d.DB.First(&user, "id = ?", client.UserID).Error; err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to load profile")
				return
			}
		}

		roles := getUserRoleIDs(d.DB, user.ID)

		ws.SendOK(client, msg.Type, msg.ID, userProfileMap(&user, roles))
	}
}

func handleUserUpdate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			DisplayName *string `json:"display_name"`
			AvatarID    *string `json:"avatar_id"`
			Bio         *string `json:"bio"`
			Status      *string `json:"status"`
		}
		if !parsePayload(msg, &req) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid payload")
			return
		}

		var user models.User
		if err := d.DB.First(&user, "id = ?", client.UserID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to load user")
			return
		}

		updates := map[string]any{}
		eventFields := map[string]any{
			"pubkey": client.PubKeyHex,
		}

		if req.DisplayName != nil {
			updates["display_name"] = *req.DisplayName
			eventFields["display_name"] = *req.DisplayName
		}
		if req.AvatarID != nil {
			updates["avatar"] = *req.AvatarID
			eventFields["avatar_id"] = *req.AvatarID
		}
		if req.Bio != nil {
			updates["bio"] = *req.Bio
			eventFields["bio"] = *req.Bio
		}
		if req.Status != nil {
			updates["status"] = *req.Status
			eventFields["status"] = *req.Status
		}

		if len(updates) == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "no fields to update")
			return
		}

		user.Version++
		updates["version"] = user.Version
		if err := d.DB.Model(&user).Updates(updates).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to update user")
			return
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"version": user.Version,
		})

		eventFields["version"] = user.Version
		broadcastEvent(d, shared.TypeEventUserUpdated, eventFields)
	}
}

func handleUserList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var users []models.User
		d.DB.Where("id != ?", shared.SentinelUserID).Find(&users)

		out := make([]map[string]any, len(users))
		for i, u := range users {
			roles := getUserRoleIDs(d.DB, u.ID)
			out[i] = map[string]any{
				"pubkey":       hex.EncodeToString(u.PublicKey),
				"display_name": u.DisplayName,
				"avatar_hash":  u.AvatarHash,
				"status":       u.Status,
				"roles":        roles,
				"version":      u.Version,
			}
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"users": out,
		})
	}
}

func handleUserKick(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermKickUsers) {
			return
		}

		var req struct {
			PubKey string `json:"pubkey"`
		}
		if !parsePayload(msg, &req) || req.PubKey == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "pubkey is required")
			return
		}

		pubKey, err := hex.DecodeString(req.PubKey)
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid pubkey")
			return
		}

		// Cannot kick owners
		if d.Hot.IsOwner(pubKey) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrForbidden, "cannot kick server owner")
			return
		}

		var target models.User
		if err := d.DB.Where("public_key = ?", pubKey).First(&target).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "user not found")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditUserKick, shared.TargetTypeUser, target.ID, map[string]any{
			"pubkey": req.PubKey,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		broadcastEvent(d, shared.TypeEventUserKicked, map[string]any{
			"pubkey": req.PubKey,
		})

		// Disconnect the kicked user
		if targetClient := d.Hub.GetClient(req.PubKey); targetClient != nil {
			// Set grace period on session
			auth.UpdateGracePeriod(d.DB, targetClient.SessionToken, 0)
			targetClient.Close()
		}
	}
}

func handleUserLeave(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			Mode string `json:"mode"`
		}
		if !parsePayload(msg, &req) || req.Mode == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "mode is required")
			return
		}

		var user models.User
		if err := d.DB.First(&user, "id = ?", client.UserID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to load user")
			return
		}

		switch req.Mode {
		case "leave":
			handleSimpleLeave(d, client, &user)
		case shared.ErasureModeGhost:
			handleGhostLeave(d, client, &user)
		case shared.ErasureModeForget:
			handleForgetLeave(d, client, &user)
		default:
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "mode must be leave, ghost, or forget")
			return
		}
	}
}

func handleSimpleLeave(d *Deps, client *ws.Client, user *models.User) {
	// Remove role assignments
	d.DB.Where("user_id = ?", user.ID).Delete(&models.UserRole{})
	// Delete sessions
	d.DB.Where("user_id = ?", user.ID).Delete(&models.Session{})
	// Set offline
	d.DB.Model(user).Update("status", shared.StatusOffline)
	// Close connection
	client.Close()
}

func handleGhostLeave(d *Deps, client *ws.Client, user *models.User) {
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Create sentinel user if it doesn't exist
		ensureSentinelUser(tx)

		// 2. Reassign messages to sentinel
		tx.Model(&models.Message{}).Where("author_id = ?", user.ID).
			Update("author_id", shared.SentinelUserID)

		// 3. Reassign files to sentinel
		tx.Model(&models.File{}).Where("uploader_id = ?", user.ID).
			Update("uploader_id", shared.SentinelUserID)

		// 4. Delete role assignments
		tx.Where("user_id = ?", user.ID).Delete(&models.UserRole{})

		// 5. Soft-leave DM conversations
		now := time.Now()
		tx.Model(&models.DMParticipant{}).
			Where("user_id = ? AND left_at IS NULL", user.ID).
			Update("left_at", now)

		// 6. Delete user
		tx.Delete(user)

		// 7. Create erasure record
		tx.Create(&models.ErasureRecord{
			ID:        newULID(),
			PublicKey: user.PublicKey,
			Mode:      shared.ErasureModeGhost,
			ErasedAt:  now,
		})

		return nil
	})
	if err != nil {
		slog.Error("ghost leave transaction failed", "error", err)
		return
	}

	// 8. Broadcast erasure event
	broadcastEvent(d, shared.TypeEventUserErased, map[string]any{
		"pubkey": hex.EncodeToString(user.PublicKey),
		"mode":   shared.ErasureModeGhost,
	})

	// 9. Invalidate sessions and close
	d.DB.Where("user_id = ?", user.ID).Delete(&models.Session{})
	client.Close()
}

func handleForgetLeave(d *Deps, client *ws.Client, user *models.User) {
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Delete all messages
		tx.Where("author_id = ?", user.ID).Delete(&models.Message{})

		// 2. Delete all files (DB rows + disk)
		var files []models.File
		tx.Where("uploader_id = ?", user.ID).Find(&files)
		for _, f := range files {
			os.Remove(f.StoragePath)
			if f.ThumbPath != nil {
				os.Remove(*f.ThumbPath)
			}
		}
		tx.Where("uploader_id = ?", user.ID).Delete(&models.File{})

		// 3. Delete DM messages
		tx.Where("sender_id = ?", user.ID).Delete(&models.DMMessage{})

		// 4. Soft-leave DM conversations
		now := time.Now()
		tx.Model(&models.DMParticipant{}).
			Where("user_id = ? AND left_at IS NULL", user.ID).
			Update("left_at", now)

		// 5. Delete empty DM conversations (no active participants)
		var convIDs []string
		tx.Model(&models.DMParticipant{}).
			Select("conversation_id").
			Where("user_id = ?", user.ID).
			Pluck("conversation_id", &convIDs)
		for _, cid := range convIDs {
			var activeCount int64
			tx.Model(&models.DMParticipant{}).
				Where("conversation_id = ? AND left_at IS NULL", cid).
				Count(&activeCount)
			if activeCount == 0 {
				tx.Where("conversation_id = ?", cid).Delete(&models.DMParticipant{})
				tx.Where("conversation_id = ?", cid).Delete(&models.DMMessage{})
				tx.Where("id = ?", cid).Delete(&models.DMConversation{})
			}
		}

		// 6. Delete role assignments
		tx.Where("user_id = ?", user.ID).Delete(&models.UserRole{})

		// 7. Delete user
		tx.Delete(user)

		// 8. Create erasure record
		tx.Create(&models.ErasureRecord{
			ID:        newULID(),
			PublicKey: user.PublicKey,
			Mode:      shared.ErasureModeForget,
			ErasedAt:  now,
		})

		return nil
	})
	if err != nil {
		slog.Error("forget leave transaction failed", "error", err)
		return
	}

	// 9. Broadcast erasure event
	broadcastEvent(d, shared.TypeEventUserErased, map[string]any{
		"pubkey": hex.EncodeToString(user.PublicKey),
		"mode":   shared.ErasureModeForget,
	})

	// 10. Invalidate sessions and close
	d.DB.Where("user_id = ?", user.ID).Delete(&models.Session{})
	client.Close()
}

func ensureSentinelUser(tx *gorm.DB) {
	var count int64
	tx.Model(&models.User{}).Where("id = ?", shared.SentinelUserID).Count(&count)
	if count > 0 {
		return
	}
	tx.Create(&models.User{
		ID:          shared.SentinelUserID,
		PublicKey:   make([]byte, 32),
		DisplayName: "Deleted User",
		Status:      shared.StatusOffline,
		Version:     0,
	})
}

func userProfileMap(u *models.User, roles []string) map[string]any {
	m := map[string]any{
		"pubkey":       hex.EncodeToString(u.PublicKey),
		"display_name": u.DisplayName,
		"status":       u.Status,
		"roles":        roles,
		"version":      u.Version,
	}
	if u.Avatar != nil {
		m["avatar_id"] = *u.Avatar
	}
	if u.AvatarHash != "" {
		m["avatar_hash"] = u.AvatarHash
	}
	if u.Bio != nil {
		m["bio"] = *u.Bio
	}
	return m
}
