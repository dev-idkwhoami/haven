package models

import "time"

// CachedUser stores cached remote user profiles from servers.
// Unique constraint: (ServerID, PublicKey).
type CachedUser struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ServerID    int64     `gorm:"column:server_id;uniqueIndex:idx_cached_user_server_pubkey;not null" json:"serverId"`
	PublicKey   []byte    `gorm:"column:public_key;type:blob;uniqueIndex:idx_cached_user_server_pubkey;not null" json:"publicKey"`
	DisplayName string    `gorm:"column:display_name;type:text;not null" json:"displayName"`
	Avatar      []byte    `gorm:"column:avatar;type:blob" json:"avatar"`
	AvatarHash  *string   `gorm:"column:avatar_hash;type:text" json:"avatarHash"`
	Bio         *string   `gorm:"column:bio;type:text" json:"bio"`
	Version     int64     `gorm:"column:version;not null;default:0" json:"version"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}
