package models

import "time"

// LocalProfile is the user's own identity and profile. Singleton row.
type LocalProfile struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PublicKey   []byte    `gorm:"column:public_key;type:blob;uniqueIndex;not null" json:"publicKey"`
	DisplayName string    `gorm:"column:display_name;type:text;not null" json:"displayName"`
	Avatar      []byte    `gorm:"column:avatar;type:blob" json:"avatar"`
	AvatarHash  *string   `gorm:"column:avatar_hash;type:text" json:"avatarHash"`
	Bio         *string   `gorm:"column:bio;type:text" json:"bio"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updatedAt"`
}
