package models

import "time"

// Session tracks active session tokens for HTTP request authentication.
type Session struct {
	ID        string    `gorm:"column:id;type:text;primaryKey"`
	UserID    *string   `gorm:"column:user_id;type:text;index"`
	Token     string    `gorm:"column:token;type:text;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}
