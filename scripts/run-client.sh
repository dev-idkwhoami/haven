#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT/client/frontend"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ok()   { echo -e "  ${GREEN}[OK]${NC}  $1"; }
warn() { echo -e "  ${YELLOW}[!!]${NC}  $1"; }
fail() { echo -e "  ${RED}[FAIL]${NC}  $1"; }

echo ""
echo "=== Haven Client — Dependency Check ==="
echo ""

ERRORS=0
WARNINGS=0

# Go
if command -v go &>/dev/null; then
    GO_VER=$(go version | awk '{print $3}')
    ok "Go installed ($GO_VER)"
else
    fail "Go not found. Install from https://go.dev/dl/"
    ERRORS=$((ERRORS + 1))
fi

# CGo compiler (gcc/clang)
if command -v gcc &>/dev/null; then
    GCC_VER=$(gcc --version 2>&1 | head -1)
    ok "GCC found ($GCC_VER)"
elif command -v clang &>/dev/null; then
    CLANG_VER=$(clang --version 2>&1 | head -1)
    ok "Clang found ($CLANG_VER)"
else
    fail "C compiler not found. Install GCC or Clang (required for CGo)"
    echo "       Windows: install MSYS2 + mingw-w64-x86_64-gcc"
    echo "       macOS:   xcode-select --install"
    echo "       Linux:   sudo apt install build-essential"
    ERRORS=$((ERRORS + 1))
fi

# pkg-config
if command -v pkg-config &>/dev/null; then
    ok "pkg-config found"
else
    warn "pkg-config not found (needed to locate C libraries)"
    echo "       Windows: pacman -S mingw-w64-x86_64-pkg-config (MSYS2)"
    echo "       macOS:   brew install pkg-config"
    echo "       Linux:   sudo apt install pkg-config"
    WARNINGS=$((WARNINGS + 1))
fi

# PortAudio
check_lib() {
    local name="$1"
    local pkg="$2"
    local install_hint="$3"

    if command -v pkg-config &>/dev/null && pkg-config --exists "$pkg" 2>/dev/null; then
        LIB_VER=$(pkg-config --modversion "$pkg" 2>/dev/null || echo "found")
        ok "$name ($LIB_VER)"
    else
        fail "$name not found"
        echo "       $install_hint"
        ERRORS=$((ERRORS + 1))
    fi
}

check_lib "PortAudio" "portaudio-2.0" \
    "Windows: pacman -S mingw-w64-x86_64-portaudio | macOS: brew install portaudio | Linux: sudo apt install libportaudio2 portaudio19-dev"

check_lib "Opus" "opus" \
    "Windows: pacman -S mingw-w64-x86_64-opus | macOS: brew install opus | Linux: sudo apt install libopus-dev"

# Node.js / npm (for frontend)
if command -v node &>/dev/null; then
    NODE_VER=$(node --version)
    ok "Node.js installed ($NODE_VER)"
else
    fail "Node.js not found. Install from https://nodejs.org/"
    ERRORS=$((ERRORS + 1))
fi

if command -v npm &>/dev/null; then
    NPM_VER=$(npm --version)
    ok "npm installed (v$NPM_VER)"
else
    fail "npm not found"
    ERRORS=$((ERRORS + 1))
fi

# Wails CLI (optional but recommended)
if command -v wails &>/dev/null; then
    WAILS_VER=$(wails version 2>&1 | head -1 || echo "found")
    ok "Wails CLI ($WAILS_VER)"
else
    warn "Wails CLI not found (needed for full desktop app build)"
    echo "       Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""
echo "---"
if [ "$ERRORS" -gt 0 ]; then
    fail "$ERRORS dependency issue(s) found. Fix them and try again."
    exit 1
fi
if [ "$WARNINGS" -gt 0 ]; then
    warn "$WARNINGS non-critical warning(s). The build may still work."
fi

echo ""
echo "=== Installing Frontend Dependencies ==="
echo ""

if [ -f "$FRONTEND_DIR/package.json" ]; then
    cd "$FRONTEND_DIR"
    if [ ! -d "node_modules" ]; then
        npm install
        ok "Frontend dependencies installed"
    else
        ok "Frontend dependencies already installed"
    fi
    cd "$ROOT"
else
    warn "No package.json found at $FRONTEND_DIR"
fi

echo ""
echo "=== Building Client (CGo enabled) ==="
echo ""

mkdir -p "$ROOT/build"
cd "$ROOT"
CGO_ENABLED=1 go build -o "$ROOT/build/haven-client" ./client

ok "Built successfully: $ROOT/build/haven-client"

echo ""
echo "=== Starting Client ==="
echo ""
echo "Note: Full desktop UI requires Wails. Running in headless mode."
echo "      To run as desktop app: cd $ROOT && wails dev"
echo ""

exec "$ROOT/build/haven-client"
