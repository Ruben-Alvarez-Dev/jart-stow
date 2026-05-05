# Jart-Stow Architecture Specification

**Version:** 1.0.0  
**Status:** Draft  
**Author:** Ruben Alvarez  
**Date:** 2026-05-05  

---

## 1. Executive Summary

Jart-Stow is a macOS-native development hygiene and backup exclusion manager. It performs two distinct but complementary functions:

1. **Backup Exclusion:** monitors configurable workspace roots (local and external drives) via a background daemon, automatically scans projects for development artifacts (dependencies, caches, build outputs), and applies exclusions to Time Machine and Carbon Copy Cloner.

2. **System Hygiene:** detects and presents system junk for review — unused Docker resources, stale APFS snapshots, system caches, temporary files — and allows granular, user-verified cleanup across all connected volumes.

A Bubble Tea TUI and a FastAPI 3.0 REST API expose the system state for interactive management and external consumption.

---

## 2. Architectural Decision: Hybrid Go + Python

### ADR-001: Hybrid Architecture

**Context:** The system requires a low-level macOS daemon (FSEvents, launchd, tmutil) and a professional TUI (Bubble Tea), both of which are best served by Go. Simultaneously, it requires a standards-compliant REST API with OpenAPI 3.0 documentation, best served by FastAPI 3.0 in Python.

**Decision:** Adopt a **Hybrid Architecture** with two runtimes communicating through a shared SQLite database.

**Rationale:**
- Go provides zero-dependency binaries, native FSEvents support, and the best TUI ecosystem (Charmbracelet).
- Python/FastAPI provides the richest OpenAPI ecosystem, automatic interactive docs, and rapid API development.
- SQLite as the shared state layer avoids IPC complexity (no gRPC, no message queue) while remaining filesystem-portable.

**Consequences:**
- Two runtimes to build and package.
- SQLite must be accessed with WAL mode to allow concurrent reads from both processes.
- Deployment must ensure both binaries are installed together.

---

## 3. Hexagonal Architecture

The system follows **Hexagonal (Ports & Adapters)** architecture, with domain logic isolated from infrastructure concerns.

```
┌──────────────────────────────────────────────────────────┐
│                     PRIMARY ACTORS                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │   CLI    │  │   TUI    │  │ REST API │  │  Daemon   │ │
│  │ (Cobra)  │  │(BubbleT) │  │(FastAPI) │  │(FSEvents) │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘ │
│       │             │             │             │        │
│       └─────────────┴──────┬──────┴─────────────┘        │
│                            │                              │
│                   ┌────────▼────────┐                     │
│                   │   APPLICATION   │                     │
│                   │    SERVICES     │                     │
│                   │  (Use Cases)    │                     │
│                   └────────┬────────┘                     │
│                            │                              │
│                   ┌────────▼────────┐                     │
│                   │     DOMAIN      │                     │
│                   │  (Entities,     │                     │
│                   │   Value Objs)   │                     │
│                   └────────┬────────┘                     │
│                            │                              │
│                   ┌────────▼────────┐                     │
│                   │      PORTS      │                     │
│                   │  (Interfaces)   │                     │
│                   └────────┬────────┘                     │
│                            │                              │
│       ┌────────────────────┼────────────────────┐        │
│       │                    │                    │        │
│  ┌────▼─────┐  ┌───────────▼──┐  ┌─────────────▼───┐    │
│  │  SQLite  │  │   tmutil     │  │      CCC         │    │
│  │ Adapter  │  │   Adapter    │  │    Adapter       │    │
│  └──────────┘  └──────────────┘  └─────────────────┘    │
│                                                          │
│                   SECONDARY ADAPTERS                     │
└──────────────────────────────────────────────────────────┘
```

### Layer Dependency Rule

Dependencies flow **inward only**:

```
Primary Actors → Services → Domain ← Ports ← Adapters
```

- **Domain** has zero external dependencies.
- **Ports** are Go interfaces defined in `internal/ports/`.
- **Adapters** implement ports and are injected at startup.
- **Services** depend on ports, never on adapters directly.

---

## 4. SOLID Principles Applied

| Principle | Application |
|---|---|
| **S**ingle Responsibility | Each service handles one use case (ScanService, ExcludeService, MonitorService). Each adapter wraps one external system. |
| **O**pen/Closed | New backup systems (e.g., rsync, rclone) are added as new adapters implementing the `BackupProvider` port — no domain changes. |
| **L**iskov Substitution | Every adapter must satisfy its port interface contract fully. `TMUtilAdapter` and `CCCAdapter` are interchangeable behind `BackupProvider`. |
| **I**nterface Segregation | Ports are small: `ProjectRepository`, `ExclusionRepository`, `RuleRepository`, `BackupProvider`, `FileSystemWatcher`. |
| **D**ependency Inversion | Services depend on port interfaces, not concrete adapters. Wiring happens in `cmd/jart-stow/main.go` via constructor injection. |

---

## 5. Data Flow

### Daemon Auto-Exclusion Flow

```
FSEvents → Daemon detects new folder in /Code
         → ScanService.FindArtifacts(path)
         → ExclusionRepository.Save(exclusion)
         → BackupProvider.Exclude(path)   [TM + CCC]
         → DaemonEventRepository.Log(event)
```

### TUI Status Query Flow

```
User presses "s" in TUI
         → StatusScreen loads
         → ProjectRepository.FindAll()
         → ExclusionRepository.FindByProject(projectID)
         → TUI renders tree + stats
```

### REST API Query Flow

```
GET /api/v1/projects
         → FastAPI router
         → ProjectService (Python)
         → SQLiteRepository (Python, same DB)
         → JSON response (OpenAPI schema)
```

---

## 6. Directory Structure

```
jart-stow/
├── cmd/
│   └── jart-stow/
│       └── main.go                  # Binary entry point
├── internal/
│   ├── domain/                      # Entities, value objects
│   │   ├── project.go
│   │   ├── exclusion.go
│   │   ├── rule.go
│   │   └── event.go
│   ├── ports/                       # Interfaces
│   │   ├── repository.go            # Project, Exclusion, Rule repos
│   │   ├── backup.go                # BackupProvider interface
│   │   └── watcher.go               # FileSystemWatcher interface
│   ├── services/                    # Use cases
│   │   ├── scanner.go               # ScanService (development artifacts)
│   │   ├── excluder.go              # ExcludeService (TM + CCC)
│   │   ├── junk_scanner.go          # JunkScanner (Docker, APFS, caches, temp)
│   │   ├── monitor.go               # MonitorService (daemon logic)
│   │   ├── auditor.go               # AuditService
│   │   └── reporter.go              # ReportService
│   ├── cli/                         # Cobra commands
│   │   ├── root.go
│   │   ├── daemon.go
│   │   ├── scan.go
│   │   ├── status.go
│   │   ├── inspect.go
│   │   ├── audit.go
│   │   ├── rule.go
│   │   └── report.go
│   ├── tui/                         # Bubble Tea interface
│   │   ├── app.go                   # Main TUI model
│   │   ├── screens/
│   │   │   ├── dashboard.go
│   │   │   ├── scanner.go
│   │   │   ├── exclusions.go
│   │   │   ├── rules.go
│   │   │   ├── audit.go
│   │   │   └── report.go
│   │   ├── components/
│   │   │   ├── tree.go
│   │   │   ├── table.go
│   │   │   ├── gauge.go
│   │   │   ├── log.go
│   │   │   └── spinner.go
│   │   └── theme/
│   │       └── style.go             # Lipgloss theme
│   └── adapters/                    # Infrastructure
│       ├── sqlite/
│       │   ├── project_repo.go
│       │   ├── exclusion_repo.go
│       │   ├── rule_repo.go
│       │   ├── event_repo.go
│       │   ├── junk_repo.go
│       │   └── migrations.go
│       ├── tmutil/
│       │   └── adapter.go
│       ├── ccc/
│       │   └── adapter.go
│       ├── docker/
│       │   └── scanner.go
│       ├── apfs/
│       │   └── snapshot_scanner.go
│       └── fsevents/
│           └── watcher.go
├── api/                             # FastAPI Python application
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py                  # FastAPI app factory
│   │   ├── api/
│   │   │   ├── __init__.py
│   │   │   └── v1/
│   │   │       ├── __init__.py
│   │   │       ├── router.py
│   │   │       ├── endpoints/
│   │   │       │   ├── projects.py
│   │   │       │   ├── exclusions.py
│   │   │       │   ├── daemon.py
│   │   │       │   ├── rules.py
│   │   │       │   └── reports.py
│   │   │       └── schemas/
│   │   │           ├── project.py
│   │   │           ├── exclusion.py
│   │   │           ├── daemon.py
│   │   │           ├── rule.py
│   │   │           └── report.py
│   │   ├── core/
│   │   │   ├── __init__.py
│   │   │   ├── config.py
│   │   │   └── database.py
│   │   ├── domain/
│   │   │   ├── __init__.py
│   │   │   ├── models.py
│   │   │   └── services.py
│   │   └── infrastructure/
│   │       ├── __init__.py
│   │       └── sqlite_repository.py
│   ├── tests/
│   │   ├── conftest.py
│   │   ├── test_projects.py
│   │   ├── test_exclusions.py
│   │   ├── test_daemon.py
│   │   └── test_rules.py
│   ├── pyproject.toml
│   └── requirements.txt
├── docs/                            # MkDocs + Material
│   ├── index.md
│   ├── architecture/
│   │   ├── overview.md
│   │   └── decisions/
│   │       └── adr-001-hybrid-architecture.md
│   ├── api/
│   │   └── reference.md
│   ├── tui/
│   │   └── user-guide.md
│   ├── daemon/
│   │   └── lifecycle.md
│   └── development/
│       ├── contributing.md
│       └── setup.md
├── SPECS/                           # Design specifications (this directory)
│   ├── ARCHITECTURE.md
│   ├── API.md
│   ├── TUI.md
│   ├── DAEMON.md
│   ├── DATA.md
│   └── WORKFLOW.md
├── scripts/
│   ├── install.sh
│   └── build.sh
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   └── release.yml
│   └── ISSUE_TEMPLATE/
│       └── bug_report.md
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
├── mkdocs.yml
├── README.md
├── CHANGELOG.md
├── CONTRIBUTING.md
└── LICENSE
```

---

## 7. Runtime Communication

Both runtimes (Go binary + Python API) share the **same SQLite database file** at `~/.local/share/jart-stow/jart-stow.db`.

```
~/.local/share/jart-stow/
├── jart-stow.db          # SQLite (WAL mode)
├── jart-stow.db-wal      # Write-Ahead Log
├── jart-stow.db-shm      # Shared Memory
└── daemon.log            # Daemon activity log
```

### WAL Mode Requirement

```sql
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
```

WAL mode allows the Go daemon and Python API to read concurrently while one writes.

---

## 8. Design Patterns

| Pattern | Where | Purpose |
|---|---|---|
| **Repository** | `ports/repository.go` → `adapters/sqlite/` | Abstract data access |
| **Strategy** | `ports/backup.go` → `TMUtilAdapter`, `CCCAdapter` | Pluggable backup systems |
| **Observer** | `ports/watcher.go` → `FSEventsWatcher` | File system events |
| **Factory** | `services/` constructor functions | Service instantiation with DI |
| **Command** | `cli/` Cobra commands | CLI encapsulation |
| **Model-Update-View** | `tui/` Bubble Tea | TUI state management |
| **Facade** | `services/monitor.go` | Simplified daemon interface |

---

## 9. Technology Stack

| Component | Technology | Version |
|---|---|---|
| Daemon + CLI + TUI | Go | 1.22+ |
| TUI Framework | Bubble Tea + Bubbles + Lipgloss | Latest |
| CLI Framework | Cobra + Viper | Latest |
| REST API | Python + FastAPI | 3.0+ |
| API Docs | OpenAPI 3.0 (auto-generated) | — |
| Database | SQLite | 3.x |
| File Watching | FSEvents (macOS native) | — |
| Service Manager | launchd (macOS native) | — |
| Backup Integration | tmutil, CCC Exclusions.txt | — |
| Documentation | MkDocs + Material | Latest |
| CI/CD | GitHub Actions | — |
