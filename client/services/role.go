package services

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"haven/client/connection"
	"haven/shared"
)

// Role is the frontend-facing role data.
type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Position    int    `json:"position"`
	IsDefault   bool   `json:"isDefault"`
	Permissions int64  `json:"permissions"`
}

// RoleService manages role CRUD and assignment.
type RoleService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
}

// NewRoleService creates a new RoleService.
func NewRoleService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *RoleService {
	return &RoleService{
		db:      db,
		manager: manager,
		privKey: privKey,
	}
}

// SetContext is called by Wails during startup.
func (r *RoleService) SetContext(ctx context.Context) {
	r.ctx = ctx
}

// GetRoles returns all roles for a server.
func (r *RoleService) GetRoles(serverID int64) ([]Role, error) {
	conn, err := r.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeRoleList, nil)
	if err != nil {
		return nil, fmt.Errorf("role.list: %w", err)
	}

	var result struct {
		Roles []roleWS `json:"roles"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal roles: %w", err)
	}

	roles := make([]Role, len(result.Roles))
	for i, wr := range result.Roles {
		roles[i] = wr.toRole()
	}
	return roles, nil
}

// CreateRole creates a new role on a server.
func (r *RoleService) CreateRole(serverID int64, name string, color string, permissions int64) (Role, error) {
	conn, err := r.manager.Get(serverID)
	if err != nil {
		return Role{}, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{
		"name":        name,
		"permissions": permissions,
	}
	if color != "" {
		payload["color"] = color
	}

	resp, err := conn.Request(shared.TypeRoleCreate, payload)
	if err != nil {
		return Role{}, fmt.Errorf("role.create: %w", err)
	}

	var created struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
		Version  int64  `json:"version"`
	}
	if err := json.Unmarshal(resp.Payload, &created); err != nil {
		return Role{}, fmt.Errorf("unmarshal role: %w", err)
	}

	return Role{
		ID:          created.ID,
		Name:        name,
		Color:       color,
		Position:    created.Position,
		Permissions: permissions,
	}, nil
}

// UpdateRole updates a role on a server.
func (r *RoleService) UpdateRole(serverID int64, id string, name string, color string, position int, permissions int64) error {
	conn, err := r.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{
		"id":          id,
		"name":        name,
		"color":       color,
		"position":    position,
		"permissions": permissions,
	}

	_, err = conn.Request(shared.TypeRoleUpdate, payload)
	if err != nil {
		return fmt.Errorf("role.update: %w", err)
	}
	return nil
}

// DeleteRole deletes a role.
func (r *RoleService) DeleteRole(serverID int64, id string) error {
	conn, err := r.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeRoleDelete, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("role.delete: %w", err)
	}
	return nil
}

// AssignRole assigns a role to a user.
func (r *RoleService) AssignRole(serverID int64, pubKey string, roleID string) error {
	conn, err := r.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeRoleAssign, map[string]interface{}{
		"pubkey":  pubKey,
		"role_id": roleID,
	})
	if err != nil {
		return fmt.Errorf("role.assign: %w", err)
	}
	return nil
}

// RevokeRole removes a role from a user.
func (r *RoleService) RevokeRole(serverID int64, pubKey string, roleID string) error {
	conn, err := r.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeRoleRevoke, map[string]interface{}{
		"pubkey":  pubKey,
		"role_id": roleID,
	})
	if err != nil {
		return fmt.Errorf("role.revoke: %w", err)
	}
	return nil
}

// roleWS is the wire format from the server.
type roleWS struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Position    int    `json:"position"`
	IsDefault   bool   `json:"is_default"`
	Permissions int64  `json:"permissions"`
	Version     int64  `json:"version"`
}

func (wr roleWS) toRole() Role {
	return Role{
		ID:          wr.ID,
		Name:        wr.Name,
		Color:       wr.Color,
		Position:    wr.Position,
		IsDefault:   wr.IsDefault,
		Permissions: wr.Permissions,
	}
}
