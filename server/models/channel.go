package models

import "time"

// Category is a channel grouping container, as seen in the sidebar.
// HasMany: Channel
type Category struct {
	ID        string    `gorm:"column:id;type:text;primaryKey"`
	Name      string    `gorm:"column:name;type:text;not null"`
	Position  int       `gorm:"column:position;not null;default:0"`
	Type      string    `gorm:"column:type;type:text;not null;default:'text'"`
	Version   int64     `gorm:"column:version;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// Channel is a text or voice channel belonging to a category.
// HasMany: Message, ChannelRoleAccess
type Channel struct {
	ID          string    `gorm:"column:id;type:text;primaryKey"`
	CategoryID  string    `gorm:"column:category_id;type:text;index;not null"`
	Name        string    `gorm:"column:name;type:text;not null"`
	Type        string    `gorm:"column:type;type:text;not null;default:'text'"`
	Position    int       `gorm:"column:position;not null;default:0"`
	OpusBitrate *int      `gorm:"column:opus_bitrate"`
	Version     int64     `gorm:"column:version;not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// ChannelRoleAccess is the join table controlling which roles can access a channel.
// No rows for a channel means open to everyone. Any rows means restricted.
type ChannelRoleAccess struct {
	ChannelID string `gorm:"column:channel_id;type:text;primaryKey"`
	RoleID    string `gorm:"column:role_id;type:text;primaryKey"`
}
