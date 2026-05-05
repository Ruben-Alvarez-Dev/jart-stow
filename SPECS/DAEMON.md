# Jart-Stow Daemon Specification

**Version:** 1.0.0  
**Status:** Draft  
**Author:** Ruben Alvarez  
**Date:** 2026-05-05  

---

## 1. Purpose

The Jart-Stow daemon (`jart-stowd`) is a background process that continuously monitors the designated workspace root (`/Code`) for new and modified project directories. Upon detection, it automatically scans for development artifacts and applies backup exclusions without user intervention.

---

## 2. Lifecycle

```
┌──────────┐    install    ┌──────────────┐    start     ┌──────────────┐
│  Binary   │──────────────►│ launchd plist │──────────────►│   Running     │
│ installed │               │   created     │               │   Daemon      │
└──────────┘               └──────────────┘               └──────┬───────┘
                                                                 │
                                                    ┌────────────▼──────────┐
                                                    │   FSEvents Watcher     │
                                                    │   (blocks on events)   │
                                                    └────────────┬──────────┘
                                                                 │
                                                    ┌────────────▼──────────┐
                                                    │   Event: new folder    │
                                                    │   in /Code detected    │
                                                    └────────────┬──────────┘
                                                                 │
                                                    ┌────────────▼──────────┐
                                                    │   ScanService:         │
                                                    │   find -maxdepth 3     │
                                                    └────────────┬──────────┘
                                                                 │
                                                    ┌────────────▼──────────┐
                                                    │   ExcludeService:      │
                                                    │   tmutil + CCC apply   │
                                                    └────────────┬──────────┘
                                                                 │
                                                    ┌────────────▼──────────┐
                                                    │   Log event to DB      │
                                                    │   + daemon.log         │
                                                    └───────────────────────┘
```

---

## 3. Installation (launchd)

The daemon is registered as a macOS `launchd` user agent.

### Plist location

```
~/Library/LaunchAgents/dev.rubenalvarez.jart-stow.plist
```

### Plist content

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.rubenalvarez.jart-stow</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/jart-stow</string>
        <string>daemon</string>
        <string>run</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/Users/ruben/.local/share/jart-stow/daemon.log</string>

    <key>StandardErrorPath</key>
    <string>/Users/ruben/.local/share/jart-stow/daemon.log</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>JART_STOW_DB_PATH</key>
        <string>/Users/ruben/.local/share/jart-stow/jart-stow.db</string>
        <key>JART_STOW_WATCH_ROOT</key>
        <string>/Users/ruben/Code</string>
        <key>JART_STOW_LOG_LEVEL</key>
        <string>info</string>
    </dict>
</dict>
</plist>
```

### CLI commands

```bash
jart-stow daemon install    # Creates plist, loads into launchd
jart-stow daemon uninstall  # Unloads and removes plist
jart-stow daemon start      # launchctl load
jart-stow daemon stop       # launchctl unload
jart-stow daemon restart    # stop + start
jart-stow daemon status     # Checks if loaded + running
jart-stow daemon logs       # Tails daemon.log
```

---

## 4. FSEvents Watcher

The daemon uses macOS FSEvents API via the Go `fsnotify` package to monitor the watched root recursively.

### Watched root

Watched roots are configured in the `watch_roots` database table. The local `/Users/<username>/Code` is added by default on first run, but the user can add any local or external volume paths via CLI, TUI, or API. Each root is independently enabled/disabled.

### Event handling

| FSEvent | Action |
|---|---|
| **New directory created** | Debounce 2s, then scan for artifacts |
| **Directory renamed** | Update project path in DB |
| **Directory deleted** | Mark project as `archived` (do not delete exclusions — they may be wanted) |

### Ignored paths

The watcher ignores these paths to avoid noise:

```
/Code/**/.git/**
/Code/**/node_modules/**
/Code/**/.venv/**
/Code/**/__pycache__/**
/Code/**/.DS_Store
```

---

## 5. Auto-Scan Algorithm

When a new directory is detected under the watched root:

```
function handleNewProject(path):
    // 1. Debounce: wait 2 seconds for filesystem to settle
    sleep(2000ms)

    // 2. Check if already in DB
    project = projectRepo.FindByPath(path)
    if project != nil and project.status == "active":
        return  // Already known

    // 3. Insert or reactivate project
    project = projectRepo.Upsert(path, status="active")

    // 4. Log event
    eventRepo.Log("project_detected", project.id, path)

    // 5. Scan for artifacts
    scanJob = scanJobRepo.Create(project.id, status="running")
    artifacts = scanService.FindArtifacts(path, maxDepth=3)

    // 6. Apply exclusions for each artifact
    for each artifact in artifacts:
        size = du(artifact.path)
        if TM is configured:
            tmutilAdapter.Exclude(artifact.path)
        if CCC is configured:
            cccAdapter.Exclude(artifact.path)
        exclusionRepo.Save(project.id, artifact.path,
                          artifact.pattern, size)

    // 7. Update scan job
    scanJobRepo.MarkCompleted(scanJob.id, len(artifacts), totalSize)

    // 8. Log completion
    eventRepo.Log("scan_completed", project.id, path,
                  {folders_found: len(artifacts), total_size: totalSize})
```

---

## 6. Artifact Patterns (Default)

Hardcoded patterns that the daemon scans for:

```go
var DefaultArtifactPatterns = []string{
    "node_modules",    // JavaScript / Node.js
    ".venv",           // Python virtual environment
    "venv",            // Python virtual environment (alt)
    "__pycache__",     // Python bytecode cache
    ".pytest_cache",   // Python test cache
    "target",          // Rust build output
    "vendor",          // Go vendor directory
    "build",           // Generic build output
    "dist",            // Generic distribution output
    ".next",           // Next.js build
    ".nuxt",           // Nuxt.js build
    ".cache",          // Generic cache
    ".turbo",          // Turborepo cache
    ".eslintcache",    // ESLint cache
    "coverage",        // Test coverage reports
}
```

---

## 7. Graceful Shutdown

The daemon handles SIGTERM and SIGINT:

```
on signal:
    log "daemon_stopped" event
    close FSEvents stream
    close SQLite connection
    exit 0
```

---

## 8. Error Recovery

| Failure | Recovery |
|---|---|
| `tmutil` fails | Log error event, retry on next daemon restart |
| CCC file missing | Create directory and file, retry |
| SQLite locked | Retry with exponential backoff (max 5 attempts) |
| FSEvents stream dies | Reconnect and rescan all active projects |
| Disk full | Log critical error, stop accepting new exclusions |

---

## 9. Configuration

Daemon behavior is configured via environment variables in the launchd plist:

| Variable | Default | Description |
|---|---|---|
| `JART_STOW_DB_PATH` | `~/.local/share/jart-stow/jart-stow.db` | SQLite database path |
| `JART_STOW_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `JART_STOW_SCAN_MAX_DEPTH` | `3` | `find` max depth |
| `JART_STOW_DEBOUNCE_MS` | `2000` | Debounce delay for new folders |
| `JART_STOW_BACKUP_SYSTEMS` | `tm,ccc` | Comma-separated: `tm`, `ccc`, `both` |
| `JART_STOW_JUNK_SCAN_INTERVAL` | `86400` | Interval in seconds between periodic junk scans (default: 24h) |

---

## 10. Junk Scanner (Periodic)

Beyond project exclusions, the daemon runs a **periodic junk scanner** (configurable interval, default 24h) that queries enabled `junk_categories` and populates the `junk_items` table for user review.

### Categories

| Category | Scanner | Source |
|---|---|---|
| Unused Docker images | `docker images -f dangling=true` | Docker CLI |
| Unused Docker containers | `docker ps -a -f status=exited` | Docker CLI |
| Unused Docker volumes | `docker volume ls -f dangling=true` | Docker CLI |
| Docker build cache | `docker builder prune --dry-run` | Docker CLI |
| Unused APFS snapshots | `tmutil listlocalsnapshots /` | tmutil |
| System caches | `du -sh /Library/Caches` | Filesystem |
| User caches | `du -sh ~/Library/Caches` | Filesystem |
| Temporary files | `find /tmp /var/tmp -mtime +7` | Filesystem |
| Xcode derived data | `find ~/Library/Developer/Xcode/DerivedData` | Filesystem |
| Homebrew cache | `brew --cache` | Homebrew |

### Verification Flow

All junk categories default to `verify_required = 1`. Items are **never** cleaned automatically. The flow is:

```
1. Daemon scans and populates junk_items
2. User opens TUI Hygiene screen
3. User reviews items (can filter by category, sort by size)
4. User marks items one-by-one or in batch as "approved"
5. User triggers cleanup → cleanup_jobs recorded
```

---

## 11. Log Format

All daemon logs use structured key=value format:

```
2026-05-05T20:30:00Z level=INFO event=daemon_started watch_root=/Users/ruben/Code
2026-05-05T20:30:15Z level=INFO event=project_detected path=/Users/ruben/Code/new-project
2026-05-05T20:30:17Z level=INFO event=scan_completed project=new-project folders=4 size=1.2GB
2026-05-05T20:30:17Z level=INFO event=exclusion_applied path=/Users/ruben/Code/new-project/node_modules system=tm+ccc
2026-05-05T21:00:00Z level=ERROR event=exclusion_failed path=/Users/ruben/Code/x/.venv error="tmutil: permission denied"
```
