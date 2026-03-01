package models

import "time"

// AccessRequest tracks a non-allowlisted user's request to join an allowlist-mode server.
type AccessRequest struct {
	ID          string `gorm:"primaryKey"`
	PublicKey   []byte `gorm:"index"`
	DisplayName string
	Message     *string
	Status      string  `gorm:"index;default:pending"` // "pending" | "approved" | "rejected"
	ReviewedBy  *string // user ID of the admin who approved/rejected
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
