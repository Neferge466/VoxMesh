#!/bin/bash
# VoxMesh Let's Encrypt certificate setup (production).
# Usage: sudo bash scripts/setup-certbot.sh <domain>
#   e.g. sudo bash scripts/setup-certbot.sh voxmesh.example.com
set -e

DOMAIN="${1:-voxmesh.local}"
EMAIL="${CERTBOT_EMAIL:-admin@${DOMAIN#*.}}"
CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
NGINX_CERT_DIR="./certs"

if [ "$DOMAIN" = "voxmesh.local" ]; then
    echo "WARNING: using default domain 'voxmesh.local'."
    echo "Specify your real domain: $0 my-real-domain.com"
    echo ""
    echo "For production, use certbot:"
    echo "  1. Point DNS A record for $DOMAIN to this server's public IP"
    echo "  2. Ensure ports 80/443 are open and reachable"
    echo "  3. Run: sudo bash $0 $DOMAIN"
    exit 1
fi

echo "=== Setting up Let's Encrypt for $DOMAIN ==="

# Stop nginx temporarily to free port 80 for certbot standalone
docker compose stop nginx 2>/dev/null || true

# Request certificate
sudo certbot certonly --standalone \
    --non-interactive --agree-tos \
    --email "$EMAIL" \
    -d "$DOMAIN"

# Copy certs to nginx directory
mkdir -p "$NGINX_CERT_DIR"
sudo cp "$CERT_DIR/fullchain.pem" "$NGINX_CERT_DIR/server.crt"
sudo cp "$CERT_DIR/privkey.pem"   "$NGINX_CERT_DIR/server.key"
sudo chown "$(whoami)" "$NGINX_CERT_DIR/server.crt" "$NGINX_CERT_DIR/server.key"

echo "=== Certificates copied to $NGINX_CERT_DIR ==="

# Update nginx config with real domain
echo "Update nginx/conf.d/voxmesh.conf server_name to '$DOMAIN'"

# Restart nginx
docker compose up -d nginx

echo "=== Done. Certificates will auto-renew via certbot timer. ==="
echo "Test renewal: sudo certbot renew --dry-run"
