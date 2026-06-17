#!/bin/bash
# VoxMesh database backup script. Run via cron daily:
#   0 3 * * * /path/to/backup-db.sh
set -e

BACKUP_DIR="${BACKUP_DIR:-./backups}"
DB_URL="${DATABASE_URL:-postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

BACKUP_FILE="$BACKUP_DIR/voxmesh_$TIMESTAMP.sql.gz"

echo "[$(date)] Starting backup to $BACKUP_FILE"

if command -v pg_dump >/dev/null 2>&1; then
    pg_dump "$DB_URL" | gzip > "$BACKUP_FILE"
else
    docker exec voxmesh-postgres-1 pg_dump -U voxmesh voxmesh | gzip > "$BACKUP_FILE"
fi

echo "[$(date)] Backup complete: $(du -h "$BACKUP_FILE" | cut -f1)"

# Cleanup old backups
find "$BACKUP_DIR" -name "voxmesh_*.sql.gz" -mtime +"$RETENTION_DAYS" -delete
echo "[$(date)] Cleaned backups older than $RETENTION_DAYS days"
