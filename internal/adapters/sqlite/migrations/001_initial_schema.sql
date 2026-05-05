-- Migration 001: Initial Schema
-- Creates all core tables, indexes, and default data for Jart-Stow.
-- PRAGMA settings (journal_mode, busy_timeout, foreign_keys) are applied
-- in connection.go before migrations run, since journal_mode=WAL cannot
-- be changed inside a transaction.

-- ============================================================================
-- Projects
-- ============================================================================
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

-- ============================================================================
-- Exclusions
-- ============================================================================
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

-- ============================================================================
-- Rules
-- ============================================================================
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

-- ============================================================================
-- Daemon Events
-- ============================================================================
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

-- ============================================================================
-- Scan Jobs
-- ============================================================================
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

-- ============================================================================
-- Watch Roots
-- ============================================================================
CREATE TABLE watch_roots (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT    NOT NULL UNIQUE,
    volume_uuid TEXT,
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ============================================================================
-- Junk Categories
-- ============================================================================
CREATE TABLE junk_categories (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL UNIQUE,
    scanner         TEXT    NOT NULL
        CHECK (scanner IN ('docker', 'apfs', 'filesystem', 'cache', 'logs', 'xcode', 'brew')),
    verify_required INTEGER NOT NULL DEFAULT 1,
    enabled         INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- Seed default junk categories
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

-- ============================================================================
-- Junk Items
-- ============================================================================
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
    FOREIGN KEY (category_id) REFERENCES junk_categories(id) ON DELETE CASCADE,
    FOREIGN KEY (scan_id)     REFERENCES scan_jobs(id) ON DELETE SET NULL
);

CREATE INDEX idx_junk_items_category ON junk_items(category_id);
CREATE INDEX idx_junk_items_verified ON junk_items(verified_by_user);
CREATE INDEX idx_junk_items_cleaned ON junk_items(cleaned_at);

-- ============================================================================
-- Cleanup Jobs
-- ============================================================================
CREATE TABLE cleanup_jobs (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id      INTEGER,
    items_count      INTEGER NOT NULL,
    total_size_bytes INTEGER NOT NULL,
    started_at       TEXT    NOT NULL DEFAULT (datetime('now')),
    finished_at      TEXT    NOT NULL,
    FOREIGN KEY (category_id) REFERENCES junk_categories(id)
);

-- ============================================================================
-- Schema version tracking
-- ============================================================================
PRAGMA user_version = 1;
