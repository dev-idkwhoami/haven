package handlers

import (
	"encoding/hex"
	"time"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterSyncHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeSyncSubscribe, handleSyncSubscribe(d))
	router.Register(shared.TypeSyncRequest, handleSyncRequest(d))
}

func handleSyncSubscribe(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		// Field selection stored for session duration. For now, acknowledge the subscription.
		// The actual field filtering is deferred to a future optimization pass
		// since it requires intercepting all outbound events.
		ws.SendOK(client, msg.Type, msg.ID, nil)
	}
}

func handleSyncRequest(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			Users        map[string]int64 `json:"users"`
			Channels     map[string]int64 `json:"channels"`
			Categories   map[string]int64 `json:"categories"`
			Roles        map[string]int64 `json:"roles"`
			Server       *int64           `json:"server"`
			ErasureSince *time.Time       `json:"erasure_since"`
		}
		parsePayload(msg, &req)

		// --- Users ---
		var allUsers []models.User
		d.DB.Where("id != ?", shared.SentinelUserID).Find(&allUsers)

		var changedUsers []map[string]any
		existingPubKeys := make(map[string]bool)
		for _, u := range allUsers {
			pkHex := hex.EncodeToString(u.PublicKey)
			existingPubKeys[pkHex] = true
			clientVer, known := req.Users[pkHex]
			if !known || u.Version > clientVer {
				roles := getUserRoleIDs(d.DB, u.ID)
				changedUsers = append(changedUsers, map[string]any{
					"pubkey":       pkHex,
					"display_name": u.DisplayName,
					"avatar_hash":  u.AvatarHash,
					"status":       u.Status,
					"roles":        roles,
					"version":      u.Version,
				})
			}
		}
		var deletedUsers []string
		for pkHex := range req.Users {
			if !existingPubKeys[pkHex] {
				deletedUsers = append(deletedUsers, pkHex)
			}
		}

		// --- Channels ---
		var allChannels []models.Channel
		d.DB.Find(&allChannels)

		var changedChannels []map[string]any
		existingChannelIDs := make(map[string]bool)
		for _, ch := range allChannels {
			existingChannelIDs[ch.ID] = true
			if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, ch.ID) {
				continue
			}
			clientVer, known := req.Channels[ch.ID]
			if !known || ch.Version > clientVer {
				changedChannels = append(changedChannels, channelSyncMap(&ch))
			}
		}
		var deletedChannels []string
		for chID := range req.Channels {
			if !existingChannelIDs[chID] {
				deletedChannels = append(deletedChannels, chID)
			}
		}

		// --- Categories ---
		var allCategories []models.Category
		d.DB.Find(&allCategories)

		var changedCategories []map[string]any
		existingCatIDs := make(map[string]bool)
		for _, c := range allCategories {
			existingCatIDs[c.ID] = true
			clientVer, known := req.Categories[c.ID]
			if !known || c.Version > clientVer {
				changedCategories = append(changedCategories, categoryToMap(&c))
			}
		}
		var deletedCategories []string
		for catID := range req.Categories {
			if !existingCatIDs[catID] {
				deletedCategories = append(deletedCategories, catID)
			}
		}

		// --- Roles ---
		var allRoles []models.Role
		d.DB.Find(&allRoles)

		var changedRoles []map[string]any
		existingRoleIDs := make(map[string]bool)
		for _, r := range allRoles {
			existingRoleIDs[r.ID] = true
			clientVer, known := req.Roles[r.ID]
			if !known || r.Version > clientVer {
				changedRoles = append(changedRoles, roleToMap(&r))
			}
		}
		var deletedRoles []string
		for rID := range req.Roles {
			if !existingRoleIDs[rID] {
				deletedRoles = append(deletedRoles, rID)
			}
		}

		// --- Server ---
		var serverOut map[string]any
		var srv models.Server
		if err := d.DB.First(&srv).Error; err == nil {
			if req.Server == nil || srv.Version > *req.Server {
				serverOut = map[string]any{
					"name":        srv.Name,
					"access_mode": srv.AccessMode,
					"version":     srv.Version,
				}
				if srv.Description != nil {
					serverOut["description"] = *srv.Description
				}
				if srv.Icon != nil {
					serverOut["icon_id"] = *srv.Icon
				}
				if srv.IconHash != "" {
					serverOut["icon_hash"] = srv.IconHash
				}
			}
		}

		// --- Erasure records ---
		var erasureRecords []models.ErasureRecord
		if req.ErasureSince != nil {
			d.DB.Where("erased_at > ?", *req.ErasureSince).Find(&erasureRecords)
		} else {
			d.DB.Find(&erasureRecords)
		}
		erasureOut := make([]map[string]any, len(erasureRecords))
		for i, er := range erasureRecords {
			erasureOut[i] = map[string]any{
				"pubkey":    hex.EncodeToString(er.PublicKey),
				"mode":      er.Mode,
				"erased_at": er.ErasedAt,
			}
		}

		resp := map[string]any{
			"users":              changedUsers,
			"channels":           changedChannels,
			"categories":         changedCategories,
			"roles":              changedRoles,
			"deleted_users":      deletedUsers,
			"deleted_channels":   deletedChannels,
			"deleted_categories": deletedCategories,
			"deleted_roles":      deletedRoles,
			"erasure_records":    erasureOut,
		}
		if serverOut != nil {
			resp["server"] = serverOut
		}

		ws.SendOK(client, msg.Type, msg.ID, resp)
	}
}

func channelSyncMap(ch *models.Channel) map[string]any {
	return map[string]any{
		"id":          ch.ID,
		"category_id": ch.CategoryID,
		"name":        ch.Name,
		"type":        ch.Type,
		"position":    ch.Position,
		"version":     ch.Version,
	}
}
