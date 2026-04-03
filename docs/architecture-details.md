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
│   │   ├── user.go                  # User domain model (incl. PrivacySettings, Tier)
│   │   ├── badge.go                 # Badge & UserBadge domain models
│   │   ├── quest.go                 # Quest & UserQuest domain models
│   │   ├── mood.go                  # MoodEntry domain model
│   │   └── responses.go             # Denormalised view types (SkillStat, Portfolio, QuestSummary, etc.)
│   ├── service/
│   │   ├── errors.go                # Domain-level sentinel errors
│   │   ├── validate.go              # Input validation helpers (email, password, mood)
│   │   ├── user_service.go          # User business logic (auth, EXP, privacy, profile)
│   │   ├── badge_service.go         # Badge business logic (CRUD, awarding, detailed stats)
│   │   ├── quest_service.go         # Quest business logic (lifecycle, rewards)
│   │   ├── mood_service.go          # Mood tracking (submit, history)
│   │   └── portfolio_service.go     # Emotional portfolio aggregation
│   └── webhttp/
│       ├── httpman.go               # Custom HTTP router with middleware support
│       ├── jsonapi.go               # JSON:API response helpers (RespondOne, RespondMany, RespondError)
│       ├── cors.go                  # CORS middleware with configurable origins/methods/headers
│       └── pagination.go            # Pagination helpers (ParsePage, Page.Apply, RespondManyPaginated)
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
| **Entrypoint** | `cmd/server/` | Loads config, creates dependencies, wires everything together, starts the HTTP server with graceful shutdown |
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
- Apply global middleware (CORS)
- Register API routes and start the HTTP server
- Graceful shutdown on SIGINT/SIGTERM (10s drain timeout)
- Structured logging via `log/slog` (JSON handler to stdout)

### `internal/config/`

**`config.go`** — Defines a `Config` struct and a `Load()` function that reads from environment variables with fallback defaults. All configuration is centralised here instead of scattered across `main.go`.

### `internal/model/`

Pure domain models with no imports from other internal packages.

**`user.go`** — `User` struct with fields: ID, Email, Password (hidden from JSON), Role, EXP, Level, Privacy. Also contains `PrivacySettings` (ShowBadges, ShowStats, ShowQuests) and a `Tier()` method that returns the user's tier (bronze/silver/gold/platinum/diamond) based on level.

**`badge.go`** — `Badge` (library definition) and `UserBadge` (per-user badge ownership with level tracking).

**`quest.go`** — `Quest` (library definition with mood tags, EXP reward, difficulty, optional badge link) and `UserQuest` (per-user quest tracking with status: active/completed/abandoned).

**`mood.go`** — `MoodEntry` struct with fields: ID, UserID, Mood, Note, CreatedAt. Records the user's emotional state over time.

**`responses.go`** — Denormalised view structs returned by services and consumed by handlers to build JSON:API resources:
- `UserBadgeDetail` — joins a `UserBadge` with its `Badge`
- `UserQuestDetail` — joins a `UserQuest` with its `Quest`
- `SkillBadgeInfo` — individual badge's contribution to a skill category
- `SkillStat` — per-category breakdown with badges, total level, badge count
- `QuestSummary` — aggregated quest completion statistics
- `Portfolio` — full emotional portfolio aggregation for a user

### `internal/middleware/`

**`auth.go`** — Contains:
- Typed context keys (`contextKey` type) to avoid string-key collisions
- `GenerateToken()` / `ValidateToken()` — JWT creation and parsing
- `Auth(secret)` — Middleware that extracts Bearer token, validates it, and injects user claims into context
- `RequireAdmin` — Middleware that checks the `role` context value
- Helper functions: `UserIDFromContext()`, `RoleFromContext()`

### `internal/service/`

Business logic layer. Services receive repositories via constructor injection.

**`user_service.go`** — Registration (bcrypt hashing, default privacy settings, email/password validation), login (credential validation + JWT), profile retrieval (with Redis caching), EXP addition with automatic level calculation, privacy settings update, profile update (email change with uniqueness check), password change (verifies old password), email uniqueness index creation.

**`validate.go`** — Input validation helpers shared across services. `ValidateEmail()` checks format via regex, `ValidatePassword()` enforces minimum 8 characters, `ValidateMood()` validates against a whitelist of 16 allowed mood values (case insensitive). Also exports `ValidMoods()` for documentation.

**`badge_service.go`** — Badge CRUD, awarding badges to users (create or upgrade), fetching user badges with details, calculating skill stats by category (both simple and detailed per-badge breakdown), ensuring MongoDB indexes.

**`quest_service.go`** — Quest CRUD, adaptive quest listing by mood tag, quest lifecycle (start → complete/abandon), **Reward Loop** (completing a quest awards EXP and optionally upgrades a linked badge), user quest detail retrieval, ensuring MongoDB indexes.

**`mood_service.go`** — Mood entry submission and retrieval. `Submit()` records a mood entry, `GetRecent()` returns the N most recent entries, `GetHistory()` returns all entries.

**`portfolio_service.go`** — Aggregates user profile, badges, detailed skill stats, quest summary, and recent moods into a single `Portfolio` struct. Powers the Emotional Portfolio endpoint.

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
| `RespondManyPaginated(w, status, resources, page)` | Writes `{ "data": [...], "meta": { "total", "offset", "limit" } }` |
| `RespondError(w, status, detail)` | Writes `{ "errors": [{ "status", "title", "detail" }] }` |
| `Resource` | Struct: `type`, `id`, `attributes`, `relationships` |
| `Relationship` | Struct: `data` (holds a nested `Resource`) |
| `ErrorObject` | Struct: `status`, `title`, `detail` |

**`pagination.go`** — Pagination helpers for JSON:API list endpoints:

| Function / Type | Description |
|-----------------|-------------|
| `Page` | Struct: `Offset`, `Limit` |
| `ParsePage(r)` | Parses `page[offset]` and `page[limit]` query params with defaults (0, 20) and max limit (100) |
| `Page.Apply(resources)` | Slices a resource list and returns `(page, total)` |
| `RespondManyPaginated` | Combines Apply + response with pagination meta |

**`cors.go`** — CORS middleware:

| Function / Type | Description |
|-----------------|-------------|
| `CORSConfig` | Struct: `AllowOrigins`, `AllowMethods`, `AllowHeaders`, `MaxAge` |
| `DefaultCORSConfig()` | Returns permissive defaults (`*` origin, standard methods/headers, 86400s max-age) |
| `CORS(cfg)` | Middleware that sets CORS headers on all responses and returns 204 for OPTIONS preflight |

---

## API Route Map

### Public Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/ping` | inline | Health check |
| POST | `/api/v1/users/register` | UserHandler.Register | Create account |
| POST | `/api/v1/users/login` | UserHandler.Login | Get JWT token |
| GET | `/api/v1/users/{userID}/portfolio` | UserHandler.GetUserPortfolio | User's emotional portfolio (respects privacy) |
| GET | `/api/v1/badges/` | BadgeHandler.ListBadges | List all badges |
| GET | `/api/v1/badges/{id}` | BadgeHandler.GetBadge | Get badge by ID |
| GET | `/api/v1/badges/users/{userID}/badges` | BadgeHandler.GetUserBadgesByID | Get user's badges (respects privacy) |
| GET | `/api/v1/badges/users/{userID}/skill-stats` | BadgeHandler.GetUserSkillStatsByID | Get user's detailed skill stats (respects privacy) |
| GET | `/api/v1/quests/` | QuestHandler.ListQuests | List active quests |
| GET | `/api/v1/quests/{id}` | QuestHandler.GetQuest | Get quest by ID |
| GET | `/api/v1/quests/mood/{mood}` | QuestHandler.ListByMood | Adaptive quests by mood |

### Authenticated Routes (Bearer token required)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/users/me` | UserHandler.Me | Current user profile (incl. tier, privacy) |
| PATCH | `/api/v1/users/me` | UserHandler.UpdateProfile | Update email |
| POST | `/api/v1/users/me/password` | UserHandler.ChangePassword | Change password (requires old password) |
| PATCH | `/api/v1/users/me/privacy` | UserHandler.UpdatePrivacy | Update privacy settings |
| POST | `/api/v1/users/me/mood` | UserHandler.SubmitMood | Submit a mood entry |
| GET | `/api/v1/users/me/moods` | UserHandler.GetMoodHistory | Get mood history |
| GET | `/api/v1/users/me/portfolio` | UserHandler.GetMyPortfolio | My emotional portfolio |
| POST | `/api/v1/badges/{id}/award` | BadgeHandler.AwardBadgeToSelf | Award badge to self |
| GET | `/api/v1/badges/me/badges` | BadgeHandler.GetMyBadges | My badges |
| GET | `/api/v1/badges/me/skill-stats` | BadgeHandler.GetMySkillStats | My detailed skill stats |
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
| `users` | Unique on `email` |
| `badges` | Unique on `name` |
| `user_badges` | Unique compound on `(user_id, badge_id)`, single on `user_id` |
| `quests` | Unique on `name`, single on `mood_tags`, single on `is_active` |
| `user_quests` | Compound on `(user_id, status)`, unique compound on `(user_id, quest_id, status)` |
| `moods` | Compound on `(user_id, created_at desc)` |

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
9. **Privacy Control** — Users can toggle visibility of badges, stats, and quests via `PrivacySettings`. Public endpoints (user badges, skill stats, portfolio) check these settings before returning data, returning 403 for hidden content.
10. **Tier/Ranking** — Automatic tier calculation based on user level: bronze (1–5), silver (6–15), gold (16–30), platinum (31–50), diamond (51+). Computed at read time via `User.Tier()`, not stored.
11. **Mood Tracking** — `MoodEntry` records the user's emotional state over time. Recent moods are included in the portfolio and can power adaptive quest recommendations.
12. **Emotional Portfolio** — `PortfolioService` aggregates user profile, badges, detailed skill stats, quest completion summary, and recent moods into a single response. This is the main showcase endpoint for verifiable soft-skill assets.
13. **Detailed Skill Stats** — `GetDetailedSkillStats()` returns per-category breakdowns with individual badge contributions (name, level, max level), replacing the simple `map[string]int32` for richer display.
14. **CORS Middleware** — Configurable CORS middleware applied globally in `main.go`. Handles OPTIONS preflight with 204 and sets `Access-Control-*` headers on all responses. Default config allows all origins (`*`) for development.
15. **Graceful Shutdown** — The server listens for SIGINT/SIGTERM via `signal.Notify` and calls `srv.Shutdown(ctx)` with a 10-second drain timeout, allowing in-flight requests to complete before exit.
16. **Structured Logging** — All logging uses `log/slog` with a JSON handler to stdout, replacing the standard `log` package. This provides structured key-value logging suitable for production log aggregation.
17. **Input Validation** — Centralised in `service/validate.go`. Email format is validated via regex, passwords require a minimum of 8 characters, and mood values are checked against a whitelist of 16 allowed values (case insensitive). Validation runs at the service layer so it applies regardless of the entry point.
18. **Pagination** — All list endpoints support JSON:API-style pagination via `page[offset]` and `page[limit]` query parameters. Responses include a `meta` object with `total`, `offset`, and `limit`. Default page size is 20, maximum is 100. Pagination is applied in-memory via `Page.Apply()` which is simple and sufficient for current data volumes.
19. **Profile Management** — Users can update their email via `PATCH /users/me` (with uniqueness enforcement) and change their password via `POST /users/me/password` (requires old password verification).
20. **Unit Tests** — Test coverage for critical non-DB components: JWT auth middleware, JSON:API response helpers, CORS middleware, pagination, input validation, and model methods. Tests use `net/http/httptest` for HTTP handler testing.
