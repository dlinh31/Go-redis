# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go-Redis is a Redis clone in Go — a TCP server implementing the RESP protocol with string/hash operations, Pub/Sub, AOF persistence, and active key expiration. No external dependencies; uses only the Go standard library. Listens on port 6379.

## Commands

```bash
# Run (development)
go run main.go

# Build
go build -o go-redis .

# Manual testing (requires redis-cli)
redis-cli -p 6379

# Docker (build and run via Compose)
docker compose up --build       # build image and start
docker compose up -d --build    # same, detached (background)
docker compose down             # stop and remove container
docker compose logs -f          # tail logs
```

There are no test files in this codebase. Validation is done manually via `redis-cli`.

## Docker

Two files handle containerization:

- **Dockerfile** — multi-stage build: Stage 1 compiles the binary using `golang:1.23-alpine`, Stage 2 copies only the binary into a minimal `alpine:3.21` image. `CGO_ENABLED=0` produces a fully static binary with no C dependencies.
- **docker-compose.yml** — defines the single `go-redis` service: builds from the local Dockerfile, maps port `6379:6379`, mounts `./data:/app/data` so the AOF file persists across container restarts, and sets `restart: unless-stopped`.

AOF persistence requires the volume mount — without it, `database.aof` is lost when the container stops.

## Architecture

All code lives in the `main` package — no subpackages. Six files with clear separation of concerns:

- **main.go** — TCP listener on `:6379`, per-connection goroutines, AOF replay on startup, expiration goroutine initialization
- **resp.go** — Full RESP parser and serializer (bulk strings, arrays, simple strings, errors, integers)
- **handler.go** — Command routing via `Handlers map[string]func([]Value) Value`, in-memory storage (`SETs`, `HSETs`), passive expiration on read
- **aof.go** — Append-only file at `./data/database.aof`; syncs every 1 second; replays on startup by re-executing commands through the Handlers map
- **expiration.go** — Background goroutine runs every 100ms, randomly samples 20 keys, deletes expired ones, logs DEL to AOF; adaptive: stops early if <10% of sample is expired
- **pubsub.go** — `PubSubManager` singleton; each SUBSCRIBE spawns a listener goroutine with a buffered channel (cap 100); PUBLISH is non-blocking (drops if full)

## Key Data Structures

```go
// String storage (handler.go)
type StringEntry struct {
    Value     string
    ExpiresAt *time.Time
}
var SETs map[string]StringEntry
var SETsMu sync.RWMutex

// Hash storage (handler.go)
var HSETs map[string]map[string]string
var HSETsMu sync.RWMutex

// Command router (handler.go)
var Handlers map[string]func([]Value) Value
```

## Important Behaviors

**AOF and EXPIRE:** When a client sends `EXPIRE key seconds`, the server writes `EXPIREAT key <unix_timestamp>` to AOF (not `EXPIRE`) so the absolute deadline is preserved across restarts.

**Expiration is dual-path:** Active (background sampler) + passive (checked inside GET/DEL handlers on every access).

**SUBSCRIBE is special-cased in main.go** — it doesn't go through the normal `Handlers` map; instead `handleSubscription()` is called directly and holds the connection open.

**AOF replay:** On startup, `aof.Read()` parses stored RESP commands and invokes them through `Handlers` to reconstruct state before accepting client connections.

**Thread safety:** `SETsMu` and `HSETsMu` use `sync.RWMutex`. GET uses `RLock` initially, then acquires a full `Lock` only when a lazily-detected expiration requires deletion.
