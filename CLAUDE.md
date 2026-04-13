# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Recharge-go is a phone recharge platform backend (Go) with a Vue 3 admin frontend (`web/`). It handles order management, multi-platform recharge integration, balance/credit tracking, notifications, and background task processing.

## Build & Run Commands

```bash
# Build individual services
make build-server          # Main HTTP API server
make build-recharge        # Recharge processing worker
make build-task            # Background task processor
make build-notification    # Notification service
make build-pullorder       # External order puller
make build-migrate         # Database migration tool

# Run locally
make run-server            # go run cmd/server/main.go
make run-recharge          # go run cmd/recharge/main.go

# Build all for Linux (default target)
make build-all-linux

# Run tests
go test ./...
go test ./internal/service/...   # Single package

# Frontend (from web/)
cd web && pnpm install && pnpm dev
```

Config file path is set via `-config` flag (default: `configs/config.yaml`).

## Architecture

### Multi-Service Design

Six binaries in `cmd/`, each with its own `main.go`:
- **server** — HTTP API (Gin), serves both internal admin and external partner APIs
- **recharge** — Worker that processes recharge orders against external platforms
- **task** — Cron-based background jobs (order status checks, statistics aggregation)
- **notification** — Push notification delivery
- **pullorder** — Pulls orders from external platforms on schedule
- **migrate** — Database migration runner

### Backend Layered Architecture

```
cmd/*/main.go → internal/app (Container) → router → controller → service → repository → model
```

- **Container** (`internal/app/container.go`): DI container that wires all repositories, services, and controllers. Initialized via `NewContainerWithConfigAndService()`. All services are created here with their dependencies.
- **Router** (`internal/router/`): Route registration. `router_v2.go` is the main entry point using the DI container's controller set. Individual route files (e.g., `order.go`, `user.go`) register route groups on `*gin.RouterGroup`.
- **Controller** (`internal/controller/`): Thin HTTP handlers, extract params and call services.
- **Service** (`internal/service/`): Business logic. Key services:
  - `RechargeService` — Core recharge logic, platform API integration
  - `OrderService` — Order lifecycle (create, success, fail, refund)
  - `UnifiedOrderService` — Unified order processing with retry support
  - `UnifiedRefundService` — Refund processing with distributed locks
  - `BalanceService` / `CreditService` — User balance and credit management
  - Sub-packages: `notification/`, `platform/`, `recharge/`, `pullorder/`, `zhangyu/`
- **Repository** (`internal/repository/`): Data access via GORM. Most repos are concrete structs (not interfaces) taking `*gorm.DB`.
- **Model** (`internal/model/`): GORM models.

### Key Infrastructure (`pkg/`)

- `database/` — GORM MySQL setup, auto-migration
- `redis/` — Redis client wrapper
- `queue/` — Asynq-based task queue (Redis-backed)
- `lock/` — Distributed locks for refund processing
- `log/` — Zap-based structured logging
- `metrics/` — Prometheus metrics
- `middleware/` — Security middleware (JWT auth, rate limiting, CORS)

### External Platform Integration

Multiple recharge platforms are supported. Each has its own route, controller, and service layer:
- MF178 (`mf178_order.go`)
- Kekebang (`kekebang_order.go`)
- Xianyinke (`xianyinke_order.go`)
- Daichong (`daichong_order.go`)

Signature handling is in `internal/signature/`. External APIs authenticate via API keys (`ExternalAPIKey` model) or platform tokens.

### Circular Dependency Resolution

Some services have circular dependencies (e.g., `RechargeService` ↔ `OrderService`). These are resolved using setter injection (`SetRechargeService()`, `SetRetryService()`) after construction.

### Database

- MySQL with GORM ORM
- Migrations in `migrations/` using golang-migrate (sequential versioned SQL files)
- Auto-migration runs on server startup via `database.AutoMigrateDB()`

### Frontend (`web/`)

Vue 3 + TypeScript + Vite 6 admin panel based on SoybeanAdmin template. Uses NaiveUI, Pinia, UnoCSS. API calls go through `web/src/service/request/` (Axios-based). The frontend connects to the Go server's API.

### Configuration

YAML-based via Viper (`configs/config.yaml`). Key sections: `server`, `database`, `redis`, `jwt`, `task`, `retry_task`, `security`, `third_party_api`. Environment-specific configs supported via the `-config` flag.
