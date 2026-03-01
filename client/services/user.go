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

// User is the frontend-facing user profile data.
type User struct {
	PubKey      string   `json:"pubKey"`
	DisplayName string   `json:"displayName"`
	AvatarHash  string   `json:"avatarHash"`
	Bio         string   `json:"bio"`
	Status      string   `json:"status"`
	RoleIDs     []string `json:"roleIds"`
}

// Ban is the frontend-facing ban data.
type Ban struct {
	ID             string `json:"id"`
	PubKey         string `json:"pubKey"`
	Reason         string `json:"reason"`
	BannedByPubKey string `json:"bannedByPubKey"`
	ExpiresAt      string `json:"expiresAt"`
	CreatedAt      string `json:"createdAt"`
}

// UserService manages user profiles, presence, and moderation.
type UserService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
}

// NewUserService creates a new UserService.
func NewUserService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *UserService {
	return &UserService{
		db:      db,
		manager: manager,
		privKey: privKey,
	}
}

// SetContext is called by Wails during startup.
func (u *UserService) SetContext(ctx context.Context) {
	u.ctx = ctx
}

// GetUsers returns all members visible to the requesting user.
func (u *UserService) GetUsers(serverID int64) ([]User, error) {
	conn, err := u.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeUserList, nil)
	if err != nil {
		return nil, fmt.Errorf("user.list: %w", err)
	}

	var result struct {
		Users []userWS `json:"users"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal users: %w", err)
	}

	users := make([]User, len(result.Users))
	for i, wu := range result.Users {
		users[i] = wu.toUser()
	}
	return users, nil
}

// GetUser returns a single user's profile.
func (u *UserService) GetUser(serverID int64, pubKey string) (User, error) {
	conn, err := u.manager.Get(serverID)
	if err != nil {
		return User{}, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeUserProfile, map[string]interface{}{
		"pubkey": pubKey,
	})
	if err != nil {
		return User{}, fmt.Errorf("user.profile: %w", err)
	}

	var wu userWS
	if err := json.Unmarshal(resp.Payload, &wu); err != nil {
		return User{}, fmt.Errorf("unmarshal user: %w", err)
	}
	return wu.toUser(), nil
}

// KickUser kicks a user from the server.
func (u *UserService) KickUser(serverID int64, pubKey string) error {
	conn, err := u.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeUserKick, map[string]interface{}{
		"pubkey": pubKey,
	})
	if err != nil {
		return fmt.Errorf("user.kick: %w", err)
	}
	return nil
}

// BanUser bans a user from the server.
func (u *UserService) BanUser(serverID int64, pubKey string, reason string, expiresAt string) error {
	conn, err := u.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	payload := map[string]interface{}{
		"pubkey": pubKey,
	}
	if reason != "" {
		payload["reason"] = reason
	}
	if expiresAt != "" {
		payload["expires_at"] = expiresAt
	}

	_, err = conn.Request(shared.TypeBanCreate, payload)
	if err != nil {
		return fmt.Errorf("ban.create: %w", err)
	}
	return nil
}

// UnbanUser removes a ban.
func (u *UserService) UnbanUser(serverID int64, pubKey string) error {
	conn, err := u.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeBanRemove, map[string]interface{}{
		"pubkey": pubKey,
	})
	if err != nil {
		return fmt.Errorf("ban.remove: %w", err)
	}
	return nil
}

// GetBans lists all bans on the server.
func (u *UserService) GetBans(serverID int64) ([]Ban, error) {
	conn, err := u.manager.Get(serverID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	resp, err := conn.Request(shared.TypeBanList, nil)
	if err != nil {
		return nil, fmt.Errorf("ban.list: %w", err)
	}

	var result struct {
		Bans []banWS `json:"bans"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("unmarshal bans: %w", err)
	}

	bans := make([]Ban, len(result.Bans))
	for i, wb := range result.Bans {
		bans[i] = Ban{
			ID:             wb.ID,
			PubKey:         wb.PubKey,
			Reason:         wb.Reason,
			BannedByPubKey: wb.BannedByPubKey,
			ExpiresAt:      wb.ExpiresAt,
			CreatedAt:      wb.CreatedAt,
		}
	}
	return bans, nil
}

// SetStatus updates the user's status on a specific server.
func (u *UserService) SetStatus(serverID int64, status string) error {
	conn, err := u.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	_, err = conn.Request(shared.TypeUserUpdate, map[string]interface{}{
		"status": status,
	})
	if err != nil {
		return fmt.Errorf("user.update: %w", err)
	}
	return nil
}

// SetStatusAll updates the user's status on all connected servers.
func (u *UserService) SetStatusAll(status string) error {
	ids := u.manager.AllConnected()
	var firstErr error
	for _, id := range ids {
		if err := u.SetStatus(id, status); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// userWS is the wire format from the server.
type userWS struct {
	PubKey      string   `json:"pubkey"`
	DisplayName string   `json:"display_name"`
	AvatarHash  string   `json:"avatar_hash"`
	Bio         string   `json:"bio"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
	Version     int64    `json:"version"`
}

func (wu userWS) toUser() User {
	roles := wu.Roles
	if roles == nil {
		roles = []string{}
	}
	return User{
		PubKey:      wu.PubKey,
		DisplayName: wu.DisplayName,
		AvatarHash:  wu.AvatarHash,
		Bio:         wu.Bio,
		Status:      wu.Status,
		RoleIDs:     roles,
	}
}

// banWS is the wire format from the server.
type banWS struct {
	ID             string `json:"id"`
	PubKey         string `json:"pubkey"`
	Reason         string `json:"reason"`
	BannedByPubKey string `json:"banned_by_pubkey"`
	ExpiresAt      string `json:"expires_at"`
	CreatedAt      string `json:"created_at"`
}
