# HeartFolio - Installation & Running Guide

## Prerequisites

- **Go** 1.26.1+
- **MongoDB** 7.0+
- **Redis** 7.0+ (optional — the app gracefully falls back if unavailable)
- **Docker & Docker Compose** (optional — for containerised setup)

---

## Option 1: Run with Docker Compose (Recommended)

This is the easiest way to get everything running. Docker Compose will start MongoDB, Redis, and the Go application together.

```bash
# 1. Clone the repository
git clone https://github.com/NaphonJangjit/HeartFolio.git
cd HeartFolio

# 2. Create a .env file (or edit the existing one)
echo "JWT_SECRET=YourSuperSecretKeyHere" > .env

# 3. Start all services
docker compose up --build

# The API will be available at http://localhost:8080
```

To stop:

```bash
docker compose down
```

To stop and remove all data volumes:

```bash
docker compose down -v
```

---

## Option 2: Run Locally (Without Docker)

### 1. Install Dependencies

Make sure MongoDB and Redis are running on your machine (or remote).

```bash
# macOS (Homebrew)
brew install go mongodb-community redis
brew services start mongodb-community
brew services start redis

# Ubuntu/Debian
sudo apt install -y golang mongodb-org redis-server
sudo systemctl start mongod redis
```

### 2. Clone & Setup

```bash
git clone https://github.com/NaphonJangjit/HeartFolio.git
cd HeartFolio
```

### 3. Configure Environment Variables

Create a `.env` file in the project root or export variables directly:

| Variable         | Required | Default                  | Description                |
|------------------|----------|--------------------------|----------------------------|
| `JWT_SECRET`     | Yes      | —                        | Secret key for JWT signing |
| `MONGODB_URI`    | No       | `mongodb://localhost:27017` | MongoDB connection string  |
| `DB_NAME`        | No       | `heartfolio`             | MongoDB database name      |
| `REDIS_ADDR`     | No       | `127.0.0.1:6379`         | Redis address              |
| `REDIS_PASSWORD`  | No       | (empty)                  | Redis password             |
| `PORT`           | No       | `8080`                   | HTTP server port           |

Example `.env`:

```env
JWT_SECRET=YourSuperSecretKeyHere
MONGODB_URI=mongodb://localhost:27017
DB_NAME=heartfolio
REDIS_ADDR=127.0.0.1:6379
PORT=8080
```

### 4. Build & Run

```bash
# Download Go modules
go mod tidy

# Build
go build -o server ./cmd/server

# Run (load .env manually or use a tool like direnv)
export $(cat .env | xargs)
./server
```

The server will start on the configured port (default `8080`).

### 5. Verify

```bash
curl http://localhost:8080/ping
# Expected response: pong
```

---

## API Quick Start

### Register a user

```bash
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "secret123"}'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "secret123"}'
```

### Use the token

```bash
TOKEN="<token from login response>"

# Get current user profile
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"

# List available quests
curl http://localhost:8080/api/v1/quests/

# Start a quest
curl -X POST http://localhost:8080/api/v1/quests/<quest_id>/start \
  -H "Authorization: Bearer $TOKEN"
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `JWT_SECRET environment variable not set` | Make sure `.env` is loaded or export the variable |
| `Failed to connect to MongoDB` | Verify MongoDB is running: `mongosh --eval "db.runCommand({ping:1})"` |
| `Redis connection failed` | Redis is optional. The app will log a warning and continue without caching |
| Port already in use | Change `PORT` in `.env` or stop the conflicting process |
