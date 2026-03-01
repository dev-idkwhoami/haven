package models

import "time"

// Ban tracks banned users. Keyed on PublicKey because the User record may be deleted.
type Ban struct {
	ID        string     `gorm:"column:id;type:text;primaryKey"`
	PublicKey []byte     `gorm:"column:public_key;type:blob;index;not null"`
	Reason    *string    `gorm:"column:reason;type:text"`
	BannedBy  *string    `gorm:"column:banned_by;type:text"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

// AuditLogEntry records admin/moderator actions. Append-only, never modified.
type AuditLogEntry struct {
	ID         string    `gorm:"column:id;type:text;primaryKey"`
	ActorID    *string   `gorm:"column:actor_id;type:text"`
	Action     string    `gorm:"column:action;type:text;not null"`
	TargetType string    `gorm:"column:target_type;type:text;not null"`
	TargetID   string    `gorm:"column:target_id;type:text;not null"`
	Details    *string   `gorm:"column:details;type:text"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// InviteCode is for servers with AccessMode = "invite". Single-use or multi-use codes.
type InviteCode struct {
	ID        string     `gorm:"column:id;type:text;primaryKey"`
	Code      string     `gorm:"column:code;type:text;uniqueIndex;not null"`
	CreatedBy *string    `gorm:"column:created_by;type:text"`
	UsesLeft  *int       `gorm:"column:uses_left"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

// ErasureRecord tracks Ghost/Forget Me departures for client-side cache propagation.
// Append-only, never modified or deleted.
type ErasureRecord struct {
	ID        string    `gorm:"column:id;type:text;primaryKey"`
	PublicKey []byte    `gorm:"column:public_key;type:blob;index;not null"`
	Mode      string    `gorm:"column:mode;type:text;not null"`
	ErasedAt  time.Time `gorm:"column:erased_at;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
