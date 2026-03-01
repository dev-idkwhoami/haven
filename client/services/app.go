package services

import (
	"context"
	"crypto/ed25519"
	"log/slog"
	"sync"

	"haven/client/connection"
	"haven/client/keystore"
	"haven/client/models"

	"gorm.io/gorm"
)

// AppState represents the current application lifecycle phase.
type AppState struct {
	Phase      string `json:"phase"`      // "loading" | "setup" | "ready"
	LoadingMsg string `json:"loadingMsg"` // current loading stage description
	Progress   int    `json:"progress"`   // 0-100
}

// AppService manages application lifecycle and loading state.
type AppService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey

	mu    sync.Mutex
	state AppState
}

// NewAppService creates a new AppService.
func NewAppService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *AppService {
	return &AppService{
		db:      db,
		manager: manager,
		privKey: privKey,
		state: AppState{
			Phase:      "loading",
			LoadingMsg: "Starting...",
			Progress:   0,
		},
	}
}

// SetContext is called by Wails during startup.
func (a *AppService) SetContext(ctx context.Context) {
	a.ctx = ctx
}

// GetState returns the current application state.
func (a *AppService) GetState() AppState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

// Initialize performs the startup sequence and emits state changes.
func (a *AppService) Initialize() {
	a.emitState("loading", "Checking identity...", 10)

	// Check if profile exists to determine setup vs ready.
	var profile models.LocalProfile
	result := a.db.First(&profile)

	if result.Error != nil {
		a.emitState("loading", "Preparing first-time setup...", 50)

		hasKey, err := keystore.Exists()
		if err != nil {
			slog.Error("failed to check keystore", "error", err)
		}

		if !hasKey || result.Error == gorm.ErrRecordNotFound {
			a.emitState("setup", "", 100)
			return
		}
	}

	a.emitState("loading", "Loading servers...", 70)
	a.emitState("loading", "Ready", 100)
	a.emitState("ready", "", 100)
}

// Shutdown gracefully shuts down the application.
func (a *AppService) Shutdown() error {
	slog.Info("application shutdown requested")
	a.manager.DisconnectAll()
	return nil
}

func (a *AppService) emitState(phase, msg string, progress int) {
	a.mu.Lock()
	a.state = AppState{
		Phase:      phase,
		LoadingMsg: msg,
		Progress:   progress,
	}
	state := a.state
	a.mu.Unlock()

	if a.ctx != nil {
		emitEvent(a.ctx, "app:stateChanged", state)
	}
}
