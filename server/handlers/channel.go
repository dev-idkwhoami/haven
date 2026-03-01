package handlers

import (
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterChannelHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeChannelList, handleChannelList(d))
	router.Register(shared.TypeChannelCreate, handleChannelCreate(d))
	router.Register(shared.TypeChannelUpdate, handleChannelUpdate(d))
	router.Register(shared.TypeChannelDelete, handleChannelDelete(d))
}

func handleChannelList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			CategoryID *string `json:"category_id"`
		}
		parsePayload(msg, &req)

		query := d.DB.Order("position ASC")
		if req.CategoryID != nil && *req.CategoryID != "" {
			query = query.Where("category_id = ?", *req.CategoryID)
		}

		var channels []models.Channel
		query.Find(&channels)

		// Filter by access
		out := make([]map[string]any, 0, len(channels))
		for _, ch := range channels {
			if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, ch.ID) {
				continue
			}
			out = append(out, channelToMap(d, &ch))
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"channels": out,
		})
	}
}

func handleChannelCreate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageChannels) {
			return
		}

		var req struct {
			CategoryID  string   `json:"category_id"`
			Name        string   `json:"name"`
			Type        string   `json:"type"`
			RoleIDs     []string `json:"role_ids"`
			OpusBitrate *int     `json:"opus_bitrate"`
		}
		if !parsePayload(msg, &req) || req.CategoryID == "" || req.Name == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "category_id and name are required")
			return
		}
		if req.Type == "" {
			req.Type = shared.ChannelTypeText
		}

		// Verify category exists
		var cat models.Category
		if err := d.DB.First(&cat, "id = ?", req.CategoryID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "category not found")
			return
		}

		// Auto-assign position at end of category
		var maxPos int
		d.DB.Model(&models.Channel{}).Where("category_id = ?", req.CategoryID).
			Select("COALESCE(MAX(position), -10)").Scan(&maxPos)

		ch := models.Channel{
			ID:          newULID(),
			CategoryID:  req.CategoryID,
			Name:        req.Name,
			Type:        req.Type,
			Position:    maxPos + 10,
			OpusBitrate: req.OpusBitrate,
			Version:     1,
		}
		if err := d.DB.Create(&ch).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create channel")
			return
		}

		// Create role access entries
		for _, roleID := range req.RoleIDs {
			d.DB.Create(&models.ChannelRoleAccess{
				ChannelID: ch.ID,
				RoleID:    roleID,
			})
		}

		auditLog(d.DB, client.UserID, shared.AuditChannelCreate, shared.TargetTypeChannel, ch.ID, map[string]any{
			"name":        ch.Name,
			"category_id": ch.CategoryID,
			"type":        ch.Type,
		})

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"id":       ch.ID,
			"position": ch.Position,
			"version":  ch.Version,
		})

		broadcastEvent(d, shared.TypeEventChannelCreated, channelToMap(d, &ch))
	}
}

func handleChannelUpdate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageChannels) {
			return
		}

		var req struct {
			ID          string   `json:"id"`
			Name        *string  `json:"name"`
			CategoryID  *string  `json:"category_id"`
			Position    *int     `json:"position"`
			RoleIDs     []string `json:"role_ids"`
			OpusBitrate *int     `json:"opus_bitrate"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var ch models.Channel
		if err := d.DB.First(&ch, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "channel not found")
			return
		}

		updates := map[string]any{}
		details := map[string]any{"id": ch.ID}

		if req.Name != nil {
			details["name_old"] = ch.Name
			updates["name"] = *req.Name
		}
		if req.CategoryID != nil {
			// Moving to a different category — reset position to end
			var maxPos int
			d.DB.Model(&models.Channel{}).Where("category_id = ?", *req.CategoryID).
				Select("COALESCE(MAX(position), -10)").Scan(&maxPos)
			updates["category_id"] = *req.CategoryID
			updates["position"] = maxPos + 10
		}
		if req.Position != nil && req.CategoryID == nil {
			updates["position"] = *req.Position
		}
		if req.OpusBitrate != nil {
			updates["opus_bitrate"] = *req.OpusBitrate
		}

		// Update role access (full replace)
		if req.RoleIDs != nil {
			d.DB.Where("channel_id = ?", ch.ID).Delete(&models.ChannelRoleAccess{})
			for _, roleID := range req.RoleIDs {
				d.DB.Create(&models.ChannelRoleAccess{
					ChannelID: ch.ID,
					RoleID:    roleID,
				})
			}
		}

		if len(updates) == 0 && req.RoleIDs == nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "no fields to update")
			return
		}

		ch.Version++
		updates["version"] = ch.Version
		if err := d.DB.Model(&ch).Updates(updates).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to update channel")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditChannelUpdate, shared.TargetTypeChannel, ch.ID, details)

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"version": ch.Version,
		})

		d.DB.First(&ch, "id = ?", req.ID)
		broadcastEvent(d, shared.TypeEventChannelUpdated, channelToMap(d, &ch))
	}
}

func handleChannelDelete(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageChannels) {
			return
		}

		var req struct {
			ID string `json:"id"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var ch models.Channel
		if err := d.DB.First(&ch, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "channel not found")
			return
		}

		// Clean up role access
		d.DB.Where("channel_id = ?", ch.ID).Delete(&models.ChannelRoleAccess{})
		d.DB.Delete(&ch)

		auditLog(d.DB, client.UserID, shared.AuditChannelDelete, shared.TargetTypeChannel, ch.ID, map[string]any{
			"name": ch.Name,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		broadcastEvent(d, shared.TypeEventChannelDeleted, map[string]any{
			"id": ch.ID,
		})
	}
}

func channelToMap(d *Deps, ch *models.Channel) map[string]any {
	// Get role IDs for this channel
	var accesses []models.ChannelRoleAccess
	d.DB.Where("channel_id = ?", ch.ID).Find(&accesses)
	roleIDs := make([]string, len(accesses))
	for i, a := range accesses {
		roleIDs[i] = a.RoleID
	}

	m := map[string]any{
		"id":          ch.ID,
		"category_id": ch.CategoryID,
		"name":        ch.Name,
		"type":        ch.Type,
		"position":    ch.Position,
		"role_ids":    roleIDs,
		"version":     ch.Version,
	}
	if ch.OpusBitrate != nil {
		m["opus_bitrate"] = *ch.OpusBitrate
	}
	return m
}
