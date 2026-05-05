# ADR-002: SQLite WAL as Sole Shared State Layer

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Ruben Alvarez

## Context

Jart-Stow's hybrid architecture has two runtimes — Go (daemon, TUI, CLI) and Python (REST API) — that must share state. We considered gRPC, Unix domain sockets, message queues, and a shared database.

## Decision

Use a single SQLite database file in WAL mode as the sole shared state layer. Both runtimes access it directly from disk. No IPC layer between them.

## Consequences

**Positive:**
- Zero IPC complexity. No protobuf compilation, no socket management, no queue.
- SQLite is embedded, requires no server process.
- WAL mode allows concurrent reads while one process writes.
- The database is a single file, easily backed up or inspected.

**Negative:**
- Both runtimes must agree on schema. Changes require coordinated migrations.
- Python uses aiosqlite; Go uses mattn/go-sqlite3. Both must be kept compatible.

## Alternatives Considered

### gRPC between Go and Python
Rejected as overengineered for a single-machine tool. Adds protobuf compilation, service definitions, and TLS complexity with no meaningful benefit.

### Unix domain sockets
Rejected due to ad-hoc protocol design burden.

### Message queue (NATS, Redis)
Rejected as requiring an external service dependency for what is fundamentally a local tool.
