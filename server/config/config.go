package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the full haven-server.toml structure.
type Config struct {
	Identity   IdentityConfig   `toml:"identity"`
	Network    NetworkConfig    `toml:"network"`
	Database   DatabaseConfig   `toml:"database"`
	Owners     OwnersConfig     `toml:"owners"`
	Defaults   DefaultsConfig   `toml:"defaults"`
	RateLimits RateLimitsConfig `toml:"rate_limits"`
	Session    SessionConfig    `toml:"session"`
	Auth       AuthConfig       `toml:"auth"`
	Allowlist  AllowlistConfig  `toml:"allowlist"`
}

// IdentityConfig holds the server's key path. Not hot-reloadable.
type IdentityConfig struct {
	PrivateKeyPath string `toml:"private_key_path"`
}

// NetworkConfig holds listen address and port. Not hot-reloadable.
type NetworkConfig struct {
	ListenAddress string `toml:"listen_address"`
	Port          int    `toml:"port"`
}

// DatabaseConfig holds driver and DSN. Not hot-reloadable.
type DatabaseConfig struct {
	Driver string `toml:"driver"`
	DSN    string `toml:"dsn"`
}

// OwnersConfig holds owner public keys (hex-encoded Ed25519). Hot-reloadable.
type OwnersConfig struct {
	PublicKeys []string `toml:"public_keys"`
}

// DefaultsConfig holds initial values seeded into the DB on first boot only.
type DefaultsConfig struct {
	ServerName        string `toml:"server_name"`
	MaxFileSize       int64  `toml:"max_file_size"`
	TotalStorageLimit int64  `toml:"total_storage_limit"`
}

// RateLimitsConfig holds rate limiting settings. Hot-reloadable.
type RateLimitsConfig struct {
	MessagesPerSecond         int `toml:"messages_per_second"`
	MessageBurst              int `toml:"message_burst"`
	AuthAttemptsPerMinute     int `toml:"auth_attempts_per_minute"`
	RegistrationsPerIPPerHour int `toml:"registrations_per_ip_per_hour"`
	ConcurrentUploads         int `toml:"concurrent_uploads"`
}

// SessionConfig holds session settings. Not hot-reloadable.
type SessionConfig struct {
	GracePeriodSeconds int `toml:"grace_period_seconds"`
}

// AuthConfig holds access control settings. Hot-reloadable.
type AuthConfig struct {
	AccessMode string `toml:"access_mode"`
}

// AllowlistConfig holds allowlisted public keys and access request settings. Hot-reloadable.
type AllowlistConfig struct {
	PublicKeys          []string `toml:"public_keys"`
	AllowAccessRequests bool     `toml:"allow_access_requests"`
	RequestTimeoutSecs  int      `toml:"request_timeout_seconds"`
}

// defaults returns a Config with the documented default values.
func defaults() Config {
	return Config{
		Identity: IdentityConfig{
			PrivateKeyPath: "data/server.key",
		},
		Network: NetworkConfig{
			ListenAddress: "0.0.0.0",
			Port:          9090,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "data/haven.db",
		},
		Defaults: DefaultsConfig{
			ServerName:        "My Haven Server",
			MaxFileSize:       52428800,
			TotalStorageLimit: 21474836480,
		},
		RateLimits: RateLimitsConfig{
			MessagesPerSecond:         20,
			MessageBurst:              50,
			AuthAttemptsPerMinute:     10,
			RegistrationsPerIPPerHour: 20,
			ConcurrentUploads:         5,
		},
		Session: SessionConfig{
			GracePeriodSeconds: 300,
		},
		Auth: AuthConfig{
			AccessMode: "open",
		},
		Allowlist: AllowlistConfig{
			AllowAccessRequests: false,
			RequestTimeoutSecs:  300,
		},
	}
}

// GenerateDefault writes a default config file with comments to the given path.
func GenerateDefault(path string) error {
	const tmpl = `# ─────────────────────────────────────────────────────────────
# Haven Server Configuration
# ─────────────────────────────────────────────────────────────
# Fields marked [hot-reloadable] take effect immediately when
# this file is saved — no server restart required.

[identity]
# Path to the Ed25519 private key file. Auto-generated on first run.
private_key_path = "data/server.key"

[network]
# Address and port the server listens on. All traffic (WebSocket +
# HTTP file transfers) uses this single TCP port.
listen_address = "0.0.0.0"
port = 9090

[database]
# Database backend. Values: "sqlite", "postgres"
driver = "sqlite"
# For sqlite: file path. For postgres: a connection string
# e.g. "host=localhost user=haven dbname=haven sslmode=disable"
dsn = "data/haven.db"

[owners]
# [hot-reloadable]
# Hex-encoded Ed25519 public keys of server owners.
# Owners bypass all permission checks.
public_keys = []

[defaults]
# Values used only on first boot to seed the database.
# Changing these after first run has no effect — use the
# client UI or API to update server settings instead.
server_name = "My Haven Server"
# Maximum upload size per file in bytes (default: 50 MB)
max_file_size = 52428800
# Total storage limit for all uploads in bytes (default: 20 GB)
total_storage_limit = 21474836480

[rate_limits]
# [hot-reloadable]
# Message rate limiting (token bucket)
messages_per_second = 20
message_burst = 50
# Auth and registration throttling
auth_attempts_per_minute = 10
registrations_per_ip_per_hour = 20
# Max concurrent file uploads per user
concurrent_uploads = 5

[session]
# Seconds a session remains valid after disconnect before cleanup
grace_period_seconds = 300

[auth]
# [hot-reloadable]
# Access mode. Values: "open", "invite", "password", "allowlist"
# Controls who can join the server. Leave empty to use the value
# from the admin panel instead.
access_mode = "open"

[allowlist]
# [hot-reloadable]
# Hex-encoded Ed25519 public keys that are always allowed to join
# when access_mode is "allowlist". Users approved via access requests
# are stored in the database and do not need to be listed here.
public_keys = []
# Allow non-allowlisted users to request access via a waiting room.
allow_access_requests = false
# Seconds before an access request times out (default: 300 = 5 min).
request_timeout_seconds = 300
`
	return os.WriteFile(path, []byte(tmpl), 0644)
}

// Load reads and parses a TOML config file, applying defaults for missing fields.
func Load(path string) (*Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	return &cfg, nil
}
