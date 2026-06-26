# miniredis

MiniRedis is a small educational TCP key-value server written in Go.

The goal of the project is to learn Go networking, goroutines, in-memory storage, simple text protocols, Docker builds, and basic load testing by building a lightweight Redis-like server from scratch.

## Current Features

- TCP server listening on port `6379`
- one goroutine per client connection
- shared in-memory store protected by `sync.RWMutex`
- simple line-based protocol
- `PING` command returning `+PONG`
- `STATUS` command returning server/store status
- `QUIT` and `EXIT` commands
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

This project is in early development. `SET`, `GET`, and `DELETE` are planned next.
