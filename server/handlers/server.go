package handlers

import (
	"encoding/json"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterServerHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeServerInfo, handleServerInfo(d))
	router.Register(shared.TypeServerUpdate, handleServerUpdate(d))
}

func handleServerInfo(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var srv models.Server
		if err := d.DB.First(&srv).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to load server info")
			return
		}

		var memberCount int64
		d.DB.Model(&models.User{}).Count(&memberCount)

		var iconID, iconHash *string
		if srv.Icon != nil {
			iconID = srv.Icon
		}
		if srv.IconHash != "" {
			iconHash = &srv.IconHash
		}

		// Resolve effective access mode: config override takes precedence.
		accessMode := srv.AccessMode
		if override := d.Hot.AccessMode(); override != "" {
			accessMode = override
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"name":         srv.Name,
			"description":  srv.Description,
			"icon_id":      iconID,
			"icon_hash":    iconHash,
			"access_mode":  accessMode,
			"member_count": memberCount,
			"version":      srv.Version,
		})
	}
}

func handleServerUpdate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageServer) {
			return
		}

		var req struct {
			Name              *string `json:"name"`
			Description       *string `json:"description"`
			IconID            *string `json:"icon_id"`
			AccessMode        *string `json:"access_mode"`
			AccessPassword    *string `json:"access_password"`
			MaxFileSize       *int64  `json:"max_file_size"`
			TotalStorageLimit *int64  `json:"total_storage_limit"`
			DefaultChannelID  *string `json:"default_channel_id"`
			WelcomeMessage    *string `json:"welcome_message"`
		}
		if !parsePayload(msg, &req) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "invalid payload")
			return
		}

		var srv models.Server
		if err := d.DB.First(&srv).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to load server")
			return
		}

		updates := map[string]any{}
		details := map[string]any{}

		if req.Name != nil {
			details["name_old"] = srv.Name
			updates["name"] = *req.Name
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		if req.IconID != nil {
			updates["icon"] = *req.IconID
		}
		if req.AccessMode != nil {
			details["access_mode_old"] = srv.AccessMode
			updates["access_mode"] = *req.AccessMode
		}
		if req.AccessPassword != nil {
			hash, err := bcrypt.GenerateFromPassword([]byte(*req.AccessPassword), bcrypt.DefaultCost)
			if err != nil {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to hash password")
				return
			}
			hashStr := string(hash)
			updates["access_password"] = hashStr
		}
		if req.MaxFileSize != nil {
			updates["max_file_size"] = *req.MaxFileSize
		}
		if req.TotalStorageLimit != nil {
			updates["total_storage_limit"] = *req.TotalStorageLimit
		}
		if req.DefaultChannelID != nil {
			updates["default_channel_id"] = *req.DefaultChannelID
		}
		if req.WelcomeMessage != nil {
			updates["welcome_message"] = *req.WelcomeMessage
		}

		if len(updates) == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "no fields to update")
			return
		}

		srv.Version++
		updates["version"] = srv.Version
		if err := d.DB.Model(&srv).Updates(updates).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to update server")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditServerUpdate, shared.TargetTypeServer, srv.ID, details)

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"version": srv.Version,
		})

		// Reload and broadcast
		d.DB.First(&srv)
		var memberCount int64
		d.DB.Model(&models.User{}).Count(&memberCount)

		// Resolve effective access mode for broadcast and enforcement.
		effectiveMode := srv.AccessMode
		if override := d.Hot.AccessMode(); override != "" {
			effectiveMode = override
		}

		eventPayload, _ := json.Marshal(map[string]any{
			"name":         srv.Name,
			"description":  srv.Description,
			"icon_id":      srv.Icon,
			"icon_hash":    srv.IconHash,
			"access_mode":  effectiveMode,
			"member_count": memberCount,
			"version":      srv.Version,
		})
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventServerUpdated, json.RawMessage(eventPayload))
		d.Hub.Broadcast(eventBytes)

		// Enforcement: if effective access_mode is "allowlist", kick non-qualifying clients.
		if effectiveMode == shared.AccessModeAllowlist {
			d.Hub.ForEachClient(func(c *ws.Client) {
				if d.Hot.IsOwner(c.PubKey) || d.Hot.IsAllowlisted(c.PubKey) {
					return
				}
				// Check for approved access request.
				var count int64
				d.DB.Model(&models.AccessRequest{}).Where("public_key = ? AND status = ?", c.PubKey, "approved").Count(&count)
				if count > 0 {
					return
				}
				// Skip the admin who made the change.
				if c.UserID == client.UserID {
					return
				}
				slog.Info("kicking client after allowlist mode enabled", "pubkey", c.PubKeyHex)
				kickMsg, _ := ws.MarshalEvent(shared.TypeAuthError, map[string]any{
					"code":    shared.ErrNotAllowlisted,
					"message": "server switched to allowlist mode",
				})
				c.SendCh <- kickMsg
				d.Hub.Unregister <- c
			})
		}
	}
}
