# Jart-Stow

**macOS development hygiene & backup exclusion manager.**

Jart-Stow performs two distinct but complementary functions:

1. **Backup Exclusion** — monitors configurable workspace roots (local and external drives) via a background daemon, automatically scans projects for development artifacts, and applies exclusions to Time Machine and Carbon Copy Cloner.

2. **System Hygiene** — detects and presents system junk for review (unused Docker resources, stale APFS snapshots, caches, temporary files) and enables granular, user-verified cleanup across all connected volumes.

---

## Quick Start

### Build and run

```bash
git clone https://github.com/Ruben-Alvarez-Dev/jart-stow.git
cd jart-stow

# Build the Go binary
go build -o jart-stow ./cmd/jart-stow/

# Try it
./jart-stow --help
./jart-stow status
./jart-stow scan

# Run the daemon
./jart-stow daemon run
```

### API (Python/FastAPI)

```bash
cd api
pip install -e .
uvicorn app.main:app --port 8420
```

---

## Documentation

Full documentation at [ruben-alvarez-dev.github.io/jart-stow](https://ruben-alvarez-dev.github.io/jart-stow)

---

## Architecture

- **Go binary** (`jart-stow`): Daemon, TUI (Bubble Tea), CLI (Cobra)
- **Python API** (FastAPI 3.0): REST API on port 8420
- **Shared state**: Single SQLite database in WAL mode at `~/.local/share/jart-stow/jart-stow.db`
- **Hexagonal architecture**: Domain → Ports → Adapters

---

## License

MIT © Ruben Alvarez
