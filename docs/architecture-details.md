# HeartFolio - Architecture Details

## Project Structure

```
HeartFolio/
├── cmd/
│   └── server/
│       └── main.go                  # Application entrypoint & dependency wiring
├── api/
│   └── openapi.yaml                 # OpenAPI 3.0 spec (source of truth)
├── internal/
│   ├── config/
│   │   └── config.go                # Centralised configuration from environment
│   ├── db/
│   │   ├── mongo/
│   │   │   ├── client.go            # MongoDB client connect/disconnect
│   │   │   ├── collection.go        # Generic collection wrapper with CRUD
│   │   │   ├── database.go          # Database wrapper
│   │   │   └── repository.go        # Generic typed repository pattern
│   │   └── redis/
│   │       ├── client.go            # Redis client connect/disconnect
│   │       └── repository.go        # Redis operations (cache, sets, hashes)
│   ├── handler/
│   │   ├── swagger/
│   │   │   ├── swagger.go           # Serves Swagger UI + embedded OpenAPI spec
│   │   │   └── openapi.yaml         # Embedded copy of api/openapi.yaml
│   │   └── v1/
│   │       ├── user_handler.go      # HTTP handlers for user endpoints
│   │       ├── badge_handler.go     # HTTP handlers for badge endpoints
│   │       ├── quest_handler.go     # HTTP handlers for quest endpoints
│   │       └── admin/
│   │           ├── badge_handler.go # Admin badge management handlers
│   │           └── quest_handler.go # Admin quest management handlers
│   ├── middleware/
│   │   └── auth.go                  # JWT auth middleware & admin guard
│   ├── model/
│   │   ├── user.go                  # User domain model
│   │   ├── badge.go                 # Badge & UserBadge domain models
│   │   ├── quest.go                 # Quest & UserQuest domain models
│   │   └── responses.go             # Denormalised view types (UserBadgeDetail, UserQuestDetail)
│   ├── service/
│   │   ├── errors.go                # Domain-level sentinel errors
│   │   ├── user_service.go          # User business logic (auth, EXP)
│   │   ├── badge_service.go         # Badge business logic (CRUD, awarding)
│   │   └── quest_service.go         # Quest business logic (lifecycle, rewards)
│   └── webhttp/
│       ├── httpman.go               # Custom HTTP router with middleware support
│       └── jsonapi.go               # JSON:API response helpers (RespondOne, RespondMany, RespondError)
├── docs/
│   ├── features.pdf                 # Feature specification (Thai)
│   ├── outline.pdf                  # Project outline (Thai)
│   ├── install-instruction.md       # Installation & running guide
│   └── architecture-details.md      # This file
├── .env                             # Environment variables (not committed)
├── Dockerfile                       # Multi-stage Docker build
├── docker-compose.yml               # Full-stack compose (Go + Mongo + Redis)
├── go.mod                           # Go module definition
└── go.sum                           # Go dependency checksums
```

---

## Layered Architecture

The project follows a **three-layer architecture** that is standard in production Go applications:

```
┌──────────────────────────────────────────────────┐
│                   HTTP Layer                      │
│  cmd/server/main.go  →  handler/v1/*.go           │
│  (routing, middleware, request/response parsing)  │
├──────────────────────────────────────────────────┤
│                 Service Layer                     │
│  service/*.go                                     │
│  (business logic, validation, orchestration)      │
├──────────────────────────────────────────────────┤
│               Data Access Layer                   │
│  db/mongo/  &  db/redis/                          │
│  (generic repositories, caching)                  │
└──────────────────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|---------------|
| **Entrypoint** | `cmd/server/` | Loads config, creates dependencies, wires everything together, starts the HTTP server |
| **Handler** | `internal/handler/v1/` | Parses HTTP requests, calls service methods, writes HTTP responses. Contains no business logic |
| **Middleware** | `internal/middleware/` | Cross-cutting concerns: JWT authentication, admin authorisation. Uses typed context keys |
| **Service** | `internal/service/` | All business rules: user registration/login, badge awarding, quest lifecycle, EXP/reward loop |
| **Model** | `internal/model/` | Pure data structures (structs) with BSON/JSON tags. Zero dependencies on other packages |
| **Config** | `internal/config/` | Reads environment variables with sensible defaults. Single `Load()` entry point |
| **Database** | `internal/db/` | Generic `Repository[T]` pattern for MongoDB. Redis cache-aside wrapper |
| **Router** | `internal/webhttp/` | Lightweight HTTP router built on `http.ServeMux` with middleware chains and route groups |

---

## Directory Details

### `cmd/server/main.go`

The single application entrypoint. Follows the Go convention of `cmd/<binary-name>/main.go`. Responsibilities:

- Load configuration via `config.Load()`
- Connect to MongoDB and Redis
- Create repositories, services, and handlers (dependency injection by constructor)
- Ensure database indexes
- Register API routes and start the HTTP server

### `internal/config/`

**`config.go`** — Defines a `Config` struct and a `Load()` function that reads from environment variables with fallback defaults. All configuration is centralised here instead of scattered across `main.go`.

### `internal/model/`

Pure domain models with no imports from other internal packages.

**`user.go`** — `User` struct with fields: ID, Email, Password (hidden from JSON), Role, EXP, Level.

**`badge.go`** — `Badge` (library definition) and `UserBadge` (per-user badge ownership with level tracking).

**`quest.go`** — `Quest` (library definition with mood tags, EXP reward, difficulty, optional badge link) and `UserQuest` (per-user quest tracking with status: active/completed/abandoned).

**`responses.go`** — Denormalised view structs returned by services and consumed by handlers to build JSON:API resources. `UserBadgeDetail` joins a `UserBadge` with its `Badge`. `UserQuestDetail` joins a `UserQuest` with its `Quest`.

### `internal/middleware/`

**`auth.go`** — Contains:
- Typed context keys (`contextKey` type) to avoid string-key collisions
- `GenerateToken()` / `ValidateToken()` — JWT creation and parsing
- `Auth(secret)` — Middleware that extracts Bearer token, validates it, and injects user claims into context
- `RequireAdmin` — Middleware that checks the `role` context value
- Helper functions: `UserIDFromContext()`, `RoleFromContext()`

### `internal/service/`

Business logic layer. Services receive repositories via constructor injection.

**`user_service.go`** — Registration (bcrypt hashing), login (credential validation + JWT), profile retrieval (with Redis caching), EXP addition with automatic level calculation.

**`badge_service.go`** — Badge CRUD, awarding badges to users (create or upgrade), fetching user badges with details, calculating skill stats by category, ensuring MongoDB indexes.

**`quest_service.go`** — Quest CRUD, adaptive quest listing by mood tag, quest lifecycle (start → complete/abandon), **Reward Loop** (completing a quest awards EXP and optionally upgrades a linked badge), user quest detail retrieval, ensuring MongoDB indexes.

**`errors.go`** — Sentinel errors (`ErrNotFound`, `ErrInvalidCredentials`, `ErrQuestAlreadyActive`, etc.) used by handlers for clean HTTP status mapping.

### `internal/handler/swagger/`

**`swagger.go`** — Serves Swagger UI at `/docs/` (CDN-loaded) and the raw OpenAPI spec at `/docs/openapi.yaml`. The spec is embedded into the binary at compile time via Go's `//go:embed` directive — no external files are needed at runtime.

**`openapi.yaml`** — Embedded copy of `api/openapi.yaml`. Must be kept in sync with the source file manually (copy after each spec update).

### `internal/handler/v1/`

Thin HTTP handlers. Each handler:
1. Parses the request (path params, JSON body, context values)
2. Calls the appropriate service method
3. Maps service errors to HTTP status codes using `errors.Is()`
4. Writes a **JSON:API** response via `webhttp.RespondOne`, `webhttp.RespondMany`, or `webhttp.RespondError`

Each handler has a `Router()` method that returns a configured `*webhttp.Router` with its routes and middleware. Resource-building helper functions (e.g. `badgeResource()`, `questResource()`) live alongside the handler to keep marshalling logic co-located with the handler, not in the service.

### `internal/handler/v1/admin/`

Admin-only handlers for badge and quest management. These routers are mounted behind `Auth` + `RequireAdmin` middleware in `main.go`.

### `internal/db/mongo/`

Generic MongoDB data access layer:
- `Client` — wraps `mongo.Client`
- `Database` — wraps `mongo.Database`
- `Collection` — wraps `mongo.Collection` with generic CRUD (auto-generates ObjectID on insert)
- `Repository[T]` — typed generic repository providing `InsertOne`, `FindByID`, `FindOne`, `Find`, `UpdateByID`, `UpdateOne`, `DeleteByID`, `DeleteOne`

### `internal/db/redis/`

Redis data access:
- `Client` — wraps `redis.Client`
- `Repository` — provides `Set`, `Get`, `Del`, `Exists`, `Expire`, hash ops, set ops, and `GetOrElse` (cache-aside pattern)

### `internal/webhttp/`

Custom HTTP router built on Go 1.22+ `http.ServeMux`:

**`httpman.go`**
- Route methods: `Get`, `Post`, `Put`, `Delete`, `Patch`, `Options`, `Head`
- Middleware chaining via `Use()`
- Route grouping with `Group(prefix)`
- Sub-router mounting with `Register(prefix, handler)` / `Mount(prefix, handler)`
- Helpers: `JSON()`, `ReadJSON()`, `Param()`, `ParamInt()`

**`jsonapi.go`** — JSON:API response primitives:

| Function | Description |
|----------|-------------|
| `RespondOne(w, status, resource)` | Writes `{ "data": { ... } }` with `Content-Type: application/vnd.api+json` |
| `RespondMany(w, status, resources)` | Writes `{ "data": [ ... ] }` (empty slice on nil) |
| `RespondError(w, status, detail)` | Writes `{ "errors": [{ "status", "title", "detail" }] }` |
| `Resource` | Struct: `type`, `id`, `attributes`, `relationships` |
| `Relationship` | Struct: `data` (holds a nested `Resource`) |
| `ErrorObject` | Struct: `status`, `title`, `detail` |

---

## API Route Map

### Public Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/ping` | inline | Health check |
| POST | `/api/v1/users/register` | UserHandler.Register | Create account |
| POST | `/api/v1/users/login` | UserHandler.Login | Get JWT token |
| GET | `/api/v1/badges/` | BadgeHandler.ListBadges | List all badges |
| GET | `/api/v1/badges/{id}` | BadgeHandler.GetBadge | Get badge by ID |
| GET | `/api/v1/badges/users/{userID}/badges` | BadgeHandler.GetUserBadgesByID | Get user's badges |
| GET | `/api/v1/badges/users/{userID}/skill-stats` | BadgeHandler.GetUserSkillStatsByID | Get user's skill stats |
| GET | `/api/v1/quests/` | QuestHandler.ListQuests | List active quests |
| GET | `/api/v1/quests/{id}` | QuestHandler.GetQuest | Get quest by ID |
| GET | `/api/v1/quests/mood/{mood}` | QuestHandler.ListByMood | Adaptive quests by mood |

### Authenticated Routes (Bearer token required)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/users/me` | UserHandler.Me | Current user profile |
| POST | `/api/v1/badges/{id}/award` | BadgeHandler.AwardBadgeToSelf | Award badge to self |
| GET | `/api/v1/badges/me/badges` | BadgeHandler.GetMyBadges | My badges |
| GET | `/api/v1/badges/me/skill-stats` | BadgeHandler.GetMySkillStats | My skill stats |
| POST | `/api/v1/quests/{id}/start` | QuestHandler.StartQuest | Start a quest |
| POST | `/api/v1/quests/{userQuestID}/complete` | QuestHandler.CompleteQuest | Complete quest (earns EXP) |
| POST | `/api/v1/quests/{userQuestID}/abandon` | QuestHandler.AbandonQuest | Abandon a quest |
| GET | `/api/v1/quests/me/quests` | QuestHandler.GetMyQuests | My quests (filterable by ?status=) |

### Admin Routes (Bearer token + admin role required)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/api/v1/admin/badges/` | admin.BadgeHandler.CreateBadge | Create badge |
| GET | `/api/v1/admin/badges/` | admin.BadgeHandler.ListBadges | List all badges |
| GET | `/api/v1/admin/badges/{id}` | admin.BadgeHandler.GetBadge | Get badge |
| PUT | `/api/v1/admin/badges/{id}` | admin.BadgeHandler.UpdateBadge | Update badge |
| DELETE | `/api/v1/admin/badges/{id}` | admin.BadgeHandler.DeleteBadge | Delete badge |
| POST | `/api/v1/admin/badges/{badgeId}/award` | admin.BadgeHandler.AwardBadgeToUser | Award badge to any user |
| POST | `/api/v1/admin/quests/` | admin.QuestHandler.CreateQuest | Create quest |
| GET | `/api/v1/admin/quests/` | admin.QuestHandler.ListQuests | List all quests (incl. inactive) |
| GET | `/api/v1/admin/quests/{id}` | admin.QuestHandler.GetQuest | Get quest |
| PUT | `/api/v1/admin/quests/{id}` | admin.QuestHandler.UpdateQuest | Update quest |
| DELETE | `/api/v1/admin/quests/{id}` | admin.QuestHandler.DeleteQuest | Delete quest |

---

## MongoDB Collections & Indexes

| Collection | Indexes |
|------------|---------|
| `users` | (default `_id`) |
| `badges` | Unique on `name` |
| `user_badges` | Unique compound on `(user_id, badge_id)`, single on `user_id` |
| `quests` | Unique on `name`, single on `mood_tags`, single on `is_active` |
| `user_quests` | Compound on `(user_id, status)`, unique compound on `(user_id, quest_id, status)` |

---

## Key Design Decisions

1. **Generic Repository** — `Repository[T]` avoids writing repetitive CRUD for each model while keeping type safety.
2. **Service layer** — Keeps handlers thin and business logic testable in isolation.
3. **Typed context keys** — `type contextKey string` prevents accidental collision with other packages using string keys.
4. **Sentinel errors** — Enable handlers to map domain errors to HTTP status codes cleanly with `errors.Is()`.
5. **Optional Redis** — The app starts and functions fully without Redis; caching is an enhancement, not a requirement.
6. **Quest Reward Loop** — Completing a quest triggers: EXP award → user level recalculation → optional badge upgrade, all within a single service call.
7. **JSON:API** — All responses use the [JSON:API v1.0](https://jsonapi.org/) envelope (`{ "data": ... }` / `{ "errors": [...] }`). Response helpers live in `webhttp/jsonapi.go`; resource-building functions live alongside handlers. The `Content-Type` header is `application/vnd.api+json` on every response.
8. **Denormalised view types** — `model.UserBadgeDetail` and `model.UserQuestDetail` in `model/responses.go` are purpose-built structs returned by service join queries. Handlers consume them to build JSON:API `relationships` without performing extra lookups.
