package middleware

import (
	"haven/server/config"
	"haven/server/models"
	"haven/shared"

	"gorm.io/gorm"
)

// CheckPermission checks if a user has a specific permission.
// Server owners (from hot config) always pass.
func CheckPermission(db *gorm.DB, hot *config.HotConfig, userID string, userPubKey []byte, required int64) bool {
	if hot.IsOwner(userPubKey) {
		return true
	}
	return checkUserPermission(db, userID, required)
}

// checkUserPermission computes effective permissions from all assigned roles and checks the required bit.
func checkUserPermission(db *gorm.DB, userID string, required int64) bool {
	var roles []models.Role
	err := db.Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return false
	}

	var effective int64
	for _, r := range roles {
		effective |= r.Permissions
	}
	return shared.HasPermission(effective, required)
}

// GetEffectivePermissions computes the OR of all role permissions for a user.
func GetEffectivePermissions(db *gorm.DB, userID string) int64 {
	var roles []models.Role
	err := db.Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return 0
	}

	var effective int64
	for _, r := range roles {
		effective |= r.Permissions
	}
	return effective
}

// GetHighestRolePosition returns the highest role position held by a user.
func GetHighestRolePosition(db *gorm.DB, userID string) int {
	var roles []models.Role
	err := db.Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil || len(roles) == 0 {
		return -1
	}

	highest := roles[0].Position
	for _, r := range roles[1:] {
		if r.Position > highest {
			highest = r.Position
		}
	}
	return highest
}
