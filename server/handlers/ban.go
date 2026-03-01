package handlers

import (
	"encoding/hex"
	"time"

	"haven/server/auth"
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterBanHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeBanCreate, handleBanCreate(d))
	router.Register(shared.TypeBanRemove, handleBanRemove(d))
	router.Register(shared.TypeBanList, handleBanList(d))
}

func handleBanCreate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermBanUsers) {
			return
		}

		var req struct {
			PubKey    string     `json:"pubkey"`
			Reason    *string    `json:"reason"`
			ExpiresAt *time.Time `json:"expires_at"`
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

		// Cannot ban owners
		if d.Hot.IsOwner(pubKey) {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrForbidden, "cannot ban server owner")
			return
		}

		ban := models.Ban{
			ID:        newULID(),
			PublicKey: pubKey,
			Reason:    req.Reason,
			BannedBy:  &client.UserID,
			ExpiresAt: req.ExpiresAt,
		}
		if err := d.DB.Create(&ban).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create ban")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditUserBan, shared.TargetTypeUser, ban.ID, map[string]any{
			"pubkey": req.PubKey,
			"reason": req.Reason,
		})

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"id": ban.ID,
		})

		broadcastEvent(d, shared.TypeEventUserBanned, map[string]any{
			"pubkey": req.PubKey,
			"reason": req.Reason,
		})

		// Disconnect the banned user
		if targetClient := d.Hub.GetClient(req.PubKey); targetClient != nil {
			auth.UpdateGracePeriod(d.DB, targetClient.SessionToken, 0)
			targetClient.Close()
		}
	}
}

func handleBanRemove(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermBanUsers) {
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

		result := d.DB.Where("public_key = ?", pubKey).Delete(&models.Ban{})
		if result.RowsAffected == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "no ban found for this pubkey")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditUserUnban, shared.TargetTypeUser, req.PubKey, map[string]any{
			"pubkey": req.PubKey,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		broadcastEvent(d, shared.TypeEventUserUnbanned, map[string]any{
			"pubkey": req.PubKey,
		})
	}
}

func handleBanList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermBanUsers) {
			return
		}

		var bans []models.Ban
		d.DB.Order("created_at DESC").Find(&bans)

		out := make([]map[string]any, len(bans))
		for i, b := range bans {
			entry := map[string]any{
				"id":         b.ID,
				"pubkey":     hex.EncodeToString(b.PublicKey),
				"created_at": b.CreatedAt,
			}
			if b.Reason != nil {
				entry["reason"] = *b.Reason
			}
			if b.ExpiresAt != nil {
				entry["expires_at"] = *b.ExpiresAt
			}
			if b.BannedBy != nil {
				entry["banned_by_pubkey"] = getActorPubKey(d, b.BannedBy)
			}
			out[i] = entry
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"bans": out,
		})
	}
}
