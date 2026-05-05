"""Async SQLite repository layer for the Jart-Stow REST API.

Every function receives an aiosqlite.Connection as its first parameter and
returns plain dictionaries.  Endpoint handlers then validate/transform
results through Pydantic models before returning JSON.

All SQL uses parameterised queries — **never** f-strings or string
interpolation — to prevent SQL injection.
"""

from datetime import UTC, datetime

import aiosqlite

# ---------------------------------------------------------------------------
# Whitelists for dynamic sort / order parameters
# ---------------------------------------------------------------------------

_PROJECT_SORT_COLUMNS = frozenset({"name", "last_scanned", "created_at", "status"})
_EXCLUSION_SORT_COLUMNS = frozenset({"applied_at", "size_bytes", "pattern_matched"})
_VALID_ORDERS = frozenset({"asc", "desc"})


# ============================================================================
# Helpers
# ============================================================================


def _now_iso() -> str:
    """Return the current UTC timestamp as an ISO-8601 string."""
    return datetime.now(UTC).isoformat()


def _row_to_dict(row: aiosqlite.Row | None) -> dict | None:
    """Convert an aiosqlite.Row into a plain dict, or return None."""
    if row is None:
        return None
    return dict(row)


def _rows_to_dicts(rows: list[aiosqlite.Row]) -> list[dict]:
    """Convert a list of aiosqlite.Row objects into plain dicts."""
    return [dict(r) for r in rows]


# ============================================================================
# Projects
# ============================================================================


async def get_projects(
    db: aiosqlite.Connection,
    status: str | None = None,
    sort_by: str = "name",
    order: str = "asc",
    limit: int = 50,
    offset: int = 0,
) -> dict:
    """Return a paginated, sorted list of projects."""
    if sort_by not in _PROJECT_SORT_COLUMNS:
        sort_by = "name"
    if order not in _VALID_ORDERS:
        order = "asc"

    base_sql = "SELECT * FROM projects"
    count_sql = "SELECT COUNT(*) AS cnt FROM projects"
    params: list = []
    where_clause = ""

    if status is not None:
        where_clause = " WHERE status = ?"
        params = [status]

    count_cursor = await db.execute(count_sql + where_clause, params)
    count_row = await count_cursor.fetchone()
    total = count_row["cnt"] if count_row else 0

    query = (
        f"{base_sql}{where_clause} ORDER BY {sort_by} {order} LIMIT ? OFFSET ?"
    )
    cursor = await db.execute(query, params + [limit, offset])
    rows = await cursor.fetchall()

    return {
        "total": total,
        "limit": limit,
        "offset": offset,
        "projects": _rows_to_dicts(rows),
    }


async def get_project(db: aiosqlite.Connection, project_id: int) -> dict | None:
    """Return a single project with exclusion stats, or None."""
    cursor = await db.execute(
        """
        SELECT p.*,
               COUNT(e.id)       AS exclusion_count,
               COALESCE(SUM(e.size_bytes), 0) AS total_excluded_size_bytes
          FROM projects p
          LEFT JOIN exclusions e
            ON e.project_id = p.id AND e.removed_at IS NULL
         WHERE p.id = ?
         GROUP BY p.id
        """,
        (project_id,),
    )
    row = await cursor.fetchone()
    return _row_to_dict(row)


async def update_project_status(
    db: aiosqlite.Connection, project_id: int, status: str
) -> dict | None:
    """Update a project's status and return the updated row."""
    now = _now_iso()
    await db.execute(
        "UPDATE projects SET status = ?, updated_at = ? WHERE id = ?",
        (status, now, project_id),
    )
    await db.commit()
    cursor = await db.execute("SELECT * FROM projects WHERE id = ?", (project_id,))
    row = await cursor.fetchone()
    return _row_to_dict(row)


# ============================================================================
# Exclusions
# ============================================================================


async def get_exclusions(
    db: aiosqlite.Connection,
    project_id: int | None = None,
    backup_system: str | None = None,
    active_only: bool = True,
    pattern: str | None = None,
    sort_by: str = "applied_at",
    order: str = "desc",
    limit: int = 50,
    offset: int = 0,
) -> dict:
    """Return a filtered, paginated list of exclusions."""
    if sort_by not in _EXCLUSION_SORT_COLUMNS:
        sort_by = "applied_at"
    if order not in _VALID_ORDERS:
        order = "desc"

    conditions: list[str] = []
    params: list = []

    if active_only:
        conditions.append("e.removed_at IS NULL")
    if project_id is not None:
        conditions.append("e.project_id = ?")
        params.append(project_id)
    if backup_system is not None:
        conditions.append("(e.backup_system = ? OR e.backup_system = 'both')")
        params.append(backup_system)
    if pattern is not None:
        conditions.append("e.pattern_matched LIKE ?")
        params.append(f"%{pattern}%")

    where_clause = ""
    if conditions:
        where_clause = " WHERE " + " AND ".join(conditions)

    count_cursor = await db.execute(
        f"SELECT COUNT(*) AS cnt FROM exclusions e{where_clause}", params
    )
    count_row = await count_cursor.fetchone()
    total = count_row["cnt"] if count_row else 0

    # Total size for active exclusions
    size_query = (
        f"SELECT COALESCE(SUM(e.size_bytes), 0) AS total_size_bytes"
        f" FROM exclusions e{where_clause}"
    )
    size_cursor = await db.execute(size_query, params)
    size_row = await size_cursor.fetchone()
    total_size_bytes = size_row["total_size_bytes"] if size_row else 0

    query = (
        f"SELECT e.*, p.name AS project_name FROM exclusions e"
        f" JOIN projects p ON p.id = e.project_id"
        f"{where_clause}"
        f" ORDER BY e.{sort_by} {order} LIMIT ? OFFSET ?"
    )
    cursor = await db.execute(query, params + [limit, offset])
    rows = await cursor.fetchall()

    return {
        "total": total,
        "total_size_bytes": total_size_bytes,
        "exclusions": _rows_to_dicts(rows),
    }


async def create_exclusion(
    db: aiosqlite.Connection,
    project_id: int,
    folder_path: str,
    pattern_matched: str,
    backup_systems: list[str],
) -> dict:
    """Insert a new exclusion and return the created row."""
    # Map the list of backup systems to the single DB column
    if "time_machine" in backup_systems and "carbon_copy_cloner" in backup_systems:
        backup_system = "both"
    elif "time_machine" in backup_systems:
        backup_system = "time_machine"
    elif "carbon_copy_cloner" in backup_systems:
        backup_system = "carbon_copy_cloner"
    else:
        backup_system = "both"

    now = _now_iso()
    cursor = await db.execute(
        """
        INSERT INTO exclusions (project_id, folder_path, pattern_matched,
                                backup_system, size_bytes, applied_at, created_at)
        VALUES (?, ?, ?, ?, 0, ?, ?)
        """,
        (project_id, folder_path, pattern_matched, backup_system, now, now),
    )
    await db.commit()
    new_id = cursor.lastrowid

    cursor = await db.execute(
        """
        SELECT e.*, p.name AS project_name
          FROM exclusions e
          JOIN projects p ON p.id = e.project_id
         WHERE e.id = ?
        """,
        (new_id,),
    )
    row = await cursor.fetchone()
    return dict(row) if row else {}


async def remove_exclusion(db: aiosqlite.Connection, exclusion_id: int) -> dict | None:
    """Mark an exclusion as removed and return the updated row."""
    now = _now_iso()
    await db.execute(
        "UPDATE exclusions SET removed_at = ? WHERE id = ?", (now, exclusion_id)
    )
    await db.commit()
    cursor = await db.execute(
        """
        SELECT e.*, p.name AS project_name
          FROM exclusions e
          JOIN projects p ON p.id = e.project_id
         WHERE e.id = ?
        """,
        (exclusion_id,),
    )
    row = await cursor.fetchone()
    return _row_to_dict(row)


# ============================================================================
# Rules
# ============================================================================


async def get_rules(
    db: aiosqlite.Connection,
    project_id: int | None = None,
    enabled_only: bool = True,
) -> dict:
    """Return all rules, optionally filtered by project and enabled state."""
    conditions: list[str] = []
    params: list = []

    if enabled_only:
        conditions.append("r.enabled = 1")
    if project_id is not None:
        conditions.append("r.project_id = ?")
        params.append(project_id)

    where_clause = ""
    if conditions:
        where_clause = " WHERE " + " AND ".join(conditions)

    count_cursor = await db.execute(
        f"SELECT COUNT(*) AS cnt FROM rules r{where_clause}", params
    )
    count_row = await count_cursor.fetchone()
    total = count_row["cnt"] if count_row else 0

    query = (
        f"SELECT r.*, p.name AS project_name"
        f" FROM rules r"
        f" LEFT JOIN projects p ON p.id = r.project_id"
        f"{where_clause}"
        f" ORDER BY r.priority DESC"
    )
    cursor = await db.execute(query, params)
    rows = await cursor.fetchall()

    return {"total": total, "rules": _rows_to_dicts(rows)}


async def create_rule(
    db: aiosqlite.Connection,
    project_id: int | None,
    pattern: str,
    max_size_bytes: int,
    action: str,
    priority: int,
) -> dict:
    """Insert a new rule and return the created row."""
    now = _now_iso()
    cursor = await db.execute(
        """
        INSERT INTO rules (project_id, pattern, max_size_bytes, action,
                           priority, enabled, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, 1, ?, ?)
        """,
        (project_id, pattern, max_size_bytes, action, priority, now, now),
    )
    await db.commit()
    new_id = cursor.lastrowid

    cursor = await db.execute(
        """
        SELECT r.*, p.name AS project_name
          FROM rules r
          LEFT JOIN projects p ON p.id = r.project_id
         WHERE r.id = ?
        """,
        (new_id,),
    )
    row = await cursor.fetchone()
    return dict(row) if row else {}


async def update_rule(db: aiosqlite.Connection, rule_id: int, **kwargs) -> dict | None:
    """Update fields on an existing rule. Only supplied kwargs are changed."""
    allowed = {"max_size_bytes", "action", "enabled"}
    updates = {k: v for k, v in kwargs.items() if k in allowed and v is not None}
    if not updates:
        cursor = await db.execute(
            """
            SELECT r.*, p.name AS project_name
              FROM rules r
              LEFT JOIN projects p ON p.id = r.project_id
             WHERE r.id = ?
            """,
            (rule_id,),
        )
        row = await cursor.fetchone()
        return _row_to_dict(row)

    set_parts = []
    set_params = []
    for col, val in updates.items():
        set_parts.append(f"{col} = ?")
        set_params.append(1 if col == "enabled" and isinstance(val, bool) else val)

    set_parts.append("updated_at = ?")
    set_params.append(_now_iso())
    set_params.append(rule_id)

    await db.execute(
        f"UPDATE rules SET {', '.join(set_parts)} WHERE id = ?", set_params
    )
    await db.commit()

    cursor = await db.execute(
        """
        SELECT r.*, p.name AS project_name
          FROM rules r
          LEFT JOIN projects p ON p.id = r.project_id
         WHERE r.id = ?
        """,
        (rule_id,),
    )
    row = await cursor.fetchone()
    return _row_to_dict(row)


async def delete_rule(db: aiosqlite.Connection, rule_id: int) -> None:
    """Delete a rule by ID."""
    await db.execute("DELETE FROM rules WHERE id = ?", (rule_id,))
    await db.commit()


# ============================================================================
# Watch Roots
# ============================================================================


async def get_watch_roots(db: aiosqlite.Connection) -> dict:
    """Return all watch roots with their project counts."""
    cursor = await db.execute(
        """
        SELECT wr.*,
               (SELECT COUNT(*) FROM projects p WHERE p.root_path = wr.path) AS projects_count
          FROM watch_roots wr
         ORDER BY wr.created_at DESC
        """
    )
    rows = await cursor.fetchall()
    return {"watch_roots": _rows_to_dicts(rows)}


async def create_watch_root(db: aiosqlite.Connection, path: str) -> dict:
    """Insert a new watch root and return the created row."""
    now = _now_iso()
    cursor = await db.execute(
        """
        INSERT INTO watch_roots (path, enabled, created_at)
        VALUES (?, 1, ?)
        """,
        (path, now),
    )
    await db.commit()
    new_id = cursor.lastrowid

    cursor = await db.execute(
        """
        SELECT wr.*,
               (SELECT COUNT(*) FROM projects p WHERE p.root_path = wr.path) AS projects_count
          FROM watch_roots wr
         WHERE wr.id = ?
        """,
        (new_id,),
    )
    row = await cursor.fetchone()
    return dict(row) if row else {}


async def update_watch_root(
    db: aiosqlite.Connection, root_id: int, enabled: bool
) -> dict | None:
    """Toggle a watch root's enabled flag."""
    await db.execute(
        "UPDATE watch_roots SET enabled = ? WHERE id = ?",
        (1 if enabled else 0, root_id),
    )
    await db.commit()

    cursor = await db.execute(
        """
        SELECT wr.*,
               (SELECT COUNT(*) FROM projects p WHERE p.root_path = wr.path) AS projects_count
          FROM watch_roots wr
         WHERE wr.id = ?
        """,
        (root_id,),
    )
    row = await cursor.fetchone()
    return _row_to_dict(row)


async def delete_watch_root(db: aiosqlite.Connection, root_id: int) -> None:
    """Delete a watch root (does not delete associated project data)."""
    await db.execute("DELETE FROM watch_roots WHERE id = ?", (root_id,))
    await db.commit()


# ============================================================================
# Daemon
# ============================================================================


async def get_daemon_status(db: aiosqlite.Connection) -> dict:
    """Return the current daemon status from the database and system checks."""
    import subprocess

    running = False
    pid: int | None = None
    try:
        result = subprocess.run(
            ["pgrep", "-f", "jart-stow daemon"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        if result.returncode == 0 and result.stdout.strip():
            running = True
            pid = int(result.stdout.strip().split("\n")[0])
    except (FileNotFoundError, subprocess.TimeoutExpired, ValueError):
        pass

    # Get the first enabled watch root
    wr_cursor = await db.execute(
        "SELECT path FROM watch_roots WHERE enabled = 1 LIMIT 1"
    )
    wr_row = await wr_cursor.fetchone()
    watched_root = wr_row["path"] if wr_row else None

    # Count active projects
    prj_cursor = await db.execute(
        "SELECT COUNT(*) AS cnt FROM projects WHERE status = 'active'"
    )
    prj_row = await prj_cursor.fetchone()
    projects_watched = prj_row["cnt"] if prj_row else 0

    # Events today
    evt_cursor = await db.execute(
        """
        SELECT COUNT(*) AS cnt
          FROM daemon_events
         WHERE created_at >= datetime('now', 'start of day')
        """
    )
    evt_row = await evt_cursor.fetchone()
    events_today = evt_row["cnt"] if evt_row else 0

    # Last event timestamp
    last_cursor = await db.execute(
        "SELECT created_at FROM daemon_events ORDER BY created_at DESC LIMIT 1"
    )
    last_row = await last_cursor.fetchone()
    last_event_at = last_row["created_at"] if last_row else None

    # Uptime: look for the most recent daemon_started event
    uptime_seconds: int | None = None
    if running:
        start_cursor = await db.execute(
            """
            SELECT created_at FROM daemon_events
             WHERE event_type = 'daemon_started'
             ORDER BY created_at DESC LIMIT 1
            """
        )
        start_row = await start_cursor.fetchone()
        if start_row:
            try:
                started = datetime.fromisoformat(start_row["created_at"])
                uptime_seconds = int(
                    (datetime.now(UTC) - started).total_seconds()
                )
            except (ValueError, TypeError):
                pass

    return {
        "running": running,
        "pid": pid,
        "watched_root": watched_root,
        "uptime_seconds": uptime_seconds,
        "projects_watched": projects_watched,
        "events_today": events_today,
        "last_event_at": last_event_at,
    }


async def get_daemon_events(
    db: aiosqlite.Connection,
    limit: int = 50,
    event_type: str | None = None,
) -> dict:
    """Return recent daemon events, optionally filtered by type."""
    if event_type is not None:
        cursor = await db.execute(
            """
            SELECT * FROM daemon_events
             WHERE event_type = ?
             ORDER BY created_at DESC
             LIMIT ?
            """,
            (event_type, limit),
        )
    else:
        cursor = await db.execute(
            "SELECT * FROM daemon_events ORDER BY created_at DESC LIMIT ?",
            (limit,),
        )
    rows = await cursor.fetchall()
    return {"events": _rows_to_dicts(rows)}


# ============================================================================
# Junk Management
# ============================================================================


async def get_junk_categories(db: aiosqlite.Connection) -> dict:
    """Return all junk categories."""
    cursor = await db.execute("SELECT * FROM junk_categories ORDER BY id")
    rows = await cursor.fetchall()
    return {"categories": _rows_to_dicts(rows)}


async def update_junk_category(
    db: aiosqlite.Connection, category_id: int, enabled: bool
) -> dict | None:
    """Toggle a junk category's enabled flag."""
    await db.execute(
        "UPDATE junk_categories SET enabled = ? WHERE id = ?",
        (1 if enabled else 0, category_id),
    )
    await db.commit()
    cursor = await db.execute(
        "SELECT * FROM junk_categories WHERE id = ?", (category_id,)
    )
    row = await cursor.fetchone()
    return _row_to_dict(row)


async def get_junk_items(
    db: aiosqlite.Connection,
    category_id: int | None = None,
    verified: int = 0,
    limit: int = 50,
) -> dict:
    """Return junk items filtered by category and verification status."""
    conditions = ["ji.verified_by_user = ?", "ji.cleaned_at IS NULL"]
    params: list = [verified]

    if category_id is not None:
        conditions.append("ji.category_id = ?")
        params.append(category_id)

    where_clause = " WHERE " + " AND ".join(conditions)

    count_cursor = await db.execute(
        f"SELECT COUNT(*) AS cnt FROM junk_items ji{where_clause}", params
    )
    count_row = await count_cursor.fetchone()
    total = count_row["cnt"] if count_row else 0

    size_query = (
        f"SELECT COALESCE(SUM(ji.size_bytes), 0) AS total_size_bytes"
        f" FROM junk_items ji{where_clause}"
    )
    size_cursor = await db.execute(size_query, params)
    size_row = await size_cursor.fetchone()
    total_size_bytes = size_row["total_size_bytes"] if size_row else 0

    query = (
        f"SELECT ji.*, jc.name AS category_name"
        f" FROM junk_items ji"
        f" JOIN junk_categories jc ON jc.id = ji.category_id"
        f"{where_clause}"
        f" ORDER BY ji.size_bytes DESC"
        f" LIMIT ?"
    )
    cursor = await db.execute(query, params + [limit])
    rows = await cursor.fetchall()

    return {
        "total": total,
        "total_size_bytes": total_size_bytes,
        "items": _rows_to_dicts(rows),
    }


async def update_junk_item(
    db: aiosqlite.Connection, item_id: int, verified_by_user: int
) -> dict | None:
    """Update the verification status of a single junk item."""
    await db.execute(
        "UPDATE junk_items SET verified_by_user = ? WHERE id = ?",
        (verified_by_user, item_id),
    )
    await db.commit()
    cursor = await db.execute(
        """
        SELECT ji.*, jc.name AS category_name
          FROM junk_items ji
          JOIN junk_categories jc ON jc.id = ji.category_id
         WHERE ji.id = ?
        """,
        (item_id,),
    )
    row = await cursor.fetchone()
    return _row_to_dict(row)


async def batch_update_junk_items(
    db: aiosqlite.Connection,
    item_ids: list[int],
    verified_by_user: int,
) -> int:
    """Batch-update verification status on multiple junk items."""
    placeholders = ", ".join("?" for _ in item_ids)
    cursor = await db.execute(
        f"UPDATE junk_items SET verified_by_user = ? WHERE id IN ({placeholders})",
        [verified_by_user] + item_ids,
    )
    await db.commit()
    return cursor.rowcount


async def clean_approved_items(db: aiosqlite.Connection) -> dict:
    """Clean all approved junk items and record a cleanup job."""
    # Count items and total size
    count_cursor = await db.execute(
        """
        SELECT COUNT(*) AS cnt,
               COALESCE(SUM(size_bytes), 0) AS total_size
          FROM junk_items
         WHERE verified_by_user = 1
           AND cleaned_at IS NULL
        """
    )
    count_row = await count_cursor.fetchone()
    items_count = count_row["cnt"] if count_row else 0
    total_size_bytes = count_row["total_size"] if count_row else 0

    if items_count == 0:
        return {
            "cleaned": 0,
            "total_size_freed_bytes": 0,
            "cleanup_job_id": None,
        }

    now = _now_iso()

    # Create cleanup job
    job_cursor = await db.execute(
        """
        INSERT INTO cleanup_jobs (items_count, total_size_bytes, started_at, finished_at)
        VALUES (?, ?, ?, ?)
        """,
        (items_count, total_size_bytes, now, now),
    )
    await db.commit()
    job_id = job_cursor.lastrowid

    # Mark items as cleaned
    await db.execute(
        "UPDATE junk_items SET cleaned_at = ? WHERE verified_by_user = 1 AND cleaned_at IS NULL",
        (now,),
    )
    await db.commit()

    return {
        "cleaned": items_count,
        "total_size_freed_bytes": total_size_bytes,
        "cleanup_job_id": job_id,
    }


# ============================================================================
# Reports
# ============================================================================


async def get_report_summary(db: aiosqlite.Connection) -> dict:
    """Return aggregated system statistics for the dashboard."""
    # Project counts
    prj_total_cursor = await db.execute("SELECT COUNT(*) AS cnt FROM projects")
    prj_total = (await prj_total_cursor.fetchone())["cnt"]

    prj_active_cursor = await db.execute(
        "SELECT COUNT(*) AS cnt FROM projects WHERE status = 'active'"
    )
    prj_active = (await prj_active_cursor.fetchone())["cnt"]

    # Active exclusion stats
    excl_cursor = await db.execute(
        """
        SELECT COUNT(*)            AS cnt,
               COALESCE(SUM(size_bytes), 0) AS total_size
          FROM exclusions
         WHERE removed_at IS NULL
        """
    )
    excl_row = await excl_cursor.fetchone()
    exclusions_active = excl_row["cnt"]
    exclusions_total_size_bytes = excl_row["total_size"]

    # Breakdown by pattern
    pat_cursor = await db.execute(
        """
        SELECT pattern_matched,
               SUM(size_bytes) AS total
          FROM exclusions
         WHERE removed_at IS NULL
         GROUP BY pattern_matched
         ORDER BY total DESC
        """
    )
    pat_rows = await pat_cursor.fetchall()
    breakdown_by_pattern = {r["pattern_matched"]: r["total"] for r in pat_rows}

    # Breakdown by backup system
    sys_cursor = await db.execute(
        """
        SELECT backup_system,
               SUM(size_bytes) AS total
          FROM exclusions
         WHERE removed_at IS NULL
         GROUP BY backup_system
        """
    )
    sys_rows = await sys_cursor.fetchall()
    breakdown_by_system = {r["backup_system"]: r["total"] for r in sys_rows}

    # Human-readable size
    size_human = _format_bytes(exclusions_total_size_bytes)

    return {
        "projects_total": prj_total,
        "projects_active": prj_active,
        "exclusions_active": exclusions_active,
        "exclusions_total_size_bytes": exclusions_total_size_bytes,
        "exclusions_total_size_human": size_human,
        "breakdown_by_pattern": breakdown_by_pattern,
        "breakdown_by_system": breakdown_by_system,
    }


async def get_report_history(db: aiosqlite.Connection, days: int = 30) -> dict:
    """Return daily exclusion history for the specified number of days."""
    days_neg = -abs(days)
    days_param = f"{days_neg} days"

    # Added exclusions per day
    added_cursor = await db.execute(
        """
        SELECT date(applied_at)      AS date,
               COUNT(*)             AS count,
               SUM(size_bytes)      AS size
          FROM exclusions
         WHERE applied_at >= datetime('now', ?)
         GROUP BY date(applied_at)
         ORDER BY date
        """,
        (days_param,),
    )
    added_rows = await added_cursor.fetchall()

    # Removed exclusions per day
    removed_cursor = await db.execute(
        """
        SELECT date(removed_at)      AS date,
               COUNT(*)             AS count,
               SUM(size_bytes)      AS size
          FROM exclusions
         WHERE removed_at >= datetime('now', ?)
         GROUP BY date(removed_at)
         ORDER BY date
        """,
        (days_param,),
    )
    removed_rows = await removed_cursor.fetchall()

    # Merge into a date-keyed lookup
    added_map: dict[str, dict] = {}
    for row in added_rows:
        d = row["date"]
        added_map[d] = {
            "exclusions_added": row["count"],
            "size_added_bytes": row["size"] or 0,
        }

    removed_map: dict[str, dict] = {}
    for row in removed_rows:
        d = row["date"]
        removed_map[d] = {
            "exclusions_removed": row["count"],
            "size_removed_bytes": row["size"] or 0,
        }

    # Generate date range via a recursive CTE
    range_cursor = await db.execute(
        """
        WITH RECURSIVE dates(d) AS (
            SELECT date('now', ?)
            UNION ALL
            SELECT date(d, '+1 day') FROM dates WHERE d < date('now')
        )
        SELECT d FROM dates ORDER BY d
        """,
        (days_param,),
    )
    range_rows = await range_cursor.fetchall()

    history: list[dict] = []
    for row in range_rows:
        d = row["d"]
        added = added_map.get(d, {"exclusions_added": 0, "size_added_bytes": 0})
        removed = removed_map.get(d, {"exclusions_removed": 0, "size_removed_bytes": 0})
        history.append(
            {
                "date": d,
                "exclusions_added": added["exclusions_added"],
                "exclusions_removed": removed["exclusions_removed"],
                "size_added_bytes": added["size_added_bytes"],
                "size_removed_bytes": removed["size_removed_bytes"],
            }
        )

    return {"history": history}


# ============================================================================
# Health
# ============================================================================


async def health_check(db: aiosqlite.Connection) -> dict:
    """Perform a health check of the database and daemon."""
    db_connected = False
    try:
        cursor = await db.execute("SELECT 1")
        await cursor.fetchone()
        db_connected = True
    except Exception:
        db_connected = False

    daemon_status = await get_daemon_status(db)

    return {
        "status": "ok" if db_connected else "degraded",
        "daemon_running": daemon_status["running"],
        "database_connected": db_connected,
        "watched_root": daemon_status["watched_root"],
        "uptime_seconds": daemon_status["uptime_seconds"],
    }


# ============================================================================
# Helpers
# ============================================================================


def _format_bytes(num_bytes: int) -> str:
    """Return a human-readable byte size string."""
    if num_bytes == 0:
        return "0 B"
    units = ["B", "KB", "MB", "GB", "TB", "PB"]
    size = float(abs(num_bytes))
    unit_idx = 0
    while size >= 1024 and unit_idx < len(units) - 1:
        size /= 1024
        unit_idx += 1
    return f"{size:.1f} {units[unit_idx]}"
