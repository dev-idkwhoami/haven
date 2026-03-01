package models

import "time"

// Role is a named permission set assigned to users. Permissions stored as a bitfield.
// HasMany: UserRole, ChannelRoleAccess
type Role struct {
	ID          string    `gorm:"column:id;type:text;primaryKey"`
	Name        string    `gorm:"column:name;type:text;uniqueIndex;not null"`
	Color       *string   `gorm:"column:color;type:text"`
	Position    int       `gorm:"column:position;not null;default:0"`
	IsDefault   bool      `gorm:"column:is_default;not null;default:false"`
	Permissions int64     `gorm:"column:permissions;not null;default:0"`
	Version     int64     `gorm:"column:version;not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// UserRole is the join table assigning roles to users (many-to-many).
type UserRole struct {
	UserID string `gorm:"column:user_id;type:text;primaryKey"`
	RoleID string `gorm:"column:role_id;type:text;primaryKey"`
}
