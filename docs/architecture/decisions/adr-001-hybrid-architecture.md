# ADR-001: Hybrid Go + Python Architecture

**Status:** Accepted  
**Date:** 2026-05-05  
**Deciders:** Ruben Alvarez  

## Context

Jart-Stow must deliver three distinct interfaces: a low-level macOS daemon (FSEvents, launchd, tmutil), a professional terminal UI (Bubble Tea), and a standards-compliant REST API (FastAPI + OpenAPI 3.0). No single language excels at all three equally.

## Decision

Adopt a **Hybrid Architecture** with two runtimes sharing a single SQLite database:

- **Go** for the daemon, TUI (Bubble Tea), and CLI (Cobra).
- **Python (FastAPI 3.0)** for the REST API.

Communication between runtimes occurs exclusively through the shared SQLite database in WAL mode. No IPC, no gRPC, no message queue.

## Consequences

### Positive
- Each runtime uses the best tool for its job: Go for systems programming and TUI, Python for rapid API development with OpenAPI docs.
- SQLite WAL mode allows concurrent reads while one process writes.
- Simple deployment: two artifacts, one shared file.

### Negative
- Two build pipelines, two test suites, two dependency trees.
- Developers must be proficient in both Go and Python.
- Schema changes must be coordinated across both codebases.

## Alternatives Considered

### A: All-in-Go
- **Rejected** because Go REST API frameworks (Gin, Chi, Echo) lack FastAPI's automatic OpenAPI generation and interactive docs. Writing OpenAPI specs manually for a dozen endpoints is error-prone and contradicts the "docs as code" principle.

### B: All-in-Python
- **Rejected** because Python lacks native FSEvents support without C extensions, and Python TUI frameworks (Textual, Rich) do not match Bubble Tea's maturity, ecosystem, or performance for this use case.

### C: Go daemon + Python API communicating via gRPC
- **Rejected** as overengineered for a single-machine tool. Adding a gRPC layer between two local processes adds complexity (protobuf compilation, service definitions, TLS) without meaningful benefit over SQLite.

### D: Separate microservices
- **Rejected** as excessive for a macOS developer tool with a single user. The overhead of container orchestration, service discovery, and network configuration is unjustified.

## References

- [SPECS/ARCHITECTURE.md](../ARCHITECTURE.md)
- [SPECS/DATA.md](../DATA.md) — SQLite WAL mode details
