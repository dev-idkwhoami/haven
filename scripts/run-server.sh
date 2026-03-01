#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG="$ROOT/haven-server.toml"
DATA_DIR="$ROOT/server/data"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ok()   { echo -e "  ${GREEN}[OK]${NC}  $1"; }
warn() { echo -e "  ${YELLOW}[!!]${NC}  $1"; }
fail() { echo -e "  ${RED}[FAIL]${NC}  $1"; }

echo ""
echo "=== Haven Server — Dependency Check ==="
echo ""

ERRORS=0

# Go
if command -v go &>/dev/null; then
    GO_VER=$(go version | awk '{print $3}')
    ok "Go installed ($GO_VER)"
else
    fail "Go not found. Install from https://go.dev/dl/"
    ERRORS=$((ERRORS + 1))
fi

# Check go.mod exists
if [ -f "$ROOT/go.mod" ]; then
    ok "go.mod found"
else
    fail "go.mod not found in $ROOT"
    ERRORS=$((ERRORS + 1))
fi

# Config file
if [ -f "$CONFIG" ]; then
    ok "Config found ($CONFIG)"
else
    warn "Config not found — creating default haven-server.toml"
    cat > "$CONFIG" <<'TOML'
[identity]
private_key_path = "data/server.key"

[network]
listen_address = "0.0.0.0"
port = 9090

[database]
driver = "sqlite"
dsn = "data/haven.db"

[owners]
public_keys = []

[defaults]
server_name = "My Haven Server"
access_mode = "open"
max_file_size = 52428800          # 50 MB
total_storage_limit = 21474836480  # 20 GB

[rate_limits]
messages_per_second = 20
message_burst = 50
auth_attempts_per_minute = 10
registrations_per_ip_per_hour = 20
concurrent_uploads = 5

[session]
grace_period_seconds = 300

[allowlist]
public_keys = []
TOML
    ok "Default config created at $CONFIG"
fi

# Data directory
mkdir -p "$DATA_DIR"
ok "Data directory ready ($DATA_DIR)"

if [ "$ERRORS" -gt 0 ]; then
    echo ""
    fail "$ERRORS dependency issue(s) found. Fix them and try again."
    exit 1
fi

echo ""
echo "=== Building Server ==="
echo ""

mkdir -p "$ROOT/build"
cd "$ROOT"
go build -o "$ROOT/build/haven-server" ./server

ok "Built successfully: $ROOT/build/haven-server"

echo ""
echo "=== Starting Server ==="
echo ""

exec "$ROOT/build/haven-server" -config "$CONFIG" -data "$DATA_DIR" -verbose
