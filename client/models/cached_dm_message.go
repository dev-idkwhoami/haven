package models

import "time"

// CachedDMMessage stores locally cached decrypted DM messages.
// Safe because SQLCipher encrypts the entire DB at rest.
type CachedDMMessage struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConversationID  string    `gorm:"column:conversation_id;type:text;index;not null" json:"conversationId"`
	SenderPubKey    []byte    `gorm:"column:sender_pub_key;type:blob;index" json:"senderPubKey"`
	Content         string    `gorm:"column:content;type:text;not null" json:"content"`
	Signature       []byte    `gorm:"column:signature;type:blob;not null" json:"signature"`
	Nonce           []byte    `gorm:"column:nonce;type:blob;not null" json:"nonce"`
	RemoteCreatedAt time.Time `gorm:"column:remote_created_at;not null" json:"remoteCreatedAt"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updatedAt"`
}
