package config

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// hotState holds the parsed hot-reloadable values.
type hotState struct {
	OwnerPubKeys        [][]byte
	RateLimits          RateLimitsConfig
	AllowlistPubKeys    [][]byte
	AccessMode          string
	AllowAccessRequests bool
	RequestTimeoutSecs  int
}

// HotConfig holds the hot-reloadable portion of the config behind a RWMutex.
type HotConfig struct {
	mu    sync.RWMutex
	state hotState

	path    string
	watcher *fsnotify.Watcher
	stopCh  chan struct{}
	once    sync.Once

	// OnReload is called after a hot-reload with old/new owner and allowlist public key lists.
	// Set this before calling Watch().
	OnReload func(oldOwners, newOwners, oldAllowlist, newAllowlist [][]byte)
}

// NewHotConfig creates a HotConfig seeded from the provided Config.
func NewHotConfig(cfg *Config, path string) (*HotConfig, error) {
	ownerKeys, err := parseHexKeys(cfg.Owners.PublicKeys)
	if err != nil {
		return nil, fmt.Errorf("parse owner public keys: %w", err)
	}
	allowKeys, err := parseHexKeys(cfg.Allowlist.PublicKeys)
	if err != nil {
		return nil, fmt.Errorf("parse allowlist public keys: %w", err)
	}

	h := &HotConfig{
		state: hotState{
			OwnerPubKeys:        ownerKeys,
			RateLimits:          cfg.RateLimits,
			AllowlistPubKeys:    allowKeys,
			AccessMode:          cfg.Auth.AccessMode,
			AllowAccessRequests: cfg.Allowlist.AllowAccessRequests,
			RequestTimeoutSecs:  cfg.Allowlist.RequestTimeoutSecs,
		},
		path:   path,
		stopCh: make(chan struct{}),
	}
	return h, nil
}

// IsOwner checks if the given public key belongs to a server owner.
func (h *HotConfig) IsOwner(pubkey []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, k := range h.state.OwnerPubKeys {
		if bytes.Equal(k, pubkey) {
			return true
		}
	}
	return false
}

// IsAllowlisted checks if the given public key is on the server allowlist.
func (h *HotConfig) IsAllowlisted(pubkey []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, k := range h.state.AllowlistPubKeys {
		if bytes.Equal(k, pubkey) {
			return true
		}
	}
	return false
}

// RateLimits returns a copy of the current rate limit settings.
func (h *HotConfig) RateLimits() RateLimitsConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.state.RateLimits
}

// AccessMode returns the hot-reloadable access mode override.
// Returns empty string if not set (meaning the DB value should be used).
func (h *HotConfig) AccessMode() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.state.AccessMode
}

// AllowAccessRequests returns whether the server allows access requests in allowlist mode.
func (h *HotConfig) AllowAccessRequests() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.state.AllowAccessRequests
}

// RequestTimeout returns the configured access request timeout duration.
func (h *HotConfig) RequestTimeout() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	secs := h.state.RequestTimeoutSecs
	if secs <= 0 {
		secs = 300
	}
	return time.Duration(secs) * time.Second
}

// Watch starts a file watcher that reloads hot-reloadable sections on config file changes.
// Watches the parent directory instead of the file directly, because many editors
// on Windows do atomic saves (write temp + rename) which breaks inode-based watchers.
func (h *HotConfig) Watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher: %w", err)
	}
	h.watcher = watcher

	dir := filepath.Dir(h.path)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return fmt.Errorf("watch config directory: %w", err)
	}

	go h.watchLoop()
	return nil
}

func (h *HotConfig) watchLoop() {
	absPath, _ := filepath.Abs(h.path)
	for {
		select {
		case event, ok := <-h.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			// Only react to events for our config file.
			evtAbs, _ := filepath.Abs(event.Name)
			if evtAbs != absPath {
				continue
			}
			slog.Info("config file changed, reloading", "op", event.Op.String())
			h.reload()
		case err, ok := <-h.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("config watcher error", "error", err)
		case <-h.stopCh:
			return
		}
	}
}

func (h *HotConfig) reload() {
	cfg, err := Load(h.path)
	if err != nil {
		slog.Error("failed to reload config", "error", err)
		return
	}

	ownerKeys, err := parseHexKeys(cfg.Owners.PublicKeys)
	if err != nil {
		slog.Error("failed to parse owner keys on reload", "error", err)
		return
	}
	allowKeys, err := parseHexKeys(cfg.Allowlist.PublicKeys)
	if err != nil {
		slog.Error("failed to parse allowlist keys on reload", "error", err)
		return
	}

	h.mu.Lock()
	oldOwners := h.state.OwnerPubKeys
	oldAllowlist := h.state.AllowlistPubKeys
	h.state.OwnerPubKeys = ownerKeys
	h.state.RateLimits = cfg.RateLimits
	h.state.AllowlistPubKeys = allowKeys
	h.state.AccessMode = cfg.Auth.AccessMode
	h.state.AllowAccessRequests = cfg.Allowlist.AllowAccessRequests
	h.state.RequestTimeoutSecs = cfg.Allowlist.RequestTimeoutSecs
	h.mu.Unlock()

	slog.Info("hot-reloaded config", "path", h.path,
		"owners", len(ownerKeys), "allowlist", len(allowKeys))

	slog.Debug("hot-reload: calling OnReload callback",
		"old_owners", len(oldOwners), "new_owners", len(ownerKeys),
		"old_allowlist", len(oldAllowlist), "new_allowlist", len(allowKeys),
		"allow_access_requests", cfg.Allowlist.AllowAccessRequests,
		"request_timeout_secs", cfg.Allowlist.RequestTimeoutSecs)
	if h.OnReload != nil {
		h.OnReload(oldOwners, ownerKeys, oldAllowlist, allowKeys)
	}
}

// Stop shuts down the file watcher.
func (h *HotConfig) Stop() {
	h.once.Do(func() {
		close(h.stopCh)
		if h.watcher != nil {
			h.watcher.Close()
		}
	})
}

// parseHexKeys decodes a slice of hex-encoded Ed25519 public keys into raw byte slices.
func parseHexKeys(hexKeys []string) ([][]byte, error) {
	keys := make([][]byte, 0, len(hexKeys))
	for _, h := range hexKeys {
		b, err := hex.DecodeString(h)
		if err != nil {
			return nil, fmt.Errorf("decode hex key %q: %w", h, err)
		}
		if len(b) != 32 {
			return nil, fmt.Errorf("key %q: expected 32 bytes, got %d", h, len(b))
		}
		keys = append(keys, b)
	}
	return keys, nil
}
