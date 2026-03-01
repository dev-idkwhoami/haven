package handlers

import (
	"haven/server/auth"
	"haven/server/config"
	"haven/server/middleware"
	"haven/server/sfu"
	"haven/server/ws"

	"gorm.io/gorm"
)

// Deps holds shared dependencies injected into all handlers.
type Deps struct {
	DB          *gorm.DB
	Hub         *ws.Hub
	Hot         *config.HotConfig
	RateLimiter *middleware.RateLimiter
	FileTokens  *FileTokenStore
	SFU         *sfu.SFU
	WaitingRoom *auth.WaitingRoom
}
