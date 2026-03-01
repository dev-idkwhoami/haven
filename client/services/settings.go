package services

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"gorm.io/gorm"

	"haven/client/connection"
	"haven/client/models"
)

// PerServerSettings is the frontend-facing per-server configuration.
type PerServerSettings struct {
	ServerID     int64 `json:"serverId"`
	ShowAvatars  bool  `json:"showAvatars"`
	ShowBios     bool  `json:"showBios"`
	ShowStatuses bool  `json:"showStatuses"`
}

// AppSettings is the frontend-facing application settings.
type AppSettings struct {
	Theme          string `json:"theme"`
	NotifySound    bool   `json:"notifySound"`
	NotifyDesktop  bool   `json:"notifyDesktop"`
	MinimizeToTray bool   `json:"minimizeToTray"`
}

// SettingsService manages app and per-server settings.
type SettingsService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *SettingsService {
	return &SettingsService{
		db:      db,
		manager: manager,
		privKey: privKey,
	}
}

// SetContext is called by Wails during startup.
func (s *SettingsService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// GetAppSettings returns the application settings.
func (s *SettingsService) GetAppSettings() (AppSettings, error) {
	// App settings are stored as a simple config. For now, return defaults.
	// A full implementation would store these in a dedicated SQLCipher table.
	return AppSettings{
		Theme:          "dark",
		NotifySound:    true,
		NotifyDesktop:  true,
		MinimizeToTray: false,
	}, nil
}

// UpdateAppSettings updates the application settings.
func (s *SettingsService) UpdateAppSettings(settings AppSettings) error {
	// Store in DB. Placeholder — would write to an AppConfig table.
	emitEvent(s.ctx, "settings:appChanged", settings)
	return nil
}

// GetServerSettings returns per-server settings.
func (s *SettingsService) GetServerSettings(serverID int64) (PerServerSettings, error) {
	var cfg models.PerServerConfig
	err := s.db.Where("server_id = ?", serverID).First(&cfg).Error

	if err == gorm.ErrRecordNotFound {
		// Return defaults.
		return PerServerSettings{
			ServerID:     serverID,
			ShowAvatars:  true,
			ShowBios:     true,
			ShowStatuses: true,
		}, nil
	}
	if err != nil {
		return PerServerSettings{}, fmt.Errorf("get server settings: %w", err)
	}

	return PerServerSettings{
		ServerID:     cfg.ServerID,
		ShowAvatars:  cfg.SyncAvatars,
		ShowBios:     cfg.SyncBios,
		ShowStatuses: cfg.SyncStatus,
	}, nil
}

// UpdateServerSettings updates per-server settings.
func (s *SettingsService) UpdateServerSettings(serverID int64, settings PerServerSettings) error {
	var cfg models.PerServerConfig
	err := s.db.Where("server_id = ?", serverID).First(&cfg).Error

	if err == gorm.ErrRecordNotFound {
		cfg = models.PerServerConfig{
			ServerID:    serverID,
			SyncAvatars: settings.ShowAvatars,
			SyncBios:    settings.ShowBios,
			SyncStatus:  settings.ShowStatuses,
		}
		if err := s.db.Create(&cfg).Error; err != nil {
			return fmt.Errorf("create server settings: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("get server settings: %w", err)
	} else {
		cfg.SyncAvatars = settings.ShowAvatars
		cfg.SyncBios = settings.ShowBios
		cfg.SyncStatus = settings.ShowStatuses
		if err := s.db.Save(&cfg).Error; err != nil {
			return fmt.Errorf("update server settings: %w", err)
		}
	}

	emitEvent(s.ctx, "settings:serverChanged", settings)
	return nil
}
