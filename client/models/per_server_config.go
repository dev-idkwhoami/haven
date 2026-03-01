package models

import "time"

// PerServerConfig stores per-server client preferences and field selection overrides.
type PerServerConfig struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ServerID    int64     `gorm:"column:server_id;uniqueIndex;not null" json:"serverId"`
	SyncAvatars bool      `gorm:"column:sync_avatars;not null;default:true" json:"syncAvatars"`
	SyncBios    bool      `gorm:"column:sync_bios;not null;default:true" json:"syncBios"`
	SyncStatus  bool      `gorm:"column:sync_status;not null;default:true" json:"syncStatus"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}
