package models

import "time"

// CachedMessage stores locally cached messages from server channels.
type CachedMessage struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ServerID        int64      `gorm:"column:server_id;index;not null" json:"serverId"`
	RemoteMessageID string     `gorm:"column:remote_message_id;type:text;index;not null" json:"remoteMessageId"`
	ChannelID       string     `gorm:"column:channel_id;type:text;index;not null" json:"channelId"`
	AuthorPubKey    []byte     `gorm:"column:author_pub_key;type:blob;index" json:"authorPubKey"`
	Content         string     `gorm:"column:content;type:text;not null" json:"content"`
	Signature       []byte     `gorm:"column:signature;type:blob;not null" json:"signature"`
	Nonce           []byte     `gorm:"column:nonce;type:blob;not null" json:"nonce"`
	EditedAt        *time.Time `gorm:"column:edited_at" json:"editedAt"`
	RemoteCreatedAt time.Time  `gorm:"column:remote_created_at;not null" json:"remoteCreatedAt"`
	Version         int64      `gorm:"column:version;not null;default:0" json:"version"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}
