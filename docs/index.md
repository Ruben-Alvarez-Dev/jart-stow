# Jart-Stow Documentation

Welcome to the Jart-Stow documentation.

## What is Jart-Stow?

Jart-Stow is a macOS-native development hygiene and backup exclusion manager. It performs two distinct but complementary functions:

1. **Backup Exclusion** — monitors configurable workspace roots via a background daemon, automatically scans projects for development artifacts, and applies exclusions to Time Machine and Carbon Copy Cloner.

2. **System Hygiene** — detects and presents system junk for review (unused Docker resources, stale APFS snapshots, caches, temporary files) and enables granular, user-verified cleanup.

## Architecture at a Glance

Jart-Stow uses a **hybrid Go + Python** architecture:

- **Go** powers the daemon (FSEvents, launchd), the TUI (Bubble Tea), and the CLI (Cobra).
- **Python (FastAPI 3.0)** powers the REST API with automatic OpenAPI 3.0 documentation.
- Both runtimes share a single **SQLite database** in WAL mode.

[Read the full architecture overview →](architecture/overview.md)

## Quick Links

- [Installation Guide](getting-started/installation.md)
- [API Reference](api/reference.md)
- [TUI User Guide](tui/user-guide.md)
- [Daemon Lifecycle](daemon/lifecycle.md)
- [Contributing Guide](development/contributing.md)
- [Architecture Decisions](architecture/decisions/)
