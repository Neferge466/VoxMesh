#!/bin/bash
# VoxMesh production deployment script.
# Usage: bash scripts/deploy-production.sh [domain]
#   e.g. bash scripts/deploy-production.sh voxmesh.example.com
set -euo pipefail

DOMAIN="${1:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "=== VoxMesh Production Deployment ==="
echo ""

# --- Prerequisites ---
if ! command -v docker &>/dev/null; then
    echo "ERROR: Docker not found. Install: https://docs.docker.com/engine/install/"
    exit 1
fi
if ! docker compose version &>/dev/null; then
    echo "ERROR: Docker Compose v2 required."
    exit 1
fi
echo "[OK] Docker $(docker --version) + Compose v2"

# --- Domain ---
if [ -z "$DOMAIN" ]; then
    DOMAIN="${DEPLOY_DOMAIN:-}"
fi
if [ -z "$DOMAIN" ]; then
    echo ""
    echo "No domain specified. Usage:"
    echo "  bash scripts/deploy-production.sh your-domain.com"
    echo ""
    echo "For local testing without a domain:"
    echo "  DEPLOY_SKIP_TLS=1 bash scripts/deploy-production.sh"
    exit 1
fi
echo "[OK] Domain: $DOMAIN"

# --- Secrets ---
mkdir -p secrets certs

if [ ! -f secrets/jwt_private.pem ]; then
    echo "[GEN] Generating JWT RS256 key pair..."
    openssl genrsa -out secrets/jwt_private.pem 2048
    openssl rsa -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem
    echo "[OK] JWT keys generated"
else
    echo "[OK] JWT keys exist"
fi

if [ ! -f .env ]; then
    echo "[GEN] Creating .env from .env.example..."
    cp .env.example .env
    # Generate random passwords
    DB_PW=$(openssl rand -hex 16)
    EMQX_PW=$(openssl rand -hex 12)
    GRAFANA_PW=$(openssl rand -hex 12)
    MINIO_PW=$(openssl rand -hex 12)
    TURN_SECRET=$(openssl rand -hex 16)
    LIVEKIT_KEYS="livekit_$(openssl rand -hex 4): $(openssl rand -hex 24)"
    sed -i "s/DB_PASSWORD=.*/DB_PASSWORD=$DB_PW/" .env
    sed -i "s/EMQX_ADMIN_PASSWORD=.*/EMQX_ADMIN_PASSWORD=$EMQX_PW/" .env
    sed -i "s/GRAFANA_PASSWORD=.*/GRAFANA_PASSWORD=$GRAFANA_PW/" .env
    sed -i "s/MINIO_PASSWORD=.*/MINIO_PASSWORD=$MINIO_PW/" .env
    sed -i "s/TURN_SECRET=.*/TURN_SECRET=$TURN_SECRET/" .env
    sed -i "s|LIVEKIT_KEYS=.*|LIVEKIT_KEYS=$LIVEKIT_KEYS|" .env
    echo "[OK] .env created with random passwords"
else
    echo "[OK] .env exists"
fi

# --- TLS Certificates ---
if [ "${DEPLOY_SKIP_TLS:-0}" = "1" ]; then
    echo "[SKIP] TLS (DEPLOY_SKIP_TLS=1)"
elif [ ! -f certs/server.crt ]; then
    echo "[GEN] Generating self-signed TLS certificate for $DOMAIN..."
    openssl req -x509 -newkey rsa:2048 \
        -keyout certs/server.key \
        -out certs/server.crt \
        -days 365 -nodes \
        -subj "/CN=$DOMAIN" \
        -addext "subjectAltName=DNS:$DOMAIN"
    echo "[OK] Self-signed certificate created"
    echo "[NOTE] Replace with Let's Encrypt: sudo bash scripts/setup-certbot.sh $DOMAIN"
else
    echo "[OK] TLS certificates exist"
fi

# --- Nginx domain ---
echo "[CONFIG] Setting nginx server_name to $DOMAIN..."
if grep -q "server_name voxmesh.local" nginx/conf.d/voxmesh.conf 2>/dev/null; then
    sed -i "s/server_name voxmesh.local/server_name $DOMAIN/g" nginx/conf.d/voxmesh.conf
fi
echo "[OK] Nginx configured"

# --- Build ---
echo ""
echo "=== Building all service images ==="
docker compose build --parallel

# --- Start ---
echo ""
echo "=== Starting all services ==="
docker compose up -d

# --- Wait for health ---
echo ""
echo "=== Waiting for services to be healthy (timeout: 120s) ==="
TIMEOUT=120
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    UNHEALTHY=$(docker compose ps --format json 2>/dev/null | grep -v '"Health":"healthy"' | grep -v '"Health":""' || true)
    if [ -z "$UNHEALTHY" ]; then
        echo "[OK] All services healthy after ${ELAPSED}s"
        break
    fi
    sleep 5
    ELAPSED=$((ELAPSED + 5))
    if [ $((ELAPSED % 30)) -eq 0 ]; then
        echo "[WAIT] Still waiting... ($ELAPSED s)"
    fi
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo "[WARN] Timeout. Some services may not be healthy:"
    docker compose ps
fi

# --- Verify ---
echo ""
echo "=== Verification ==="
echo "Health check:"
curl -sfk https://$DOMAIN/health 2>/dev/null && echo "  [OK] HTTPS /health" || echo "  [WARN] HTTPS /health failed"
curl -sf http://localhost:8080/health 2>/dev/null && echo "  [OK] HTTP :8080 /health" || echo "  [WARN] HTTP :8080 /health failed"

echo ""
echo "=== Deployment complete ==="
echo ""
echo "  Frontend:   https://$DOMAIN"
echo "  API:        https://$DOMAIN/api/v1"
echo "  Metrics:    https://$DOMAIN/metrics (ws-gateway)"
echo "  EMQX Dash:  https://$DOMAIN:8443/emqx/ (basic auth)"
echo "  Grafana:    https://$DOMAIN:8443/grafana/ (basic auth)"
echo "  MinIO:      https://$DOMAIN:8443/minio/ (basic auth)"
echo ""
echo "To set up Let's Encrypt (production TLS):"
echo "  sudo bash scripts/setup-certbot.sh $DOMAIN"
echo ""
echo "To view logs:"
echo "  docker compose logs -f"
