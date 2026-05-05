# Jart-Stow Roadmap

## v1.0.0 — MVP

- [ ] FSEvents daemon with multi-root watching
- [ ] Automatic project detection and dev artifact scanning
- [ ] Time Machine exclusion via `tmutil`
- [ ] Carbon Copy Cloner exclusion via `Exclusions.txt`
- [ ] Bubble Tea TUI with 7 screens
- [ ] FastAPI 3.0 REST API with 20+ endpoints
- [ ] SQLite persistence (WAL mode)
- [ ] System hygiene: junk detection (Docker, APFS, caches, temp)
- [ ] Granular user verification for all junk cleanup
- [ ] launchd integration for auto-start
- [ ] Homebrew installation
- [ ] MkDocs documentation published to GitHub Pages

## v1.1.0 — Hygiene Enhancements

- [ ] Daemon: live junk notifications (system notification when junk exceeds threshold)
- [ ] Daemon: configurable junk scan interval per category
- [ ] TUI: junk trend graphs (junk growth over time)
- [ ] Hazel.app rules export for users who want GUI monitoring

## v1.2.0 — Google Drive Integration

- [ ] Rclone wrapper for selective Google Drive backup
- [ ] Duplicate detection against Drive contents
- [ ] Version-aware upload (no overwriting without confirmation)
- [ ] Drive backup status in TUI dashboard

## v2.0.0 — Cross-Platform

- [ ] Linux support (FSEvents → inotify, launchd → systemd)
- [ ] Windows support (FSEvents → ReadDirectoryChangesW, launchd → Task Scheduler)
- [ ] Backup system adapters for non-macOS platforms

## Future Ideas

- [ ] Web dashboard (React, served by FastAPI)
- [ ] MCP server integration for AI-assisted hygiene decisions
- [ ] CI/CD integration (jart-stow check as a pipeline step)
