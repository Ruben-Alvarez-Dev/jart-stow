# Jart-Stow Data Model Specification

**Version:** 1.0.0  
**Status:** Draft  
**Author:** Ruben Alvarez  
**Date:** 2026-05-05  

---

## 1. Database Engine

**SQLite 3.x** with WAL mode enabled. Single database file at `~/.local/share/jart-stow/jart-stow.db`.

```sql
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA foreign_keys=ON;
```

---

## 2. Entity-Relationship Diagram

```
┌──────────────┐       ┌──────────────────┐
│   projects   │       │    exclusions     │
├──────────────┤       ├──────────────────┤
│ id (PK)      │──┐    │ id (PK)          │
│ path (U)     │  │    │ project_id (FK)  │◄──── projects.id
│ name         │  │    │ folder_path      │
│ root_path    │  │    │ pattern_matched  │
│ last_scanned │  │    │ backup_system    │
│ status       │  │    │ size_bytes       │
│ created_at   │  │    │ applied_at       │
│ updated_at   │  │    │ removed_at       │
└──────────────┘  │    │ created_at       │
                  │    └──────────────────┘
                  │
┌──────────────┐  │    ┌──────────────────┐
│    rules     │  │    │  daemon_events    │
├──────────────┤  │    ├──────────────────┤
│ id (PK)      │  │    │ id (PK)          │
│ project_id   │──┘    │ event_type       │
│ pattern      │       │ project_id (FK)  │
│ max_size     │       │ folder_path      │
│ action       │       │ details          │
│ priority     │       │ created_at       │
│ enabled      │       └──────────────────┘
│ created_at   │
│ updated_at   │       ┌──────────────────┐
└──────────────┘       │   scan_jobs      │
                       ├──────────────────┤
                       │ id (PK)          │
                       │ project_id (FK)  │
                       │ status           │
                       │ folders_found    │
                       │ total_size_bytes │
                       │ started_at       │
                       │ finished_at      │
                       └──────────────────┘
```

---

## 3. Table Definitions

### 3.1 `projects`

Tracks every project directory discovered under the watched root.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| path | TEXT | NOT NULL, UNIQUE | Absolute filesystem path |
| name | TEXT | NOT NULL | Basename of path |
| root_path | TEXT | NOT NULL | Parent watched root (e.g., `/Users/ruben/Code`) |
| last_scanned | TEXT | | ISO 8601 timestamp of last scan |
| status | TEXT | NOT NULL, DEFAULT 'active' | `active`, `ignored`, `archived` |
| created_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |
| updated_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |

```sql
CREATE TABLE projects (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    root_path   TEXT    NOT NULL,
    last_scanned TEXT,
    status      TEXT    NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'ignored', 'archived')),
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_projects_root_path ON projects(root_path);
CREATE INDEX idx_projects_status ON projects(status);
```

---

### 3.2 `exclusions`

Records every exclusion applied to a backup system.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| project_id | INTEGER | FK → projects.id, NOT NULL | Owning project |
| folder_path | TEXT | NOT NULL | Absolute path excluded |
| pattern_matched | TEXT | NOT NULL | Pattern that triggered exclusion (e.g., `node_modules`) |
| backup_system | TEXT | NOT NULL | `time_machine`, `carbon_copy_cloner`, `both` |
| size_bytes | INTEGER | NOT NULL, DEFAULT 0 | Size at exclusion time |
| applied_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | When exclusion was applied |
| removed_at | TEXT | | When exclusion was removed (NULL if active) |
| created_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |

```sql
CREATE TABLE exclusions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id      INTEGER NOT NULL,
    folder_path     TEXT    NOT NULL,
    pattern_matched TEXT    NOT NULL,
    backup_system   TEXT    NOT NULL
        CHECK (backup_system IN ('time_machine', 'carbon_copy_cloner', 'both')),
    size_bytes      INTEGER NOT NULL DEFAULT 0,
    applied_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    removed_at      TEXT,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX idx_exclusions_project_id ON exclusions(project_id);
CREATE INDEX idx_exclusions_active ON exclusions(removed_at) WHERE removed_at IS NULL;
CREATE INDEX idx_exclusions_folder_path ON exclusions(folder_path);
```

---

### 3.3 `rules`

User-defined rules for hygiene control and alerts.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| project_id | INTEGER | FK → projects.id, NULLABLE | NULL = global default rule |
| pattern | TEXT | NOT NULL | Folder pattern to match |
| max_size_bytes | INTEGER | NOT NULL | Threshold in bytes |
| action | TEXT | NOT NULL | `warn`, `alert`, `exclude`, `clean` |
| priority | INTEGER | NOT NULL, DEFAULT 0 | Higher = evaluated first |
| enabled | INTEGER | NOT NULL, DEFAULT 1 | 0 = disabled |
| created_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |
| updated_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |

```sql
CREATE TABLE rules (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id     INTEGER,
    pattern        TEXT    NOT NULL,
    max_size_bytes INTEGER NOT NULL,
    action         TEXT    NOT NULL
        CHECK (action IN ('warn', 'alert', 'exclude', 'clean')),
    priority       INTEGER NOT NULL DEFAULT 0,
    enabled        INTEGER NOT NULL DEFAULT 1,
    created_at     TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT    NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX idx_rules_project_id ON rules(project_id);
CREATE INDEX idx_rules_enabled ON rules(enabled);
```

---

### 3.4 `daemon_events`

Audit log of daemon activity.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| event_type | TEXT | NOT NULL | `project_detected`, `scan_completed`, `exclusion_applied`, `exclusion_removed`, `error` |
| project_id | INTEGER | FK → projects.id, NULLABLE | Related project |
| folder_path | TEXT | | Path involved in event |
| details | TEXT | | JSON payload with extra context |
| created_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |

```sql
CREATE TABLE daemon_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type  TEXT    NOT NULL
        CHECK (event_type IN (
            'project_detected', 'scan_completed', 'exclusion_applied',
            'exclusion_removed', 'error', 'daemon_started', 'daemon_stopped'
        )),
    project_id  INTEGER,
    folder_path TEXT,
    details     TEXT,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE INDEX idx_daemon_events_type ON daemon_events(event_type);
CREATE INDEX idx_daemon_events_created ON daemon_events(created_at);
```

---

### 3.5 `scan_jobs`

Records each scan execution for audit and reporting.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| project_id | INTEGER | FK → projects.id, NOT NULL | Project scanned |
| status | TEXT | NOT NULL | `running`, `completed`, `failed` |
| folders_found | INTEGER | NOT NULL, DEFAULT 0 | Number of junk folders found |
| total_size_bytes | INTEGER | NOT NULL, DEFAULT 0 | Total size of found folders |
| started_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |
| finished_at | TEXT | | Completion timestamp |

```sql
CREATE TABLE scan_jobs (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id       INTEGER NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed')),
    folders_found    INTEGER NOT NULL DEFAULT 0,
    total_size_bytes INTEGER NOT NULL DEFAULT 0,
    started_at       TEXT    NOT NULL DEFAULT (datetime('now')),
    finished_at      TEXT,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX idx_scan_jobs_project ON scan_jobs(project_id);
```

---

---

### 3.6 `watch_roots`

Configurable directories the daemon monitors for project detection.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| path | TEXT | NOT NULL, UNIQUE | Absolute path to watch |
| volume_uuid | TEXT | | Disk UUID for external drive tracking |
| enabled | INTEGER | NOT NULL, DEFAULT 1 | 0 = paused |
| created_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |

```sql
CREATE TABLE watch_roots (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT    NOT NULL UNIQUE,
    volume_uuid TEXT,
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);
```

---

### 3.7 `junk_categories`

Supported types of system junk that Jart-Stow can detect and clean.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| name | TEXT | NOT NULL, UNIQUE | Category name |
| scanner | TEXT | NOT NULL | Scanner module: `docker`, `apfs`, `filesystem`, `cache` |
| verify_required | INTEGER | NOT NULL, DEFAULT 1 | 0 = safe auto-clean, 1 = user must verify each item |
| enabled | INTEGER | NOT NULL, DEFAULT 0 | 0 = opt-in by user before scanning |
| created_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |

```sql
CREATE TABLE junk_categories (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL UNIQUE,
    scanner         TEXT    NOT NULL
        CHECK (scanner IN ('docker', 'apfs', 'filesystem', 'cache', 'logs', 'xcode', 'brew')),
    verify_required INTEGER NOT NULL DEFAULT 1,
    enabled         INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- Default categories (inserted at first run)
INSERT INTO junk_categories (name, scanner, verify_required, enabled) VALUES
    ('unused_docker_images',     'docker',     1, 0),
    ('unused_docker_containers', 'docker',     1, 0),
    ('unused_docker_volumes',    'docker',     1, 0),
    ('docker_build_cache',       'docker',     1, 0),
    ('unused_apfs_snapshots',    'apfs',       1, 0),
    ('system_caches',            'cache',      1, 0),
    ('user_caches',              'cache',      1, 0),
    ('tmp_files',                'filesystem', 1, 0),
    ('xcode_derived_data',       'xcode',      1, 0),
    ('brew_cache',               'brew',       0, 0);
```

---

### 3.8 `junk_items`

Individual junk items discovered by scanners.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| category_id | INTEGER | FK → junk_categories.id, NOT NULL | Category |
| volume_id | INTEGER | NULLABLE | Volume where item was found |
| path | TEXT | NOT NULL | Absolute path |
| description | TEXT | NOT NULL | Human-readable description |
| size_bytes | INTEGER | NOT NULL, DEFAULT 0 | Size of this item |
| last_accessed | TEXT | | When item was last accessed (if known) |
| scan_id | INTEGER | FK → scan_jobs.id | Which scan found this item |
| verified_by_user | INTEGER | NOT NULL, DEFAULT 0 | 0 = pending, 1 = approved, -1 = skipped |
| cleaned_at | TEXT | | When item was cleaned |
| created_at | TEXT | NOT NULL, DEFAULT CURRENT_TIMESTAMP | — |

```sql
CREATE TABLE junk_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id     INTEGER NOT NULL,
    volume_id       INTEGER,
    path            TEXT    NOT NULL,
    description     TEXT    NOT NULL,
    size_bytes      INTEGER NOT NULL DEFAULT 0,
    last_accessed   TEXT,
    scan_id         INTEGER,
    verified_by_user INTEGER NOT NULL DEFAULT 0
        CHECK (verified_by_user IN (-1, 0, 1)),
    cleaned_at      TEXT,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (category_id)  REFERENCES junk_categories(id) ON DELETE CASCADE,
    FOREIGN KEY (scan_id)      REFERENCES scan_jobs(id) ON DELETE SET NULL
);

CREATE INDEX idx_junk_items_category ON junk_items(category_id);
CREATE INDEX idx_junk_items_verified ON junk_items(verified_by_user);
CREATE INDEX idx_junk_items_cleaned ON junk_items(cleaned_at);
```

---

### 3.9 `cleanup_jobs`

Records each cleanup operation.

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | INTEGER | PK, AUTOINCREMENT | Unique identifier |
| category_id | INTEGER | FK → junk_categories.id | Category cleaned |
| items_count | INTEGER | NOT NULL | Number of items cleaned |
| total_size_bytes | INTEGER | NOT NULL | Total space freed |
| started_at | TEXT | NOT NULL | When cleanup started |
| finished_at | TEXT | NOT NULL | When cleanup completed |

```sql
CREATE TABLE cleanup_jobs (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id      INTEGER,
    items_count      INTEGER NOT NULL,
    total_size_bytes INTEGER NOT NULL,
    started_at       TEXT    NOT NULL DEFAULT (datetime('now')),
    finished_at      TEXT    NOT NULL,
    FOREIGN KEY (category_id) REFERENCES junk_categories(id)
);
```

---

## 4. Migration Strategy

Migrations are embedded in the Go binary and applied at startup. The `schema_version` is stored in SQLite as a user-defined pragma:

```sql
PRAGMA user_version = 1;
```

| Version | Description |
|---|---|
| 1 | Initial schema: projects, exclusions, rules, daemon_events, scan_jobs, watch_roots, junk_categories, junk_items, cleanup_jobs |

Migration files live in `internal/adapters/sqlite/migrations/` as numbered SQL files:

```
migrations/
├── 001_initial_schema.sql
```

Go migration runner reads `PRAGMA user_version`, applies pending migrations in order, updates version.

---

## 5. Query Patterns

### Active exclusions for reporting

```sql
SELECT p.name, e.pattern_matched, e.backup_system, e.size_bytes, e.applied_at
FROM exclusions e
JOIN projects p ON p.id = e.project_id
WHERE e.removed_at IS NULL
ORDER BY e.applied_at DESC;
```

### Total space saved by backup system

```sql
SELECT backup_system, SUM(size_bytes) as total_saved
FROM exclusions
WHERE removed_at IS NULL
GROUP BY backup_system;
```

### Projects violating hygiene rules

```sql
SELECT p.name, r.pattern, r.max_size_bytes, r.action
FROM rules r
LEFT JOIN projects p ON p.id = r.project_id
WHERE r.enabled = 1
  AND (r.project_id IS NULL OR p.status = 'active');
```

### Daemon activity in last 24 hours

```sql
SELECT event_type, folder_path, details, created_at
FROM daemon_events
WHERE created_at >= datetime('now', '-1 day')
ORDER BY created_at DESC
LIMIT 100;
```

---

## 6. Data Integrity

- **Foreign keys enforced** via `PRAGMA foreign_keys=ON`.
- **Cascading deletes**: removing a project removes its exclusions, rules, and scan jobs.
- **Check constraints** on enum-like columns (`status`, `backup_system`, `event_type`, `action`).
- **WAL mode** prevents corruption from concurrent access.
- **No mock data, no seed data, no fixtures.** The database starts empty and is populated exclusively by real system operations.
