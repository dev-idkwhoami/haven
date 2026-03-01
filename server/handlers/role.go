package handlers

import (
	"encoding/hex"

	"haven/server/middleware"
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"
)

func RegisterRoleHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeRoleList, handleRoleList(d))
	router.Register(shared.TypeRoleCreate, handleRoleCreate(d))
	router.Register(shared.TypeRoleUpdate, handleRoleUpdate(d))
	router.Register(shared.TypeRoleDelete, handleRoleDelete(d))
	router.Register(shared.TypeRoleAssign, handleRoleAssign(d))
	router.Register(shared.TypeRoleRevoke, handleRoleRevoke(d))
}

func handleRoleList(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var roles []models.Role
		d.DB.Order("position DESC").Find(&roles)

		out := make([]map[string]any, len(roles))
		for i, r := range roles {
			out[i] = roleToMap(&r)
		}

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"roles": out,
		})
	}
}

func handleRoleCreate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageRoles) {
			return
		}

		var req struct {
			Name        string  `json:"name"`
			Color       *string `json:"color"`
			Permissions int64   `json:"permissions"`
		}
		if !parsePayload(msg, &req) || req.Name == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "name is required")
			return
		}

		// Position hierarchy: new role is placed below the creator's highest role
		callerPos := middleware.GetHighestRolePosition(d.DB, client.UserID)
		if !d.Hot.IsOwner(client.PubKey) && callerPos <= 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "cannot create roles without a positioned role")
			return
		}

		// Auto-assign position: one below the creator's highest
		var minPos int
		d.DB.Model(&models.Role{}).Select("COALESCE(MIN(position), 10)").Scan(&minPos)
		newPos := minPos - 10
		if newPos < 0 {
			newPos = 0
		}

		role := models.Role{
			ID:          newULID(),
			Name:        req.Name,
			Color:       req.Color,
			Position:    newPos,
			Permissions: req.Permissions,
			Version:     1,
		}
		if err := d.DB.Create(&role).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to create role")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditRoleCreate, shared.TargetTypeRole, role.ID, map[string]any{
			"name": role.Name,
		})

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"id":       role.ID,
			"position": role.Position,
			"version":  role.Version,
		})

		broadcastEvent(d, shared.TypeEventRoleCreated, roleToMap(&role))
	}
}

func handleRoleUpdate(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageRoles) {
			return
		}

		var req struct {
			ID          string  `json:"id"`
			Name        *string `json:"name"`
			Color       *string `json:"color"`
			Position    *int    `json:"position"`
			Permissions *int64  `json:"permissions"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var role models.Role
		if err := d.DB.First(&role, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "role not found")
			return
		}

		// Position hierarchy enforcement: can only modify roles below own highest position
		if !d.Hot.IsOwner(client.PubKey) {
			callerPos := middleware.GetHighestRolePosition(d.DB, client.UserID)
			if role.Position >= callerPos {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "cannot modify roles at or above your position")
				return
			}
		}

		updates := map[string]any{}
		if req.Name != nil {
			updates["name"] = *req.Name
		}
		if req.Color != nil {
			updates["color"] = *req.Color
		}
		if req.Position != nil {
			updates["position"] = *req.Position
		}
		if req.Permissions != nil {
			updates["permissions"] = *req.Permissions
		}

		if len(updates) == 0 {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "no fields to update")
			return
		}

		role.Version++
		updates["version"] = role.Version
		if err := d.DB.Model(&role).Updates(updates).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to update role")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditRoleUpdate, shared.TargetTypeRole, role.ID, map[string]any{
			"id": role.ID,
		})

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"version": role.Version,
		})

		d.DB.First(&role, "id = ?", req.ID)
		broadcastEvent(d, shared.TypeEventRoleUpdated, roleToMap(&role))
	}
}

func handleRoleDelete(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageRoles) {
			return
		}

		var req struct {
			ID string `json:"id"`
		}
		if !parsePayload(msg, &req) || req.ID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "id is required")
			return
		}

		var role models.Role
		if err := d.DB.First(&role, "id = ?", req.ID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "role not found")
			return
		}

		if role.IsDefault {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrForbidden, "cannot delete the default role")
			return
		}

		// Position hierarchy enforcement
		if !d.Hot.IsOwner(client.PubKey) {
			callerPos := middleware.GetHighestRolePosition(d.DB, client.UserID)
			if role.Position >= callerPos {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "cannot delete roles at or above your position")
				return
			}
		}

		// Remove user-role assignments for this role
		d.DB.Where("role_id = ?", role.ID).Delete(&models.UserRole{})
		// Remove channel-role access entries
		d.DB.Where("role_id = ?", role.ID).Delete(&models.ChannelRoleAccess{})
		d.DB.Delete(&role)

		auditLog(d.DB, client.UserID, shared.AuditRoleDelete, shared.TargetTypeRole, role.ID, map[string]any{
			"name": role.Name,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		broadcastEvent(d, shared.TypeEventRoleDeleted, map[string]any{
			"id": role.ID,
		})
	}
}

func handleRoleAssign(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageRoles) {
			return
		}

		var req struct {
			PubKey string `json:"pubkey"`
			RoleID string `json:"role_id"`
		}
		if !parsePayload(msg, &req) || req.PubKey == "" || req.RoleID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "pubkey and role_id are required")
			return
		}

		var role models.Role
		if err := d.DB.First(&role, "id = ?", req.RoleID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "role not found")
			return
		}

		// Position hierarchy enforcement
		if !d.Hot.IsOwner(client.PubKey) {
			callerPos := middleware.GetHighestRolePosition(d.DB, client.UserID)
			if role.Position >= callerPos {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "cannot assign roles at or above your position")
				return
			}
		}

		target, err := getUserByPubKeyHex(d.DB, req.PubKey)
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "user not found")
			return
		}

		ur := models.UserRole{
			UserID: target.ID,
			RoleID: req.RoleID,
		}
		if err := d.DB.FirstOrCreate(&ur, ur).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrInternal, "failed to assign role")
			return
		}

		auditLog(d.DB, client.UserID, shared.AuditUserRoleAdd, shared.TargetTypeUser, target.ID, map[string]any{
			"pubkey":  req.PubKey,
			"role_id": req.RoleID,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		broadcastEvent(d, shared.TypeEventUserRoleAdded, map[string]any{
			"pubkey":  req.PubKey,
			"role_id": req.RoleID,
		})
	}
}

func handleRoleRevoke(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		if !checkPerm(d, client, msg.Type, msg.ID, shared.PermManageRoles) {
			return
		}

		var req struct {
			PubKey string `json:"pubkey"`
			RoleID string `json:"role_id"`
		}
		if !parsePayload(msg, &req) || req.PubKey == "" || req.RoleID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "pubkey and role_id are required")
			return
		}

		var role models.Role
		if err := d.DB.First(&role, "id = ?", req.RoleID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "role not found")
			return
		}

		// Position hierarchy enforcement
		if !d.Hot.IsOwner(client.PubKey) {
			callerPos := middleware.GetHighestRolePosition(d.DB, client.UserID)
			if role.Position >= callerPos {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "cannot revoke roles at or above your position")
				return
			}
		}

		target, err := getUserByPubKeyHex(d.DB, req.PubKey)
		if err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "user not found")
			return
		}

		d.DB.Where("user_id = ? AND role_id = ?", target.ID, req.RoleID).Delete(&models.UserRole{})

		auditLog(d.DB, client.UserID, shared.AuditUserRoleRemove, shared.TargetTypeUser, target.ID, map[string]any{
			"pubkey":  req.PubKey,
			"role_id": req.RoleID,
		})

		ws.SendOK(client, msg.Type, msg.ID, nil)

		broadcastEvent(d, shared.TypeEventUserRoleRemoved, map[string]any{
			"pubkey":  req.PubKey,
			"role_id": req.RoleID,
		})
	}
}

func roleToMap(r *models.Role) map[string]any {
	m := map[string]any{
		"id":          r.ID,
		"name":        r.Name,
		"position":    r.Position,
		"is_default":  r.IsDefault,
		"permissions": r.Permissions,
		"version":     r.Version,
	}
	if r.Color != nil {
		m["color"] = *r.Color
	}
	return m
}

func getActorPubKey(d *Deps, actorID *string) string {
	if actorID == nil {
		return ""
	}
	var user models.User
	if err := d.DB.Select("public_key").First(&user, "id = ?", *actorID).Error; err != nil {
		return ""
	}
	return hex.EncodeToString(user.PublicKey)
}
