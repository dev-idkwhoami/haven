package models

import "time"

// SentinelUserID is the ULID for the Ghost Mode placeholder user (26-char zero ULID).
const SentinelUserID = "00000000000000000000000000"

// SentinelPublicKey is a 32-byte zero Ed25519 public key for the sentinel user.
var SentinelPublicKey = make([]byte, 32)

// User represents an authenticated user's identity and profile on this server.
// HasMany: UserRole, Message, File, Session
type User struct {
	ID          string    `gorm:"column:id;type:text;primaryKey"`
	PublicKey   []byte    `gorm:"column:public_key;type:blob;uniqueIndex;not null"`
	DisplayName string    `gorm:"column:display_name;type:text;not null"`
	Avatar      *string   `gorm:"column:avatar;type:text"`
	AvatarHash  string    `gorm:"column:avatar_hash;type:text;not null;default:''"`
	Bio         *string   `gorm:"column:bio;type:text"`
	Status      string    `gorm:"column:status;type:text;not null;default:'offline'"`
	Version     int64     `gorm:"column:version;not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}
