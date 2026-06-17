.PHONY: up down ps logs build dev test test-integration migrate-up migrate-down clean help

# ── Docker ──
up:
	docker compose up -d

down:
	docker compose down

ps:
	docker compose ps

logs:
	docker compose logs -f

# ── Build ──
build:
	docker compose build

build-no-cache:
	docker compose build --no-cache

# ── Development ──
dev: up
	@echo "Infrastructure started. Run services individually:"
	@echo "  cd services/auth && go run ./cmd/main.go"
	@echo "  cd services/channel && go run ./cmd/main.go"
	@echo "  ... or use 'make dev-all'"

dev-infra:
	docker compose up -d postgres redis emqx minio

dev-all:
	@echo "Starting all services..."
	cd services/auth && go run ./cmd/main.go &
	cd services/channel && go run ./cmd/main.go &
	cd services/audio-mixer && go run ./cmd/main.go &
	cd services/presence && go run ./cmd/main.go &
	cd services/gateway-coordinator && go run ./cmd/main.go &
	cd services/ws-gateway && go run ./cmd/main.go &
	cd services/command-handler && go run ./cmd/main.go &
	cd services/notification && go run ./cmd/main.go &
	wait

dev-local:
	bash scripts/dev-start.sh

dev-stop:
	@cat .dev-pids 2>/dev/null | xargs kill 2>/dev/null || true
	pkill -f "go run.*services/" 2>/dev/null || true
	pkill -f "vite" 2>/dev/null || true
	@echo "All dev processes stopped."

# ── Testing ──
test:
	cd services/auth && go test ./... &
	cd services/channel && go test ./... &
	cd services/audio-mixer && go test ./... &
	cd services/presence && go test ./... &
	cd services/gateway-coordinator && go test ./... &
	cd services/ws-gateway && go test ./... &
	cd services/command-handler && go test ./... &
	cd services/notification && go test ./... &
	wait

test-integration:
	@echo "Requires 'make dev-infra' running first"
	cd services/auth && go test -tags=integration ./...
	cd services/channel && go test -tags=integration ./...

# ── Database ──
migrate-up:
	@echo "Run migrations manually or use golang-migrate:"
	@echo "  migrate -path migrations -database 'postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable' up"

migrate-down:
	@echo "migrate -path migrations -database 'postgres://voxmesh:voxmesh_dev@localhost:5432/voxmesh?sslmode=disable' down"

# ── Cleanup ──
clean:
	docker compose down -v
	rm -rf pgdata/ redisdata/ emqxdata/ emqxlog/ miniodata/ grafanadata/

# ── Help ──
help:
	@echo "VoxMesh Development Makefile"
	@echo ""
	@echo "  make up              Start all Docker services"
	@echo "  make down            Stop all Docker services"
	@echo "  make dev-infra       Start infrastructure only (DB, Redis, EMQX, MinIO)"
	@echo "  make build           Build all service images"
	@echo "  make test            Run unit tests"
	@echo "  make migrate-up      Run database migrations"
	@echo "  make clean           Remove all containers, volumes, and data"
