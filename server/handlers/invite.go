package handlers

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
	"time"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterInviteHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeInviteCreate, handleInviteCreate(d))
	router.Register(shared.TypeInviteList, handleInviteList(d))
	router.Register(shared.TypeInviteRevoke, handleInviteRevoke(d))
}

func handleInviteCreate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageInvites) {
			return
		}

		var req struct {
			UsesLeft  *int       `json:"uses_left"`
			ExpiresAt *time.Time `json:"expires_at"`
		}
		parsePayload(msg, &req)

		code := generateInviteCode()

		invite := models.InviteCode{
			ID:        newULID(),
			Code:      code,
			CreatedBy: &client.UserID,
			UsesLeft:  req.UsesLeft,
			ExpiresAt: req.ExpiresAt,
		}
		if err := d.DB.Create(&invite).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create invite")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditInviteCreate, shared.TargetTypeInvite, invite.ID, map[string]any{
			"code": code,
		})

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"id":   invite.ID,
			"code": invite.Code,
		})
	}
}

func handleInviteList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageInvites) {
			return
		}

		var invites []models.InviteCode
		d.DB.Order("created_at DESC").Find(&invites)

		out := make([]map[string]any, len(invites))
		for i, inv := range invites {
			entry := map[string]any{
				"id":         inv.ID,
				"code":       inv.Code,
				"created_at": inv.CreatedAt,
			}
			if inv.UsesLeft != nil {
				entry["uses_left"] = *inv.UsesLeft
			}
			if inv.ExpiresAt != nil {
				entry["expires_at"] = *inv.ExpiresAt
			}
			if inv.CreatedBy != nil {
				entry["created_by_pubkey"] = getActorPubKey(d, inv.CreatedBy)
			}
			out[i] = entry
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"invites": out,
		})
	}
}

func handleInviteRevoke(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageInvites) {
			return
		}

		var req struct {
			ID string `json:"id"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		result := d.DB.Where("id = ?", req.ID).Delete(&models.InviteCode{})
		if result.RowsAffected == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "invite not found")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditInviteRevoke, shared.TargetTypeInvite, req.ID, nil)

		ws.SendOK(client, msg.Type, msg.ID, nil)
	}
}

func generateInviteCode() string {
	b := make([]byte, 10)
	rand.Read(b)
	return strings.TrimRight(base32.StdEncoding.EncodeToString(b), "=")
}
