#!/bin/sh
# Generate .htpasswd for Nginx monitoring basic auth.
# Usage: MONITOR_USER=admin MONITOR_PASS=secure123 bash setup-monitor-auth.sh

USER="${MONITOR_USER:-admin}"
PASS="${MONITOR_PASS:-voxmesh_monitor}"

HTPASSWD_FILE="$(dirname "$0")/.htpasswd_monitor"

# Use openssl to generate htpasswd-compatible hash (Apache MD5/apr1 format)
if command -v openssl >/dev/null 2>&1; then
    printf "%s:%s\n" "$USER" "$(openssl passwd -apr1 "$PASS")" > "$HTPASSWD_FILE"
    echo "Created $HTPASSWD_FILE (user=$USER)"
elif command -v htpasswd >/dev/null 2>&1; then
    htpasswd -bc "$HTPASSWD_FILE" "$USER" "$PASS"
    echo "Created $HTPASSWD_FILE (user=$USER)"
else
    echo "WARNING: openssl or htpasswd not found. Create $HTPASSWD_FILE manually."
    exit 1
fi
