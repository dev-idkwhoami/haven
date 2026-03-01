package models

import "time"

// DMConversation is a unified model for both 1:1 DMs and group DMs.
// The server acts as a blind relay for encrypted messages.
// HasMany: DMParticipant, DMMessage
type DMConversation struct {
	ID        string    `gorm:"column:id;type:text;primaryKey"`
	IsGroup   bool      `gorm:"column:is_group;not null;default:false"`
	Name      *string   `gorm:"column:name;type:text"`
	CreatedBy *string   `gorm:"column:created_by;type:text"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// DMParticipant tracks membership in a DM conversation.
type DMParticipant struct {
	ConversationID string     `gorm:"column:conversation_id;type:text;primaryKey"`
	UserID         string     `gorm:"column:user_id;type:text;primaryKey"`
	IsKeyManager   bool       `gorm:"column:is_key_manager;not null;default:false"`
	JoinedAt       time.Time  `gorm:"column:joined_at;not null"`
	LeftAt         *time.Time `gorm:"column:left_at"`
}

// DMMessage is an encrypted DM message blob. The server stores and forwards but cannot read.
// Append-only: no UpdatedAt.
type DMMessage struct {
	ID               string    `gorm:"column:id;type:text;primaryKey"`
	ConversationID   string    `gorm:"column:conversation_id;type:text;index;not null"`
	SenderID         string    `gorm:"column:sender_id;type:text;not null"`
	EncryptedPayload []byte    `gorm:"column:encrypted_payload;type:blob;not null"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}
