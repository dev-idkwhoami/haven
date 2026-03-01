package models

import "time"

// TrustedServer is the TOFU trust store — maps server addresses to their public keys.
type TrustedServer struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Address         string     `gorm:"column:address;type:text;uniqueIndex;not null" json:"address"`
	PublicKey       []byte     `gorm:"column:public_key;type:blob;not null" json:"publicKey"`
	Name            *string    `gorm:"column:name;type:text" json:"name"`
	Icon            []byte     `gorm:"column:icon;type:blob" json:"icon"`
	IconHash        *string    `gorm:"column:icon_hash;type:text" json:"iconHash"`
	SessionToken    *string    `gorm:"column:session_token;type:text" json:"sessionToken"`
	IsRelayOnly     bool       `gorm:"column:is_relay_only;not null;default:false" json:"isRelayOnly"`
	FirstTrustedAt  time.Time  `gorm:"column:first_trusted_at;not null" json:"firstTrustedAt"`
	LastConnectedAt *time.Time `gorm:"column:last_connected_at" json:"lastConnectedAt"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}
