# ADR-009: Python Textual TUI as Primary Interactive Interface

**Status:** Accepted
**Date:** 2026-05-13
**Deciders:** Ruben Alvarez

## Context

The original architecture (ADR-001) assigned the TUI exclusively to Go/Bubble Tea, with Python providing only the REST API. This created a problem: the Go TUI accessed SQLite directly, duplicating data access patterns that already existed in the Python API. Every new feature required changes in two places — the API models/schemas and the Go TUI's data layer.

Additionally, Bubble Tea's manual layout management (lipgloss calculations, viewport clipping, scroll handling) proved verbose for complex screens compared to CSS-based layout systems.

## Decision

Adopt a **two-tier interactive architecture** with the Python/Textual TUI as the primary interface and the Go/Bubble Tea TUI as a lightweight fallback:

### Primary: Python TUI (Textual)
- Runs in the same process space as the FastAPI REST API.
- Communicates with the backend exclusively through the REST API via HTTP.
- Benefits from Textual's CSS-based layout, reactive widgets, DataTable, and built-in scroll handling.

### Fallback: Go TUI (Bubble Tea)
- Compiled into the `jart-stow` binary with zero runtime dependencies.
- Accesses SQLite directly (no API required).
- Serves as a lightweight alternative when the Python stack is not running.
- Ideal for quick operations: `jart-stow tui` from anywhere.

### Communication Pattern

```
┌─────────────────────────────────────────────┐
│           Python Stack (primary TUI)         │
│                                              │
│  Textual TUI → HTTP → FastAPI → SQLite      │
│  (api/tui/)         (api/app/)               │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│          Go Stack (fallback + daemon)        │
│                                              │
│  CLI → SQLite (direct)                       │
│  Bubble Tea TUI → SQLite (direct)            │
│  Daemon (FSEvents) → SQLite (direct)         │
└─────────────────────────────────────────────┘
```

### Shared State

Both stacks read/write the same SQLite database at `~/.local/share/jart-stow/jart-stow.db` with WAL mode. The Go daemon writes project detections, exclusions, and junk items. The Python API reads/writes the same data. The Textual TUI reads through the API, not directly.

## Consequences

### Positive
- **Single source of truth for data logic.** API models (Pydantic) serve both HTTP responses and the TUI — no duplication.
- **Faster TUI development.** Textual's CSS layout, reactive system, and built-in components (DataTable, Tree, Input, etc.) reduce UI code by ~40% compared to manual lipgloss.
- **API validation in production.** The Textual TUI exercising every endpoint ensures the REST API is always fully functional.
- **Smaller Go binary.** The Bubble Tea TUI can be simplified over time, reducing compile time and binary size.
- **True separation of concerns.** TUI depends on API; API depends on DB; Daemon depends on DB. No circular or bypass paths in normal operation.

### Negative
- **Python TUI requires the API to be running.** Not suitable for air-gapped or minimal environments without the Python stack.
- **Two TUIs to maintain** during the transition period.
- **HTTP latency** for every user action (~1-5ms localhost) vs direct SQLite access (~0.1ms).
- **More complex startup** — user must start the API server before launching the Textual TUI.

## Mitigations

1. The **Go TUI (Bubble Tea)** remains as a compiled-in fallback for environments without Python.
2. The **`jart-stow api`** CLI command will auto-start the API server, so the user workflow becomes `jart-stow api &` followed by `jart-stow tui` (Textual) or just `jart-stow tui` (Bubble Tea).
3. A **health check in the Textual TUI** startup detects if the API is unreachable and falls back gracefully with a helpful message.

## Alternatives Considered

### A: Go TUI + embedded Python for API calls
Rejected. Embedding Python in a Go binary (via cgo or grpc) adds build complexity, cross-compilation issues, and binary size bloat without meaningful benefit over a local HTTP call.

### B: Textual TUI with direct SQLite access (skip API)
Rejected. Bypassing the API would duplicate validation logic and create two code paths for every data operation, defeating the purpose of the hybrid architecture.

### C: Drop the Go TUI entirely
Rejected for now. The Go TUI provides a zero-dependency, instant-on experience that is valuable for power users and CI environments. It may be deprecated in a future release once the Python TUI and API are battle-tested.

### D: Communication via Unix socket instead of HTTP
Rejected for now. HTTP over localhost adds negligible overhead and simplifies debugging (curl, browser, HTTPie). Unix sockets can be adopted later if performance measurements justify it.

## References

- [ADR-001: Hybrid Go + Python Architecture](./adr-001-hybrid-architecture.md)
- [SPECS/ARCHITECTURE.md](../ARCHITECTURE.md)
- [SPECS/API.md](../API.md)
