package models

import "time"

// Server holds server metadata and runtime configuration (singleton row).
// Static config (owner pubkeys, port, DB connection) lives in the config file.
type Server struct {
	ID                string    `gorm:"column:id;type:text;primaryKey"`
	Name              string    `gorm:"column:name;type:text;not null"`
	Description       *string   `gorm:"column:description;type:text"`
	Icon              *string   `gorm:"column:icon;type:text"`
	IconHash          string    `gorm:"column:icon_hash;type:text;not null;default:''"`
	AccessMode        string    `gorm:"column:access_mode;type:text;not null;default:'open'"`
	AccessPassword    *string   `gorm:"column:access_password;type:text"`
	MaxFileSize       int64     `gorm:"column:max_file_size;not null;default:52428800"`
	TotalStorageLimit int64     `gorm:"column:total_storage_limit;not null;default:21474836480"`
	DefaultChannelID  *string   `gorm:"column:default_channel_id;type:text"`
	WelcomeMessage    *string   `gorm:"column:welcome_message;type:text"`
	Version           int64     `gorm:"column:version;not null;default:0"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}
