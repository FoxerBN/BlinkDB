# BlinkDB

BlinkDB is a small educational TCP key-value server written in Go.

The goal is to learn Go networking, goroutines, in-memory storage, simple text protocols, Docker builds, and basic load testing by building a lightweight Redis-like server from scratch.

## Current Features

- TCP server listening on port `6379`
- one goroutine per client connection
- shared in-memory store protected by `sync.RWMutex`
- simple line-based protocol
- commands: `PING`, `STATUS`, `SET`, `GET`, `EXISTS`, `DELETE`, `CLEAR`, `HELP`, `QUIT`, `EXIT`
- configurable limits with `BLINKDB_*` environment variables
<!-- TODO: Add a config section that lists every BLINKDB_* variable, including
BLINKDB_SHUTDOWN_TIMEOUT, and explain the default values used by local runs and
Docker Compose. -->
- Docker and Docker Compose setup

## Run Locally

```bash
go run ./cmd/server
```

In another terminal:

```bash
nc localhost 6379
```

Example:

```text
PING
SET token 123456
GET token
EXISTS token
HELP
STATUS
QUIT
```

## Run With Docker

```bash
docker compose up --build
```

Stop the server:

```bash
docker compose down
```

## Status

This project is in early development.
