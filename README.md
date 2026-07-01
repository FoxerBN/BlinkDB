# BlinkDB

BlinkDB is a small educational TCP key-value server written in Go — a lightweight,
Redis-like server built from scratch to learn Go networking, goroutines, in-memory
storage, simple text protocols, Docker, and load testing.

## Features

- TCP server, one goroutine per client connection
- Shared in-memory store protected by `sync.RWMutex`
- Simple line-based text protocol (works with `nc` / `telnet`)
- Commands: `PING`, `STATUS`, `SET`, `GET`, `EXISTS`, `DELETE`, `CLEAR`, `HELP`, `QUIT`, `EXIT`
- `STATUS` reports live client count, key count, and runtime memory usage
- Per-server and per-IP rate limiting with automatic cleanup of stale IP buckets
- Configurable limits and timeouts via `BLINKDB_*` environment variables
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Docker and Docker Compose setup

## Run Locally

```bash
go run ./cmd/server
```

In another terminal:

```bash
nc localhost 6379
```

Example session:

```text
PING
SET token 123456
GET token
EXISTS token
DELETE token
CLEAR
STATUS
HELP
QUIT
```

## Run With Docker

```bash
docker compose up --build   # start
docker compose down         # stop
```

## Configuration

All settings are read from environment variables (loaded from `.env` if present).
Values accept Go durations (`30s`, `5m`) or plain seconds (`30`) for timeouts.

| Variable | Default | Meaning |
|---|---|---|
| `BLINKDB_PORT` | `6379` | TCP listen port |
| `BLINKDB_MEMORY_MB` | `256` | Go runtime soft memory target (`debug.SetMemoryLimit`) |
| `BLINKDB_MAX_CLIENTS` | `15000` | Max concurrent connections (0 = unlimited) |
| `BLINKDB_MAX_VALUE_BYTES` | `1048576` | Max size of a single value (0 = unlimited) |
| `BLINKDB_GLOBAL_RATE_LIMIT_PER_SECOND` | `50000` | Server-wide command rate cap (0 = off) |
| `BLINKDB_IP_RATE_LIMIT_PER_SECOND` | `15000` | Per-IP command rate cap (0 = off) |
| `BLINKDB_READ_TIMEOUT` | `30s` | Fallback read deadline when idle timeout is unset |
| `BLINKDB_WRITE_TIMEOUT` | `5s` | Deadline for writing a response |
| `BLINKDB_IDLE_TIMEOUT` | `30s` | How long a client may stay silent before disconnect |
| `BLINKDB_SHUTDOWN_TIMEOUT` | `5s` | Grace period for connections to finish on shutdown |

## Tests

```bash
go test ./test/                  # unit tests
cd test && go run . -test stress # manual load test (server must be running)
```

## Status

Early development / learning project.
</content>
</invoke>
