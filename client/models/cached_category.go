package models

import "time"

// CachedCategory stores cached server categories for offline sidebar rendering.
// Unique constraint: (ServerID, RemoteCategoryID).
type CachedCategory struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ServerID         int64     `gorm:"column:server_id;uniqueIndex:idx_cached_category_server_remote;not null" json:"serverId"`
	RemoteCategoryID string    `gorm:"column:remote_category_id;type:text;uniqueIndex:idx_cached_category_server_remote;not null" json:"remoteCategoryId"`
	Name             string    `gorm:"column:name;type:text;not null" json:"name"`
	Position         int       `gorm:"column:position;not null;default:0" json:"position"`
	Type             string    `gorm:"column:type;type:text;not null;default:'text'" json:"type"`
	Version          int64     `gorm:"column:version;not null;default:0" json:"version"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updatedAt"`
}
