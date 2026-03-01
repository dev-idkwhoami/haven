package main

import (
	"context"
	"crypto/ed25519"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"haven/client/connection"
	havenCrypto "haven/client/crypto"
	"haven/client/keystore"
	"haven/client/services"
	"haven/shared"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	verbose := flag.Bool("verbose", false, "enable debug logging")
	flag.Parse()
	shared.InitLogger(*verbose, "text")

	// Load or create identity key pair.
	privKey, err := loadOrCreateIdentity()
	if err != nil {
		slog.Error("identity setup failed", "error", err)
		os.Exit(1)
	}
	pubKey := privKey.Public().(ed25519.PublicKey)
	slog.Info("identity loaded", "pubkey", fmt.Sprintf("%x", pubKey))

	// Open encrypted client database.
	dataDir := getDataDir()
	db, err := connection.OpenDatabase(privKey, dataDir)
	if err != nil {
		slog.Error("database setup failed", "error", err)
		os.Exit(1)
	}

	// Create connection manager.
	manager := connection.NewManager()

	// Create all services.
	appService := services.NewAppService(db, manager, privKey)
	serverService := services.NewServerService(db, manager, privKey)
	channelService := services.NewChannelService(db, manager, privKey)
	messageService := services.NewMessageService(db, manager, privKey)
	userService := services.NewUserService(db, manager, privKey)
	profileService := services.NewProfileService(db, manager, privKey)
	roleService := services.NewRoleService(db, manager, privKey)
	dmService := services.NewDMService(db, manager, privKey)
	settingsService := services.NewSettingsService(db, manager, privKey)
	adminService := services.NewAdminService(db, manager, privKey)
	fileService := services.NewFileService(db, manager, privKey)
	voiceService := services.NewVoiceService(db, manager, privKey)

	// Run Wails application.
	err = wails.Run(&options.App{
		Title:  "Haven",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			// Wire Wails runtime context and event emitter into services.
			services.SetEmitFunc(wailsRuntime.EventsEmit)
			manager.SetWailsContext(ctx, wailsRuntime.EventsEmit)

			appService.SetContext(ctx)
			serverService.SetContext(ctx)
			channelService.SetContext(ctx)
			messageService.SetContext(ctx)
			userService.SetContext(ctx)
			profileService.SetContext(ctx)
			roleService.SetContext(ctx)
			dmService.SetContext(ctx)
			settingsService.SetContext(ctx)
			adminService.SetContext(ctx)
			fileService.SetContext(ctx)
			voiceService.SetContext(ctx)

			appService.Initialize()
		},
		OnShutdown: func(ctx context.Context) {
			_ = appService.Shutdown()
		},
		Bind: []interface{}{
			appService,
			serverService,
			channelService,
			messageService,
			userService,
			profileService,
			roleService,
			dmService,
			settingsService,
			adminService,
			fileService,
			voiceService,
		},
	})
	if err != nil {
		slog.Error("wails app error", "error", err)
		os.Exit(1)
	}
}

// loadOrCreateIdentity loads the Ed25519 private key from the OS keystore,
// or generates and stores a new one if none exists.
func loadOrCreateIdentity() (ed25519.PrivateKey, error) {
	privKey, err := keystore.Load()
	if err != nil {
		return nil, fmt.Errorf("load keystore: %w", err)
	}
	if privKey != nil {
		return privKey, nil
	}

	// Generate new key pair.
	_, priv, err := havenCrypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}

	if err := keystore.Store(priv); err != nil {
		return nil, fmt.Errorf("store key: %w", err)
	}

	slog.Info("generated new identity key pair")
	return priv, nil
}

// getDataDir returns the platform-appropriate data directory for Haven client.
func getDataDir() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		if appData != "" {
			return filepath.Join(appData, "Haven")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Local", "Haven")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Haven")
	default: // linux and others
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome != "" {
			return filepath.Join(dataHome, "haven")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "haven")
	}
}
