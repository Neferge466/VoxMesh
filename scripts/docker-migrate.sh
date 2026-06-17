#!/bin/sh
# Docker entrypoint — runs database migrations then exits.
# Mount the migrations directory to /migrations in the container.

set -e

DB_URL="${DATABASE_URL:-postgres://voxmesh:voxmesh_dev@postgres:5432/voxmesh?sslmode=disable}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-/migrations}"
MIGRATE_IMAGE="migrate/migrate:v4"

echo "Running migrations against $DB_URL..."

docker run --rm \
  --network voxmesh_default \
  -v "$(pwd)/migrations:$MIGRATIONS_DIR:ro" \
  "$MIGRATE_IMAGE" \
  -path "$MIGRATIONS_DIR" \
  -database "$DB_URL" \
  up

echo "Migrations complete."
