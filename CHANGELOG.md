# Changelog

All notable changes to Jart-Stow will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Go daemon with FSEvents watcher, multi-root watching, and launchd integration
- Automatic project detection with dev artifact scanning (node_modules, .venv, target, etc.)
- Time Machine exclusion via `tmutil` adapter
- Carbon Copy Cloner exclusion via `Exclusions.txt` adapter
- SQLite persistence with 9 repositories, WAL mode, and migration runner
- 31 integration tests for SQLite adapter (83% coverage)
- System hygiene: junk scanners for Docker (images, containers, volumes, build cache)
- System hygiene: APFS snapshot scanner via `tmutil listlocalsnapshots`
- System hygiene: filesystem scanners for temp files, caches, Xcode DerivedData, Brew
- ExcludeService coordinating multi-backup exclusion with DB tracking
- JunkScanService orchestrating all junk scanners by category
- MonitorService: daemon event loop with debounced FSEvents and periodic junk scans
- QuickExcludeService: standalone scan→exclude flow (no DB required)
- Bubble Tea TUI with 7 screens: Dashboard, Scanner, Exclusions, Rules, Audit, Hygiene, Report
- Viewport-safe Screen component system with Card, ActionBar, and NavBar primitives
- Navigation stack with push/pop and mouse support in main menu
- Action bridges wiring real services to TUI screens (ScanEngine, JunkScanRunner, ExclusionManager, QuickExclude)
- CLI commands: scan, exclude (on/off/list), daemon (install/start/stop/uninstall/run), tui
- CLI commands: inspect, audit, rule (list/add/remove), report (all wired to real services)
- Pager support for long CLI output (pipes through `less -R -F -X`)
- FastAPI REST API with 22 endpoints across 8 endpoint groups
- Async SQLite repository (933 lines) with parameterized queries
- Pydantic v2 domain models mirroring Go entities
- Python Textual TUI as primary interactive interface (6 screens)
- Async HTTP client wrapping all REST API endpoints
- Textual CSS theme with card layout, status indicators, and split panels
- Confirm modal, nav bar, and status card reusable widgets
- AuditService: verifies exclusion consistency against actual backup system state
- ReportService: generates aggregate summaries with breakdowns by pattern and system
- ADR-001 through ADR-009 documenting all architectural decisions
- CI pipeline with GitHub Actions (lint, test, build)
- Release workflow with goreleaser configuration
- Makefile with build, test, lint, docs, api, and tui targets

### Changed
- Architecture evolved from Go Bubble Tea TUI to single Python Textual TUI
- All interactive UI is now in Python/Textual, communicating via REST API
- Go binary is CLI + Daemon only (no TUI, no Bubble Tea, no lipgloss)
- Go module dependencies reduced (bubbletea, lipgloss, bubbles removed)
- All CLI commands updated to use CLIDependencies struct for constructor injection

### Removed
- Go Bubble Tea TUI entirely (`internal/tui/`, `internal/cli/tui.go`)
- Bubble Tea, Lipgloss, and Bubbles dependencies from go.mod

### Fixed
- `__pycache__` files removed from version control
- .gitignore updated for `api/tui/**/__pycache__/`
- Git author configured to Ruben-Alvarez-Dev <ruben.alvarez.dev@gmail.com>

## [1.0.0] — Not yet released

See [SPECS/](SPECS/) for the complete design documentation and [ROADMAP.md](ROADMAP.md) for the release plan.
