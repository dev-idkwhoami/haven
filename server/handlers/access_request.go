package handlers

import (
	"encoding/hex"
	"encoding/json"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterAccessRequestHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeAccessRequestList, handleAccessRequestList(d))
	router.Register(shared.TypeAccessRequestApprove, handleAccessRequestApprove(d))
	router.Register(shared.TypeAccessRequestReject, handleAccessRequestReject(d))
}

func handleAccessRequestList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageAccessRequests) {
			return
		}

		var requests []models.AccessRequest
		d.DB.Where("status = ?", "pending").Order("created_at ASC").Find(&requests)

		out := make([]map[string]any, len(requests))
		for i, r := range requests {
			pubKeyHex := hex.EncodeToString(r.PublicKey)
			entry := map[string]any{
				"id":           r.ID,
				"pubkey":       pubKeyHex,
				"display_name": r.DisplayName,
				"is_online":    d.WaitingRoom.IsOnline(pubKeyHex),
				"created_at":   r.CreatedAt,
			}
			if r.Message != nil {
				entry["message"] = *r.Message
			}
			out[i] = entry
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"requests": out,
		})
	}
}

func handleAccessRequestApprove(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageAccessRequests) {
			return
		}

		var req struct {
			ID string `json:"id"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var ar models.AccessRequest
		if err := d.DB.Where("id = ? AND status = ?", req.ID, "pending").First(&ar).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrAccessRequestNotFound, "access request not found")
			return
		}

		d.DB.Model(&ar).Updates(map[string]any{
			"status":      "approved",
			"reviewed_by": client.UserID,
		})

		pubKeyHex := hex.EncodeToString(ar.PublicKey)
		d.WaitingRoom.Approve(pubKeyHex)

		auditLog(d.DB, client.UserID, shared.AuditAccessRequestApprove, shared.TargetTypeAccessRequest, ar.ID, map[string]any{
			"pubkey":       pubKeyHex,
			"display_name": ar.DisplayName,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		// Broadcast event.
		eventPayload, _ := json.Marshal(map[string]any{
			"id":           ar.ID,
			"pubkey":       pubKeyHex,
			"display_name": ar.DisplayName,
			"approved_by":  client.PubKeyHex,
		})
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventAccessRequestApproved, json.RawMessage(eventPayload))
		d.Hub.Broadcast(eventBytes)
	}
}

func handleAccessRequestReject(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageAccessRequests) {
			return
		}

		var req struct {
			ID string `json:"id"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var ar models.AccessRequest
		if err := d.DB.Where("id = ? AND status = ?", req.ID, "pending").First(&ar).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrAccessRequestNotFound, "access request not found")
			return
		}

		d.DB.Model(&ar).Updates(map[string]any{
			"status":      "rejected",
			"reviewed_by": client.UserID,
		})

		pubKeyHex := hex.EncodeToString(ar.PublicKey)
		d.WaitingRoom.Reject(pubKeyHex)

		auditLog(d.DB, client.UserID, shared.AuditAccessRequestReject, shared.TargetTypeAccessRequest, ar.ID, map[string]any{
			"pubkey":       pubKeyHex,
			"display_name": ar.DisplayName,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		// Broadcast event.
		eventPayload, _ := json.Marshal(map[string]any{
			"id":           ar.ID,
			"pubkey":       pubKeyHex,
			"display_name": ar.DisplayName,
			"rejected_by":  client.PubKeyHex,
		})
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventAccessRequestRejected, json.RawMessage(eventPayload))
		d.Hub.Broadcast(eventBytes)
	}
}
