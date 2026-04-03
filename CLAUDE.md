# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HeartFolio is a gamified mental health and soft-skill development platform built in Go. Users earn EXP by completing quests (adaptive to mood), which unlocks and upgrades badges organized by skill category (resilience, emotional intelligence, etc.).

## Build & Run

```bash
# Install dependencies
go mod tidy

# Run locally (requires .env with JWT_SECRET)
go run ./cmd/server

# Build binary
go build -o server ./cmd/server

# Full stack with Docker (recommended)
docker compose up --build
```

**Required environment variables** (`.env`):
- `JWT_SECRET` — required, no default
- `MONGODB_URI` — default `mongodb://localhost:27017`
- `DB_NAME` — default `heartfolio`
- `REDIS_ADDR` — default `127.0.0.1:6379`
- `PORT` — default `8080`

Swagger UI is available at `/docs/` when the server is running.

## Architecture

**Three-layer clean architecture** wired via constructor-based DI in `cmd/server/main.go`:

```
internal/handler/v1/   — HTTP handlers (thin, no business logic)
internal/service/      — Business logic, validation, orchestration
internal/db/mongo/     — Generic Repository[T] for MongoDB CRUD
internal/db/redis/     — Cache-aside Redis wrapper
```

**Key supporting packages:**
- `internal/model/` — Domain structs with BSON/JSON tags
- `internal/middleware/` — JWT auth + admin role guard
- `internal/config/` — Env var loading with defaults
- `internal/webhttp/` — Custom router on top of `http.ServeMux` with middleware chaining and route grouping

## Key Patterns

**Generic Repository** — `internal/db/mongo/repository.go` provides a type-safe `Repository[T]` with `InsertOne`, `FindByID`, `FindOne`, `Find`, `UpdateByID`, etc. All collections use this; no collection-specific CRUD boilerplate.

**Sentinel errors** — Services return domain-level errors (e.g., `ErrNotFound`, `ErrDuplicateEmail`). Handlers use `errors.Is()` to map these to HTTP status codes.

**Typed context keys** — `type contextKey string` in `internal/middleware/` prevents string-key collisions; JWT claims are injected this way.

**Quest reward loop** — A single `QuestService.CompleteQuest()` call atomically: marks quest complete → awards EXP → recalculates user level → optionally upgrades a linked badge.

**Resource builders** — Handler files define `badgeResource()`, `questResource()`, etc. helpers to build JSON:API resource objects, co-located with their handler.

**Indexes on startup** — `BadgeService.EnsureIndexes()` and `QuestService.EnsureIndexes()` are called in `main.go`. Failures are logged as warnings but don't stop startup.

## API Conventions

All responses use **JSON:API v1.0** with `Content-Type: application/vnd.api+json`:

```json
// Single resource
{ "data": { "type": "users", "id": "...", "attributes": { ... } } }

// Collection
{ "data": [ { "type": "badges", "id": "...", "attributes": { ... } } ] }

// Error
{ "errors": [{ "status": "404", "title": "Not Found", "detail": "..." }] }
```

**Route structure:**
- `POST /api/v1/users/register`, `POST /api/v1/users/login` — public auth
- `/api/v1/badges/*`, `/api/v1/quests/*` — mixed public/authenticated
- `/api/v1/admin/badges/*`, `/api/v1/admin/quests/*` — admin role required
- `GET /ping` — health check

Authentication uses `Authorization: Bearer <JWT>` (HS256, 24h expiry).

## Database

MongoDB collections: `users`, `badges`, `user_badges`, `quests`, `user_quests`. Indexes are defined and created at startup, not via migrations.

Redis is optional — `UserService.GetByID()` uses cache-aside via `redis.Repository.GetOrElse()`. The app functions fully if Redis is unavailable.

## External Docs

- `api/openapi.yaml` — OpenAPI 3.0 spec (embedded in binary via `//go:embed`, served at `/docs/`)
- `docs/architecture-details.md` — detailed architecture and design decisions
- `docs/install-instruction.md` — Docker Compose setup and local dev guide
