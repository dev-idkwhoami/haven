package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"github.com/oklog/ulid/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	tea "github.com/charmbracelet/bubbletea"

	"haven/server/auth"
	"haven/server/config"
	servercrypto "haven/server/crypto"
	"haven/server/handlers"
	"haven/server/middleware"
	"haven/server/models"
	"haven/server/sfu"
	"haven/server/tui"
	"haven/server/ws"
	"haven/shared"
)

func main() {
	// CLI flags
	configPath := flag.String("config", "haven-server.toml", "path to config file")
	dataDir := flag.String("data", "data", "path to data directory")
	verbose := flag.Bool("verbose", false, "enable debug logging")
	logFormat := flag.String("log-format", "text", "log format: text or json")
	generateKey := flag.Bool("generate-key", false, "generate a new server key and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	headless := flag.Bool("headless", false, "run without TUI (logs to stderr)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Haven Server %s\n", shared.Version)
		os.Exit(0)
	}

	shared.InitLogger(*verbose, *logFormat)

	if *generateKey {
		keyPath := filepath.Join(*dataDir, "server.key")
		os.MkdirAll(*dataDir, 0700)
		key, err := servercrypto.LoadOrGenerateKey(keyPath)
		if err != nil {
			slog.Error("generate key failed", "error", err)
			os.Exit(1)
		}
		pub := key.Public().(ed25519.PublicKey)
		fmt.Printf("Server public key: %x\n", pub)
		fmt.Printf("Key saved to: %s\n", keyPath)
		os.Exit(0)
	}

	// Auto-generate config if it doesn't exist
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		if err := config.GenerateDefault(*configPath); err != nil {
			slog.Error("generate default config", "error", err)
			os.Exit(1)
		}
		slog.Info("created default config", "path", *configPath)
	}

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	// Init hot config
	hot, err := config.NewHotConfig(cfg, *configPath)
	if err != nil {
		slog.Error("init hot config", "error", err)
		os.Exit(1)
	}
	if err := hot.Watch(); err != nil {
		slog.Warn("config hot-reload disabled", "error", err)
	}
	defer hot.Stop()

	// Ensure data directory exists
	keyDir := filepath.Dir(cfg.Identity.PrivateKeyPath)
	if keyDir != "" && keyDir != "." {
		os.MkdirAll(keyDir, 0700)
	}
	if *dataDir != "" {
		os.MkdirAll(*dataDir, 0700)
	}

	// Init database
	db, err := initDB(cfg)
	if err != nil {
		slog.Error("init database", "error", err)
		os.Exit(1)
	}

	// Auto-migrate all server models
	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		slog.Error("auto-migrate", "error", err)
		os.Exit(1)
	}

	// First-boot seed: create default role and server singleton
	seedFirstBoot(db, cfg)

	// Load/generate server key
	serverKey, err := servercrypto.LoadOrGenerateKey(cfg.Identity.PrivateKeyPath)
	if err != nil {
		slog.Error("load server key", "error", err)
		os.Exit(1)
	}
	serverPub := serverKey.Public().(ed25519.PublicKey)
	slog.Info("server identity loaded", "pubkey", fmt.Sprintf("%x", serverPub))

	// Create hub and router
	hub := ws.NewHub()
	router := ws.NewRouter()
	hub.Router = router

	// Rate limiter
	rateLimiter := middleware.NewRateLimiter(hot)

	// Create file token store
	fileTokens := handlers.NewFileTokenStore()

	// Create SFU
	voiceSFU := sfu.NewSFU()

	// Create waiting room for access requests
	waitingRoom := auth.NewWaitingRoom()

	// Create handler deps and register all handlers
	deps := &handlers.Deps{
		DB:          db,
		Hub:         hub,
		Hot:         hot,
		RateLimiter: rateLimiter,
		FileTokens:  fileTokens,
		SFU:         voiceSFU,
		WaitingRoom: waitingRoom,
	}
	handlers.RegisterAll(deps, router)

	// Wire up channel access check for broadcasting
	hub.ChannelAccess = func(client *ws.Client, channelID string) bool {
		// Check if channel has role restrictions
		var count int64
		db.Model(&models.ChannelRoleAccess{}).Where("channel_id = ?", channelID).Count(&count)
		if count == 0 {
			return true
		}
		if hot.IsOwner(client.PubKey) {
			return true
		}
		var accessCount int64
		db.Model(&models.ChannelRoleAccess{}).
			Joins("JOIN user_roles ON user_roles.role_id = channel_role_accesses.role_id").
			Where("channel_role_accesses.channel_id = ? AND user_roles.user_id = ?", channelID, client.UserID).
			Count(&accessCount)
		return accessCount > 0
	}

	// Wire up hot-reload callback to notify clients of owner status changes
	// and enforce allowlist changes.
	hot.OnReload = func(oldOwners, newOwners, oldAllowlist, newAllowlist [][]byte) {
		contains := func(list [][]byte, key []byte) bool {
			for _, k := range list {
				if bytes.Equal(k, key) {
					return true
				}
			}
			return false
		}
		// Clients whose pubkey was removed from owners
		for _, old := range oldOwners {
			if !contains(newOwners, old) {
				msg, _ := ws.MarshalEvent(shared.TypeEventOwnerChanged, map[string]any{"is_owner": false})
				hub.SendTo(fmt.Sprintf("%x", old), msg)
			}
		}
		// Clients whose pubkey was added as owner
		for _, nk := range newOwners {
			if !contains(oldOwners, nk) {
				msg, _ := ws.MarshalEvent(shared.TypeEventOwnerChanged, map[string]any{"is_owner": true})
				hub.SendTo(fmt.Sprintf("%x", nk), msg)
			}
		}

		// Allowlist enforcement: if server is in allowlist mode, kick clients
		// that are no longer on the allowlist, not owners, and have no approved access request.
		slog.Debug("hot-reload: checking allowlist enforcement",
			"old_owners", len(oldOwners), "new_owners", len(newOwners),
			"old_allowlist", len(oldAllowlist), "new_allowlist", len(newAllowlist))

		// Resolve effective access mode: config override takes precedence.
		accessMode := hot.AccessMode()
		if accessMode == "" {
			var srv models.Server
			if err := db.First(&srv).Error; err != nil {
				slog.Error("hot-reload: failed to load server for enforcement", "error", err)
				return
			}
			accessMode = srv.AccessMode
		}
		slog.Debug("hot-reload: effective access mode", "access_mode", accessMode)
		if accessMode == shared.AccessModeAllowlist {
			clientCount := hub.ClientCount()
			slog.Debug("hot-reload: enforcing allowlist", "connected_clients", clientCount)
			hub.ForEachClient(func(c *ws.Client) {
				isOwner := contains(newOwners, c.PubKey)
				isAllowlisted := contains(newAllowlist, c.PubKey)
				if isOwner || isAllowlisted {
					slog.Debug("hot-reload: client exempt", "pubkey", c.PubKeyHex, "is_owner", isOwner, "is_allowlisted", isAllowlisted)
					return
				}
				// Check for approved access request.
				var count int64
				db.Model(&models.AccessRequest{}).Where("public_key = ? AND status = ?", c.PubKey, "approved").Count(&count)
				if count > 0 {
					slog.Debug("hot-reload: client has approved access request", "pubkey", c.PubKeyHex)
					return
				}
				slog.Info("kicking client removed from allowlist", "pubkey", c.PubKeyHex)
				kickMsg, _ := ws.MarshalEvent(shared.TypeAuthError, map[string]any{
					"code":    shared.ErrNotAllowlisted,
					"message": "you have been removed from the allowlist",
				})
				c.SendCh <- kickMsg
				hub.Unregister <- c
			})
		}
	}

	go hub.Run()

	// Start periodic session cleanup
	go sessionCleanupLoop(db)

	// WebSocket upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	// HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !rateLimiter.AllowAuth(ip) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Debug("ws upgrade failed", "error", err)
			return
		}

		isTLS := r.TLS != nil
		go auth.HandleNewConnection(conn, hub, db, cfg, hot, serverKey, isTLS, waitingRoom)
	})

	// File upload endpoint
	mux.HandleFunc("/upload", handlers.HandleUpload(db, hub, fileTokens, *dataDir))

	// File download endpoint
	mux.HandleFunc("/files/", handlers.HandleDownload(db, fileTokens))

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Network.ListenAddress, cfg.Network.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	slog.Info("starting server", "address", addr)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Read server name from DB for TUI title
	var srv models.Server
	db.First(&srv)

	if *headless {
		<-ctx.Done()
	} else {
		// Redirect logs to ring buffer for TUI display
		logBuf := tui.NewLogBuffer(500)
		slog.SetDefault(slog.New(slog.NewTextHandler(logBuf, nil)))

		tuiApp := tui.NewApp(&tui.Deps{
			DB:          db,
			Hub:         hub,
			Hot:         hot,
			WaitingRoom: waitingRoom,
			LogBuffer:   logBuf,
			ServerName:  srv.Name,
			ListenAddr:  addr,
			StartTime:   time.Now(),
		})
		p := tea.NewProgram(tuiApp, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			slog.Error("TUI error", "error", err)
		}
		stop() // Cancel signal context to trigger shutdown
	}

	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("server stopped")
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch strings.ToLower(cfg.Database.Driver) {
	case "sqlite":
		// Ensure directory exists for SQLite file
		dir := filepath.Dir(cfg.Database.DSN)
		if dir != "" && dir != "." {
			os.MkdirAll(dir, 0700)
		}
		return gorm.Open(sqlite.Open(cfg.Database.DSN), gormCfg)
	case "postgres":
		return gorm.Open(postgres.Open(cfg.Database.DSN), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}
}

func seedFirstBoot(db *gorm.DB, cfg *config.Config) {
	// Create default role if none exists
	var roleCount int64
	db.Model(&models.Role{}).Count(&roleCount)
	if roleCount == 0 {
		defaultRole := models.Role{
			ID:          ulid.Make().String(),
			Name:        "Member",
			Position:    0,
			IsDefault:   true,
			Permissions: shared.PermDefaultMember,
			Version:     1,
		}
		if err := db.Create(&defaultRole).Error; err != nil {
			slog.Error("create default role", "error", err)
		} else {
			slog.Info("created default role", "id", defaultRole.ID)
		}
	}

	// Create server singleton if none exists
	var serverCount int64
	db.Model(&models.Server{}).Count(&serverCount)
	if serverCount == 0 {
		accessMode := cfg.Auth.AccessMode
		if accessMode == "" {
			accessMode = "open"
		}
		srv := models.Server{
			ID:                ulid.Make().String(),
			Name:              cfg.Defaults.ServerName,
			AccessMode:        accessMode,
			MaxFileSize:       cfg.Defaults.MaxFileSize,
			TotalStorageLimit: cfg.Defaults.TotalStorageLimit,
			Version:           1,
		}
		if err := db.Create(&srv).Error; err != nil {
			slog.Error("create server singleton", "error", err)
		} else {
			slog.Info("created server", "name", srv.Name, "id", srv.ID)
		}
	}
}

func sessionCleanupLoop(db *gorm.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if err := auth.CleanExpiredSessions(db); err != nil {
			slog.Error("session cleanup", "error", err)
		}
	}
}

func extractIP(r *http.Request) string {
	// Check X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
