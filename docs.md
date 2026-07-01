# BlinkDB — Code Documentation

A file-by-file guide to the codebase: what each file holds, what its functions do,
and which other files they call. Read top to bottom to understand how a request
flows from a TCP connection down to the in-memory store.

## Architecture at a Glance

```
cmd/server/main.go            ← entrypoint: wires everything, handles signals
        │ builds
        ▼
internal/config/config.go     ← loads .env + env vars into a typed Config
        │ passed as Options
        ▼
internal/network/server.go    ← accepts TCP conns, rate limiting, shutdown
        │ per connection
        ▼
internal/network/handler.go   ← reads lines, runs commands, writes replies
        │ parses with          │ reads/writes
        ▼                       ▼
protocol.go (ParseCommand)   internal/store/store.go (the key-value map)
```

Dependency direction: `main` → `config`, `network`, `store`. `network` → `store`.
Nothing depends back on `main`; `store` depends on nothing internal.

---

## `cmd/server/main.go` — entrypoint (`package main`)

Boots the whole server and blocks until shutdown.

- **`main()`** — sets up logging; calls `config.Load(".env")`; applies the memory
  target via `debug.SetMemoryLimit`; creates the store with `store.NewStore()`;
  builds the server with `network.NewServer(...)` passing a `network.Options`
  filled from `Config`; runs `srv.Start()` in a goroutine; waits on either a start
  error or a `SIGINT`/`SIGTERM` signal, then calls `srv.Shutdown()` for a clean stop.

**Calls into:** `config.Load`, `store.NewStore`, `network.NewServer`,
`srv.Start`, `srv.Shutdown`.

---

## `internal/config/config.go` — configuration (`package config`)

Turns environment variables (and an optional `.env` file) into a typed `Config`.

- **`Config` struct** — all tunables: port, memory, client/value limits, rate
  limits, and the four timeouts.
- **`Load(path)`** — reads `.env`, then builds `Config` from env vars with defaults.
- **`loadDotEnv(path)`** — parses `KEY=value` lines from `.env` and sets them as env
  vars, without overwriting anything already set in the environment.
- **`envString / envInt / envDuration`** — typed getters with fallbacks;
  `envDuration` accepts Go durations (`30s`) or plain seconds (`30`).

**Called by:** `main.main()`. **Depends on:** stdlib only.

---

## `internal/store/store.go` — in-memory storage (`package store`)

The thread-safe key-value map. Pure data layer, knows nothing about networking.

- **`Store` struct** — a `map[string]string` guarded by a `sync.RWMutex`.
- **`NewStore()`** — creates an empty store.
- **`Set(key, value)`** — writes a value (write-locked).
- **`Get(key) (value, ok)`** — reads a value (read-locked).
- **`Delete(key) bool`** — removes a key; returns whether it existed.
- **`Exists(key) bool`** — reports whether a key is present.
- **`Count() int`** — number of stored keys (used by `STATUS`).
- **`Clear()`** — replaces the map with a fresh empty one.

**Called by:** `network` (server + handler). **Depends on:** stdlib `sync` only.

---

## `internal/network/protocol.go` — command parsing (`package network`)

Validates raw input lines before the handler executes them.

- **`Command` struct** — parsed `Name` (uppercased) and `Args`.
- **`ParseCommand(input) (*Command, error)`** — trims, splits into fields,
  uppercases the name, and validates argument counts and key validity per command.
  Rejects unknown commands and bad keys.
- **`validKey(key) bool`** — enforces non-empty keys ≤ `maxKeyBytes` (512) with no
  whitespace or control characters.

**Called by:** `handler.handleConnection`. **Depends on:** stdlib only.

---

## `internal/network/server.go` — TCP server & rate limiting (`package network`)

Owns the listener, connection lifecycle, client accounting, rate limiting, and
graceful shutdown.

### Server
- **`Options` struct** — runtime limits/timeouts passed in from `main`.
- **`Server` struct** — port, `*store.Store`, options, active-client counter,
  rate limiter, listener, and the set of active connections (for shutdown).
- **`NewServer(port, db, options)`** — constructs the server and its rate limiter.
- **`Start()`** — opens the TCP listener and loops on `Accept()`. For each
  connection it reserves a client slot (`tryAddClient`), registers the conn
  (`addConnection`), and spawns a goroutine running `handleConnection` (in
  `handler.go`). Returns cleanly when shutting down.
- **`Shutdown()`** — stops accepting, closes the listener, waits for in-flight
  connections up to `ShutdownTimeout`, then force-closes any that remain.
- **`Addr()`** — exported accessor returning the listen address (or nil before
  bind); lets external tests discover the OS-assigned port.
- **`isShuttingDown` / `addConnection` / `removeConnection` / `closeActiveConnections`**
  — connection bookkeeping under `mu`.
- **`tryAddClient` / `removeClient` / `activeClientCount`** — atomic client-slot
  counter enforcing `MaxClients`.
- **`clientIP(conn)`** — extracts the bare IP (for per-IP rate limiting).

### Rate limiter
- **`rateLimiter` struct** — a global bucket plus a `perIP` map of buckets,
  with a `lastCleanup` timestamp.
- **`rateBucket` struct** — one fixed one-second window: `window`, `count`, and
  `lastSeen`.
- **`newRateLimiter(globalPS, ipPS)`** — buckets are disabled when limits ≤ 0.
- **`allow(ip) bool`** — the hot path: runs `cleanupLocked`, checks the global
  bucket, then the per-IP bucket; updates `lastSeen`.
- **`cleanupLocked(now)`** — at most once per minute, deletes per-IP buckets unused
  for longer than `ipBucketTTL` (10m) so the map cannot grow unbounded.
- **`allowBucket(bucket, limit, now)`** — resets the window each second and rejects
  once the count reaches the limit.

**Calls into:** `handler.handleConnection`, `store` (via `s.db`).
**Called by:** `main`. Tested by `server_test.go`.

---

## `internal/network/handler.go` — connection & command handling (`package network`)

Drives one client from connect to disconnect and executes commands against the store.

- **`handleConnection(conn)`** — the per-client loop: logs connect/disconnect,
  sends the greeting (`statusResponse`), then repeatedly sets the read deadline
  (`setReadDeadline`), scans a line, rate-limits (`rateLimiter.allow`), parses
  (`ParseCommand`), executes (`executeCommand`), and writes the reply
  (`writeResponse`).
- **`executeCommand(cmd) (response, shouldClose)`** — the command switch:
  `PING`, `EXISTS`, `HELP`, `STATUS`, `SET` (size-checked via `checkValueSize`),
  `GET`, `DELETE`, `CLEAR`, `QUIT`/`EXIT`. Talks to the store through `s.db`.
- **`helpResponse()`** — the static `HELP` text.
- **`statusResponse()`** — builds the `+STATUS OK clients=… keys=… mem_used_mb=…
  mem_sys_mb=…` line; reads live memory via `runtime.ReadMemStats`. Used for the
  greeting and the `STATUS` command.
- **`writeResponse(conn, s)`** — applies the write deadline and writes one reply.
- **`setReadDeadline(conn)`** — bounds how long to wait for the next command;
  prefers `IdleTimeout`, falls back to `ReadTimeout`.
- **`setWriteDeadline(conn)`** — applies `WriteTimeout` before a write.
- **`setDeadline(conn, timeout, setter)`** — shared helper for read/write deadlines
  (no-op when the timeout is ≤ 0).
- **`checkValueSize(value) error`** — rejects values larger than `MaxValueBytes`.
- **`maxCommandBytes()`** — the scanner's max line size (`MaxValueBytes` + overhead).

**Calls into:** `protocol.ParseCommand`, `store` (via `s.db`), the rate limiter.

---

## `test/` — load-test CLI + unit tests (`package main`)

Everything test-related lives here as one `package main`. The load runner and the
`*_test.go` unit tests share the package; `go build` / `go run` ignore `_test.go`
files, so the two coexist. Run unit tests with `go test ./test/` and the load
runner with `cd test && go run . -test <name>`.

**Load-test CLI** (hammers a *running* server over TCP; imports nothing internal
except through the socket):

- **`config-test.go`** — the harness: `TestConfig`, the `tests` registry,
  `main()` (selects a scenario by `-test` flag), `registerTest`, and `runTest`
  which spawns `Users` goroutines and reports processed/failed/rate.
- **`stress-test.go`** — registers the `stress` scenario: connect, read greeting,
  send `STATUS`, validate the reply. Measures raw throughput.
- **`set-get-test.go`** — registers the `set-get` scenario: seeds two values once,
  then each simulated user connects and `GET`s its key, checking the exact value.

**Unit tests** (import `blinkdb/internal/network` and `blinkdb/internal/store`):

- **`protocol_test.go`** — tests `network.ParseCommand` for key validation, `HELP`,
  and `EXISTS` argument handling.
- **`server_test.go`** — tests graceful shutdown: starts a real server on port `0`,
  reads the listen address via the exported `srv.Addr()`, connects a client, calls
  `Shutdown()`, and verifies the connection is closed. Because it lives in
  `package main` (not `network`) it uses `Addr()` instead of the unexported
  `listener`/`mu` fields.
</content>
