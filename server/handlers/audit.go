package handlers

import (
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterAuditHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeAuditList, handleAuditList(d))
}

func handleAuditList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageServer) {
			return
		}

		var req struct {
			Before      string  `json:"before"`
			Limit       int     `json:"limit"`
			Action      *string `json:"action"`
			ActorPubKey *string `json:"actor_pubkey"`
		}
		parsePayload(msg, &req)

		if req.Limit <= 0 || req.Limit > 100 {
			req.Limit = 50
		}

		query := d.DB.Model(&models.AuditLogEntry{}).Order("id DESC").Limit(req.Limit + 1)

		if req.Before != "" {
			query = query.Where("id < ?", req.Before)
		}
		if req.Action != nil && *req.Action != "" {
			query = query.Where("action = ?", *req.Action)
		}
		if req.ActorPubKey != nil && *req.ActorPubKey != "" {
			user, err := getUserByPubKeyHex(d.DB, *req.ActorPubKey)
			if err == nil {
				query = query.Where("actor_id = ?", user.ID)
			}
		}

		var entries []models.AuditLogEntry
		query.Find(&entries)

		hasMore := len(entries) > req.Limit
		if hasMore {
			entries = entries[:req.Limit]
		}

		out := make([]map[string]any, len(entries))
		for i, e := range entries {
			entry := map[string]any{
				"id":          e.ID,
				"action":      e.Action,
				"target_type": e.TargetType,
				"target_id":   e.TargetID,
				"created_at":  e.CreatedAt,
			}
			if e.ActorID != nil {
				entry["actor_pubkey"] = getActorPubKey(d, e.ActorID)
			}
			if e.Details != nil {
				entry["details"] = *e.Details
			}
			out[i] = entry
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"entries":  out,
			"has_more": hasMore,
		})
	}
}
