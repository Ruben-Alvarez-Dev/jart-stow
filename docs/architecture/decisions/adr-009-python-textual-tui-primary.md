# ADR-009: Python Textual TUI as the Sole Interactive Interface

**Status:** Accepted (updated 2026-05-13)
**Deciders:** Ruben Alvarez

## Context

The original architecture (ADR-001) assigned the TUI to Go/Bubble Tea. This was later reconsidered: Bubble Tea's manual layout management (lipgloss calculations, viewport clipping, scroll handling) proved verbose for complex screens compared to CSS-based systems, and the Go TUI duplicated data access patterns already present in the Python API.

## Decision

Adopt a **single TUI architecture** with all interactive user interfaces in Python/Textual:

### TUI: Python (Textual)
- Runs in the same process space as the FastAPI REST API.
- Communicates with the backend exclusively through the REST API via HTTP.
- Benefits from Textual's CSS-based layout, reactive widgets, DataTable, and built-in scroll handling.

### Go Stack (CLI + Daemon only)
- CLI commands (`scan`, `exclude`, `inspect`, `audit`, `rule`, `report`) access SQLite directly.
- Daemon (FSEvents watcher, backup exclusion, junk scanning) accesses SQLite directly.
- No TUI code in Go. No Bubble Tea. No lipgloss.

### Communication Pattern

```
┌─────────────────────────────────────────────┐
│           Python Stack (TUI + API)           │
│                                              │
│  Textual TUI → HTTP → FastAPI → SQLite      │
│  (api/tui/)         (api/app/)               │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│          Go Stack (CLI + Daemon)            │
│                                              │
│  CLI (scan, exclude, audit, report...)       │
│    → SQLite (direct)                         │
│  Daemon (FSEvents, tmutil, CCC)             │
│    → SQLite (direct)                         │
└─────────────────────────────────────────────┘
```

### Shared State

Both stacks read/write the same SQLite database at `~/.local/share/jart-stow/jart-stow.db` with WAL mode. The Go daemon writes project detections, exclusions, and junk items. The Python API reads/writes the same data. The Textual TUI reads through the API, not directly.

## Consequences

### Positive
- **Single TUI codebase.** No duplication between Go and Python TUI implementations.
- **Single source of truth for data logic.** API models (Pydantic) serve both HTTP responses and the TUI.
- **Faster TUI development.** Textual's CSS layout, reactive system, and built-in components.
- **API validation in production.** The Textual TUI exercising every endpoint ensures the REST API is always fully functional.
- **Smaller Go binary.** No Bubble Tea, lipgloss, or bubbles dependencies in the Go module.
- **True separation of concerns.** TUI depends on API; API depends on DB; Daemon depends on DB.

### Negative
- **Textual TUI requires the API to be running.** User must start the API server before launching the TUI.
- **HTTP latency** for every user action (~1-5ms localhost) vs direct SQLite access (~0.1ms).
- **More complex startup** — user must start the API server before launching the Textual TUI.

## Mitigations

1. The **`jart-stow api`** CLI command will auto-start the API server.
2. A **health check in the Textual TUI** startup detects if the API is unreachable and shows a helpful message with instructions.
3. All **read-only operations** (scan, inspect, report) remain available via the Go CLI without the API running.

## Alternatives Considered

### A: Go Bubble Tea TUI (original approach)
Rejected. Duplicated data access logic, verbose layout code, and added 12+ dependencies to the Go module (bubbletea, lipgloss, bubbles, etc.) for a single binary that duplicated functionality already provided by the Python TUI.

### B: Go TUI + embedded Python for API calls
Rejected. Embedding Python in a Go binary adds build complexity and cross-compilation issues without meaningful benefit over a local HTTP call.

### C: Textual TUI with direct SQLite access (skip API)
Rejected. Bypassing the API would duplicate validation logic and create two code paths for every data operation, defeating the purpose of the hybrid architecture.

### D: Two TUIs (Go fallback + Python primary)
Rejected after implementation. Maintaining two TUI codebases for the same screens doubled maintenance burden without compensating benefits. The Go TUI was removed in favor of a single Python/Textual TUI.

### E: Communication via Unix socket instead of HTTP
Rejected for now. HTTP over localhost adds negligible overhead and simplifies debugging (curl, browser, HTTPie). Unix sockets can be adopted later if performance measurements justify it.

## References

- [ADR-001: Hybrid Go + Python Architecture](./adr-001-hybrid-architecture.md)
- [SPECS/ARCHITECTURE.md](../ARCHITECTURE.md)
- [SPECS/API.md](../API.md)
