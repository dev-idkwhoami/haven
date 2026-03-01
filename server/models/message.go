package models

import "time"

// Message is a message in a text or voice channel.
// ULID as PK provides inherent chronological ordering.
type Message struct {
	ID        string     `gorm:"column:id;type:text;primaryKey"`
	ChannelID string     `gorm:"column:channel_id;type:text;index;not null"`
	AuthorID  string     `gorm:"column:author_id;type:text;index;not null"`
	Content   string     `gorm:"column:content;type:text;not null"`
	Signature []byte     `gorm:"column:signature;type:blob;not null"`
	Nonce     []byte     `gorm:"column:nonce;type:blob;not null"`
	EditedAt  *time.Time `gorm:"column:edited_at"`
	Version   int64      `gorm:"column:version;not null;default:0"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

// File holds uploaded file metadata. Created on upload completion.
type File struct {
	ID          string    `gorm:"column:id;type:text;primaryKey"`
	UploaderID  string    `gorm:"column:uploader_id;type:text;index;not null"`
	ChannelID   *string   `gorm:"column:channel_id;type:text;index"`
	Name        string    `gorm:"column:name;type:text;not null"`
	MimeType    string    `gorm:"column:mime_type;type:text;not null"`
	Size        int64     `gorm:"column:size;not null"`
	StoragePath string    `gorm:"column:storage_path;type:text;not null"`
	ThumbPath   *string   `gorm:"column:thumb_path;type:text"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// MessageFile is the join table linking messages to file attachments (many-to-many).
type MessageFile struct {
	MessageID string `gorm:"column:message_id;type:text;primaryKey"`
	FileID    string `gorm:"column:file_id;type:text;primaryKey"`
}
