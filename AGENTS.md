# Repository Guidelines

## Project Structure & Module Organization

This repository is a small Go module named `miniredis`, an educational Redis-like TCP key-value server.

- `cmd/server/main.go` is the executable entry point and wires the server to the in-memory store.
- `internal/network/` contains TCP listener, connection handling, and command parsing logic.
- `internal/store/` contains the concurrency-safe in-memory key/value store.
- `README.md` documents current features and manual usage.
- `Dockerfile` and `docker-compose.yml` provide containerized local runs.

Keep new application code under `internal/` unless it is a runnable command, which should live under `cmd/<name>/`.

## Build, Test, and Development Commands

- `go run ./cmd/server` starts the TCP server on port `6379`.
- `nc localhost 6379` connects to the running server for manual protocol testing.
- `go test ./...` runs all Go tests once test files are added.
- `go build ./cmd/server` builds the server binary.
- `docker compose up --build` builds and runs the service in Docker.
- `docker compose down` stops the Docker Compose stack.

Run `gofmt` before committing changed Go files, for example: `gofmt -w internal/network internal/store cmd/server`.

## Coding Style & Naming Conventions

Use standard Go formatting and idioms: tabs from `gofmt`, short package names, exported identifiers only when needed outside the package, and clear error wrapping with `%w`. Keep protocol command names uppercase (`PING`, `STATUS`, `QUIT`) while preserving user-provided key/value casing. Prefer small functions around parsing, connection handling, and store operations so concurrency and protocol behavior remain easy to test.

## Testing Guidelines

Use Go's built-in `testing` package. Place tests beside implementation files with names like `protocol_test.go` or `store_test.go`, and name test functions by behavior, such as `TestParseCommandRejectsEmptyInput`. Prioritize table-driven tests for command parsing and store behavior. Add network tests carefully, using local listeners and cleanup to avoid port conflicts. Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines

The current history uses short imperative commit messages, for example `ParseCommand func` and `Initial commit`. Keep commits focused and describe the changed behavior. Pull requests should include a short summary, test results, and any protocol or Docker changes. Link issues when applicable and include terminal examples when changing client-visible command behavior.

## Agent-Specific Instructions

Do not overwrite user changes. Before broad refactors, inspect related files and keep changes tightly scoped to the requested behavior.
