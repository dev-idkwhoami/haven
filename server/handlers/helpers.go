package handlers

import (
	"encoding/hex"
	"encoding/json"
	"log/slog"

	"github.com/oklog/ulid/v2"

	"haven/server/config"
	"haven/server/middleware"
	"haven/server/models"
	"haven/server/ws"
	"haven/shared"

	"gorm.io/gorm"
)

// newULID generates a new ULID string.
func newULID() string {
	return ulid.Make().String()
}

// parsePayload unmarshals a WSMessage payload into the given struct.
func parsePayload(msg *ws.WSMessage, out any) bool {
	if msg.Payload == nil {
		return true
	}
	return json.Unmarshal(msg.Payload, out) == nil
}

// checkPerm verifies a client has the required permission. Sends an error and returns false if not.
func checkPerm(d *Deps, client *ws.Client, msgType, msgID string, perm int64) bool {
	if !middleware.CheckPermission(d.DB, d.Hot, client.UserID, client.PubKey, perm) {
		ws.SendError(client, msgType, msgID, shared.ErrPermissionDenied, "insufficient permissions")
		return false
	}
	return true
}

// auditLog creates an audit log entry.
func auditLog(db *gorm.DB, actorID, action, targetType, targetID string, details any) {
	var detailsStr *string
	if details != nil {
		b, err := json.Marshal(details)
		if err == nil {
			s := string(b)
			detailsStr = &s
		}
	}
	entry := models.AuditLogEntry{
		ID:         newULID(),
		ActorID:    &actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    detailsStr,
	}
	if err := db.Create(&entry).Error; err != nil {
		slog.Error("create audit log entry", "action", action, "error", err)
	}
}

// hasChannelAccess checks if a user (by ID) has access to a channel.
// Returns true if the channel has no role restrictions, or if the user holds a required role.
// Owners always have access.
func hasChannelAccess(db *gorm.DB, hot *config.HotConfig, userID string, userPubKey []byte, channelID string) bool {
	if hot.IsOwner(userPubKey) {
		return true
	}

	// Check if channel has role restrictions
	var count int64
	db.Model(&models.ChannelRoleAccess{}).Where("channel_id = ?", channelID).Count(&count)
	if count == 0 {
		return true // no restrictions
	}

	// Check if user has any of the required roles
	var accessCount int64
	db.Model(&models.ChannelRoleAccess{}).
		Joins("JOIN user_roles ON user_roles.role_id = channel_role_accesses.role_id").
		Where("channel_role_accesses.channel_id = ? AND user_roles.user_id = ?", channelID, userID).
		Count(&accessCount)
	return accessCount > 0
}

// getUserByPubKeyHex resolves a hex-encoded public key to a User record.
func getUserByPubKeyHex(db *gorm.DB, pubKeyHex string) (*models.User, error) {
	pubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, err
	}
	var user models.User
	if err := db.Where("public_key = ?", pubKey).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// getUserRoleIDs returns all role IDs assigned to a user.
func getUserRoleIDs(db *gorm.DB, userID string) []string {
	var userRoles []models.UserRole
	db.Where("user_id = ?", userID).Find(&userRoles)
	ids := make([]string, len(userRoles))
	for i, ur := range userRoles {
		ids[i] = ur.RoleID
	}
	return ids
}
