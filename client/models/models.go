package models

// AllModels returns all client-side GORM models for auto-migration.
func AllModels() []any {
	return []any{
		&TrustedServer{},
		&LocalProfile{},
		&CachedUser{},
		&CachedMessage{},
		&PerServerConfig{},
		&CachedCategory{},
		&CachedChannel{},
		&CachedDMMessage{},
	}
}
