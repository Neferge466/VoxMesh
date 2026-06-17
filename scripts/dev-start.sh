#!/bin/bash
# VoxMesh — one-command dev startup (no Docker).
# Usage: bash scripts/dev-start.sh
#
# Starts: PostgreSQL, Redis → auth, channel → ws-gateway → Vite frontend.
# Logs go to ./logs/ — use "tail -f ./logs/*.log" to watch.

set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p logs

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }

# ── 0. Kill any previous VoxMesh processes ──
echo "=== Cleaning up old processes ==="
pkill -f "go run.*services/.*/cmd/main.go" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true
sleep 1

# ── 1. Infrastructure ──
echo ""
echo "=== Infrastructure ==="

# PostgreSQL
if pg_isready -q 2>/dev/null; then
  ok "PostgreSQL running"
else
  echo -n "Starting PostgreSQL... "
  sudo service postgresql start 2>/dev/null && ok "PostgreSQL started" || fail "PostgreSQL failed"
fi

# Redis
if redis-cli ping >/dev/null 2>&1; then
  ok "Redis running"
else
  echo -n "Starting Redis... "
  sudo service redis-server start 2>/dev/null && ok "Redis started" || fail "Redis failed"
fi

# JWT keys
JWT_PRIV="$ROOT/secrets/jwt_private.pem"
JWT_PUB="$ROOT/secrets/jwt_public.pem"
if [ -f "$JWT_PRIV" ] && [ -f "$JWT_PUB" ]; then
  ok "JWT keys present"
else
  warn "JWT keys missing — generating..."
  mkdir -p "$ROOT/secrets"
  openssl genrsa -out "$JWT_PRIV" 2048 2>/dev/null
  openssl rsa -in "$JWT_PRIV" -pubout -out "$JWT_PUB" 2>/dev/null
  ok "JWT keys generated"
fi

# ── 2. Backend services ──
echo ""
echo "=== Backend services ==="

export JWT_PRIVATE_KEY=./secrets/jwt_private.pem
export JWT_PUBLIC_KEY=./secrets/jwt_public.pem
WIN_IP="${WIN_IP:-10.23.183.132}"
CORS_EXTRA="http://${WIN_IP}:5173"

# Auth (8081) — must start first
echo -n "Starting auth :8081... "
CORS_ORIGINS="${CORS_EXTRA}" DATABASE_URL='postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable' REDIS_URL='redis://localhost:6379/0' JWT_PRIVATE_KEY="$JWT_PRIV" JWT_PUBLIC_KEY="$JWT_PUB" HTTP_PORT=8081 go run ./services/auth/cmd/main.go > logs/auth.log 2>&1 &
AUTH_PID=$!
sleep 2
if kill -0 $AUTH_PID 2>/dev/null && curl -s http://localhost:8081/health >/dev/null 2>&1; then
  ok "auth running (pid=$AUTH_PID)"
else
  fail "auth failed — check logs/auth.log"
fi

# Channel (8082)
echo -n "Starting channel :8082... "
CORS_ORIGINS="${CORS_EXTRA}" DATABASE_URL='postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable' REDIS_URL='redis://localhost:6379/1' JWT_PRIVATE_KEY="$JWT_PRIV" JWT_PUBLIC_KEY="$JWT_PUB" HTTP_PORT=8082 go run ./services/channel/cmd/main.go > logs/channel.log 2>&1 &
CH_PID=$!
sleep 2
if kill -0 $CH_PID 2>/dev/null && curl -s http://localhost:8082/health >/dev/null 2>&1; then
  ok "channel running (pid=$CH_PID)"
else
  fail "channel failed — check logs/channel.log"
fi

# WS Gateway (8085) — with CORS for external access
echo -n "Starting ws-gateway :8085... "
CORS_ORIGINS="${CORS_EXTRA}" REDIS_URL='redis://localhost:6379/5' AUTH_SERVICE_URL='http://localhost:8081' CHANNEL_SERVICE_URL='http://localhost:8082' JWT_PRIVATE_KEY="$JWT_PRIV" JWT_PUBLIC_KEY="$JWT_PUB" HTTP_PORT=8085 go run ./services/ws-gateway/cmd/main.go > logs/ws-gateway.log 2>&1 &
WS_PID=$!
sleep 3
if kill -0 $WS_PID 2>/dev/null && curl -s http://localhost:8085/health >/dev/null 2>&1; then
  ok "ws-gateway running (pid=$WS_PID)"
else
  fail "ws-gateway failed — check logs/ws-gateway.log"
fi

# ── 3. Frontend ──
echo ""
echo "=== Frontend ==="
echo -n "Starting Vite :5173... "
cd "$ROOT/web-client"
npm run dev > "$ROOT/logs/frontend.log" 2>&1 &
FE_PID=$!
sleep 3
if kill -0 $FE_PID 2>/dev/null; then
  ok "frontend running (pid=$FE_PID)"
else
  fail "frontend failed — check logs/frontend.log"
fi

cd "$ROOT"

# ── 4. Summary ──
echo ""
echo "════════════════════════════════════════════════"
echo -e "  ${GREEN}All services running!${NC}"
echo ""
echo "  Local access:"
echo "    Frontend:  http://localhost:5173"
echo "    API/WS:    http://localhost:8085"
echo ""
echo "  External access (LAN):"
echo "    Frontend:  http://${WIN_IP}:5173"
echo "    API/WS:    http://${WIN_IP}:8085"
echo ""
echo "  View logs:  tail -f ./logs/*.log"
echo ""
echo "  Stop all:   pkill -f 'go run.*services/' && pkill -f vite"
echo "════════════════════════════════════════════════"

# Save PIDs for stop script
echo "$AUTH_PID $CH_PID $WS_PID $FE_PID" > "$ROOT/.dev-pids"

# Wait so script doesn't exit immediately
wait
