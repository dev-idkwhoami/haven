package auth

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	servercrypto "haven/server/crypto"
	"haven/server/models"

	"github.com/oklog/ulid/v2"
)

// CreateSession creates a new session for the given user and returns the token.
func CreateSession(db *gorm.DB, userID string) (string, error) {
	token, err := servercrypto.GenerateSessionToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	session := models.Session{
		ID:        ulid.Make().String(),
		UserID:    &userID,
		Token:     token,
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour), // far future while WS is active
	}
	if err := db.Create(&session).Error; err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

// ValidateSession checks if a session token is valid and returns the associated user ID.
func ValidateSession(db *gorm.DB, token string) (string, error) {
	var session models.Session
	err := db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&session).Error
	if err != nil {
		return "", fmt.Errorf("validate session: %w", err)
	}
	if session.UserID == nil {
		return "", fmt.Errorf("session has no user")
	}
	return *session.UserID, nil
}

// InvalidateSession deletes a session by token.
func InvalidateSession(db *gorm.DB, token string) error {
	result := db.Where("token = ?", token).Delete(&models.Session{})
	if result.Error != nil {
		return fmt.Errorf("invalidate session: %w", result.Error)
	}
	return nil
}

// UpdateGracePeriod sets the session expiry to now + gracePeriod seconds.
func UpdateGracePeriod(db *gorm.DB, token string, gracePeriod int) error {
	expires := time.Now().Add(time.Duration(gracePeriod) * time.Second)
	result := db.Model(&models.Session{}).Where("token = ?", token).Update("expires_at", expires)
	if result.Error != nil {
		return fmt.Errorf("update grace period: %w", result.Error)
	}
	return nil
}

// CleanExpiredSessions deletes all sessions past their expiry time.
func CleanExpiredSessions(db *gorm.DB) error {
	result := db.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return fmt.Errorf("clean expired sessions: %w", result.Error)
	}
	return nil
}
