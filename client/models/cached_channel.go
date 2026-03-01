package models

import "time"

// CachedChannel stores cached server channels for offline sidebar rendering.
// Unique constraint: (ServerID, RemoteChannelID).
type CachedChannel struct {
	ID                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ServerID          int64     `gorm:"column:server_id;uniqueIndex:idx_cached_channel_server_remote;not null" json:"serverId"`
	RemoteChannelID   string    `gorm:"column:remote_channel_id;type:text;uniqueIndex:idx_cached_channel_server_remote;not null" json:"remoteChannelId"`
	RemoteCategoryID  string    `gorm:"column:remote_category_id;type:text;not null" json:"remoteCategoryId"`
	Name              string    `gorm:"column:name;type:text;not null" json:"name"`
	Type              string    `gorm:"column:type;type:text;not null;default:'text'" json:"type"`
	Position          int       `gorm:"column:position;not null;default:0" json:"position"`
	LastReadMessageID *string   `gorm:"column:last_read_message_id;type:text" json:"lastReadMessageId"`
	Version           int64     `gorm:"column:version;not null;default:0" json:"version"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"updatedAt"`
}
