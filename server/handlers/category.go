package handlers

import (
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterCategoryHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeCategoryList, handleCategoryList(d))
	router.Register(shared.TypeCategoryCreate, handleCategoryCreate(d))
	router.Register(shared.TypeCategoryUpdate, handleCategoryUpdate(d))
	router.Register(shared.TypeCategoryDelete, handleCategoryDelete(d))
}

func handleCategoryList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var categories []models.Category
		d.DB.Order("position ASC").Find(&categories)

		out := make([]map[string]any, len(categories))
		for i, c := range categories {
			out[i] = categoryToMap(&c)
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"categories": out,
		})
	}
}

func handleCategoryCreate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageChannels) {
			return
		}

		var req struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if !parsePayload(msg, &req) || req.Name == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "name is required")
			return
		}
		if req.Type == "" {
			req.Type = shared.ChannelTypeText
		}

		// Auto-assign position at end
		var maxPos int
		d.DB.Model(&models.Category{}).Select("COALESCE(MAX(position), -10)").Scan(&maxPos)

		cat := models.Category{
			ID:       newULID(),
			Name:     req.Name,
			Position: maxPos + 10,
			Type:     req.Type,
			Version:  1,
		}
		if err := d.DB.Create(&cat).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create category")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditCategoryCreate, shared.TargetTypeCategory, cat.ID, map[string]any{
			"name": cat.Name,
		})

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"id":       cat.ID,
			"position": cat.Position,
			"version":  cat.Version,
		})

		broadcastEvent(d, shared.TypeEventCategoryCreated, categoryToMap(&cat))
	}
}

func handleCategoryUpdate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageChannels) {
			return
		}

		var req struct {
			ID       string  `json:"id"`
			Name     *string `json:"name"`
			Position *int    `json:"position"`
			Type     *string `json:"type"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var cat models.Category
		if err := d.DB.First(&cat, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "category not found")
			return
		}

		updates := map[string]any{}
		details := map[string]any{"id": cat.ID}

		if req.Name != nil {
			details["name_old"] = cat.Name
			updates["name"] = *req.Name
		}
		if req.Position != nil {
			updates["position"] = *req.Position
		}
		if req.Type != nil {
			updates["type"] = *req.Type
		}

		if len(updates) == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "no fields to update")
			return
		}

		cat.Version++
		updates["version"] = cat.Version
		if err := d.DB.Model(&cat).Updates(updates).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to update category")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditCategoryUpdate, shared.TargetTypeCategory, cat.ID, details)

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"version": cat.Version,
		})

		d.DB.First(&cat, "id = ?", req.ID)
		broadcastEvent(d, shared.TypeEventCategoryUpdated, categoryToMap(&cat))
	}
}

func handleCategoryDelete(d *Deps) ws.HandlerFunc {
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

		var cat models.Category
		if err := d.DB.First(&cat, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "category not found")
			return
		}

		// Collect channel IDs that will be cascade-deleted
		var channels []models.Channel
		d.DB.Where("category_id = ?", req.ID).Find(&channels)
		deletedChannelIDs := make([]string, len(channels))
		for i, ch := range channels {
			deletedChannelIDs[i] = ch.ID
		}

		// Delete channels in category
		d.DB.Where("category_id = ?", req.ID).Delete(&models.Channel{})
		// Delete channel role access entries for those channels
		for _, chID := range deletedChannelIDs {
			d.DB.Where("channel_id = ?", chID).Delete(&models.ChannelRoleAccess{})
		}
		// Delete the category
		d.DB.Delete(&cat)

		auditLog(d.DB, client.UserID, shared.AuditCategoryDelete, shared.TargetTypeCategory, cat.ID, map[string]any{
			"name":                cat.Name,
			"deleted_channel_ids": deletedChannelIDs,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		broadcastEvent(d, shared.TypeEventCategoryDeleted, map[string]any{
			"id":                  cat.ID,
			"deleted_channel_ids": deletedChannelIDs,
		})
	}
}

func categoryToMap(c *models.Category) map[string]any {
	return map[string]any{
		"id":       c.ID,
		"name":     c.Name,
		"position": c.Position,
		"type":     c.Type,
		"version":  c.Version,
	}
}

func broadcastEvent(d *Deps, eventType string, payload any) {
	eventBytes, err := ws.MarshalEvent(eventType, payload)
	if err != nil {
		return
	}
	d.Hub.Broadcast(eventBytes)
}
