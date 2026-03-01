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

// AuditEntry is the frontend-facing audit log entry.
type AuditEntry struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	ActorKey  string `json:"actorKey"`
	ActorName string `json:"actorName"`
	Details   string `json:"details"`
	Timestamp string `json:"timestamp"`
}

// AuditPage is a paginated audit log response.
type AuditPage struct {
	Entries []AuditEntry `json:"entries"`
	HasMore bool         `json:"hasMore"`
}

// InviteInfo is the frontend-facing invite code data.
type InviteInfo struct {
	Code      string `json:"code"`
	CreatedBy string `json:"createdBy"`
	UsesLeft  *int   `json:"usesLeft"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	CreatedAt string `json:"createdAt"`
}

// AccessRequestInfo is the frontend-facing access request data.
type AccessRequestInfo struct {
	ID          string `json:"id"`
	PubKey      string `json:"pubKey"`
	DisplayName string `json:"displayName"`
	Message     string `json:"message"`
	IsOnline    bool   `json:"isOnline"`
	CreatedAt   string `json:"createdAt"`
}

// AdminService manages admin operations: audit log, invites, and server settings.
type AdminService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
}

// NewAdminService creates a new AdminService.
func NewAdminService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *AdminService {
	return &AdminService{
		db:      db,
		manager: manager,
		privKey: privKey,
	}
}

// SetContext is called by Wails during startup.
func (a *AdminService) SetContext(ctx context.Context) {
	a.ctx = ctx
}

// GetAuditLog fetches audit log entries with cursor pagination.
func (a *AdminService) GetAuditLog(serverID int64, cursor string, limit int) (AuditPage, error) {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return AuditPage{}, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{}
	if cursor != "" {
		payload["before"] = cursor
	}
	if limit > 0 {
		payload["limit"] = limit
	}

	resp, err := conn.Request(shared.TypeAuditList, payload)
	if err != nil {
		return AuditPage{}, fmt.Errorf("audit.list: %w", err)
	}

	var result struct {
		Entries []auditEntryWS `json:"entries"`
		HasMore bool           `json:"has_more"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return AuditPage{}, fmt.Errorf("unmarshal audit: %w", err)
	}

	entries := make([]AuditEntry, len(result.Entries))
	for i, we := range result.Entries {
		entries[i] = AuditEntry{
			ID:        we.ID,
			Action:    we.Action,
			ActorKey:  we.ActorPubKey,
			Details:   we.Details,
			Timestamp: we.CreatedAt,
		}
	}

	return AuditPage{
		Entries: entries,
		HasMore: result.HasMore,
	}, nil
}

// CreateInvite creates a new invite code.
func (a *AdminService) CreateInvite(serverID int64, usesLeft *int, expiresAt string) (InviteInfo, error) {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return InviteInfo{}, fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{}
	if usesLeft != nil {
		payload["uses_left"] = *usesLeft
	}
	if expiresAt != "" {
		payload["expires_at"] = expiresAt
	}

	resp, err := conn.Request(shared.TypeInviteCreate, payload)
	if err != nil {
		return InviteInfo{}, fmt.Errorf("invite.create: %w", err)
	}

	var result struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return InviteInfo{}, fmt.Errorf("unmarshal invite: %w", err)
	}

	return InviteInfo{
		Code:     result.Code,
		UsesLeft: usesLeft,
	}, nil
}

// GetInvites lists all invite codes.
func (a *AdminService) GetInvites(serverID int64) ([]InviteInfo, error) {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeInviteList, nil)
	if err != nil {
		return nil, fmt.Errorf("invite.list: %w", err)
	}

	var result struct {
		Invites []inviteWS `json:"invites"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal invites: %w", err)
	}

	invites := make([]InviteInfo, len(result.Invites))
	for i, wi := range result.Invites {
		invites[i] = InviteInfo{
			Code:      wi.Code,
			CreatedBy: wi.CreatedByPubKey,
			UsesLeft:  wi.UsesLeft,
			ExpiresAt: wi.ExpiresAt,
			CreatedAt: wi.CreatedAt,
		}
	}
	return invites, nil
}

// RevokeInvite revokes an invite code.
func (a *AdminService) RevokeInvite(serverID int64, code string) error {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeInviteRevoke, map[string]interface{}{
		"id": code,
	})
	if err != nil {
		return fmt.Errorf("invite.revoke: %w", err)
	}
	return nil
}

// UpdateServer updates server settings.
func (a *AdminService) UpdateServer(serverID int64, name string, description string) error {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{}
	if name != "" {
		payload["name"] = name
	}
	if description != "" {
		payload["description"] = description
	}

	_, err = conn.Request(shared.TypeServerUpdate, payload)
	if err != nil {
		return fmt.Errorf("server.update: %w", err)
	}
	return nil
}

// SetServerIcon uploads and sets the server icon.
func (a *AdminService) SetServerIcon(serverID int64, filePath string) error {
	// This uses FileService internally to upload, then calls server.update with icon_id.
	// For now, this is a placeholder — the full implementation would:
	// 1. Upload file via FileService.Upload(serverID, "", filePath) (no channel = icon)
	// 2. Call server.update with the returned file_id as icon_id
	return fmt.Errorf("SetServerIcon requires FileService integration")
}

// GetAccessRequests lists pending access requests.
func (a *AdminService) GetAccessRequests(serverID int64) ([]AccessRequestInfo, error) {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeAccessRequestList, nil)
	if err != nil {
		return nil, fmt.Errorf("access_request.list: %w", err)
	}

	var result struct {
		Requests []accessRequestWS `json:"requests"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal access requests: %w", err)
	}

	out := make([]AccessRequestInfo, len(result.Requests))
	for i, r := range result.Requests {
		out[i] = AccessRequestInfo{
			ID:          r.ID,
			PubKey:      r.PubKey,
			DisplayName: r.DisplayName,
			Message:     r.Message,
			IsOnline:    r.IsOnline,
			CreatedAt:   r.CreatedAt,
		}
	}
	return out, nil
}

// ApproveAccessRequest approves a pending access request.
func (a *AdminService) ApproveAccessRequest(serverID int64, requestID string) error {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeAccessRequestApprove, map[string]interface{}{
		"id": requestID,
	})
	if err != nil {
		return fmt.Errorf("access_request.approve: %w", err)
	}
	return nil
}

// RejectAccessRequest rejects a pending access request.
func (a *AdminService) RejectAccessRequest(serverID int64, requestID string) error {
	conn, err := a.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeAccessRequestReject, map[string]interface{}{
		"id": requestID,
	})
	if err != nil {
		return fmt.Errorf("access_request.reject: %w", err)
	}
	return nil
}

// auditEntryWS is the wire format from the server.
type auditEntryWS struct {
	ID          string `json:"id"`
	ActorPubKey string `json:"actor_pubkey"`
	Action      string `json:"action"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Details     string `json:"details"`
	CreatedAt   string `json:"created_at"`
}

// inviteWS is the wire format from the server.
type inviteWS struct {
	ID              string `json:"id"`
	Code            string `json:"code"`
	UsesLeft        *int   `json:"uses_left"`
	ExpiresAt       string `json:"expires_at"`
	CreatedByPubKey string `json:"created_by_pubkey"`
	CreatedAt       string `json:"created_at"`
}

// accessRequestWS is the wire format from the server.
type accessRequestWS struct {
	ID          string `json:"id"`
	PubKey      string `json:"pubkey"`
	DisplayName string `json:"display_name"`
	Message     string `json:"message"`
	IsOnline    bool   `json:"is_online"`
	CreatedAt   string `json:"created_at"`
}
