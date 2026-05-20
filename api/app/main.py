from __future__ import annotations

from collections import defaultdict
from datetime import date
from pathlib import Path
from typing import Any

import aiosqlite
from fastapi import FastAPI, HTTPException, Query, status

from .config import api_version
from .db import connection_scope, get_db_info, parse_timestamp
from .models import (
    ExclusionOut,
    HealthResponse,
    JunkCategoryOut,
    JunkItemOut,
    ProjectOut,
    ReportHistoryEntryOut,
    ReportPatternBreakdown,
    ReportSummaryOut,
    ReportSystemBreakdown,
    RuleCreate,
    RuleOut,
    RuleUpdate,
    WatchRootCreate,
    WatchRootOut,
    WatchRootUpdate,
)

app = FastAPI(
    title="Jart-Stow API",
    version=api_version(),
    description="SQLite-backed API for Jart-Stow projects, exclusions, rules, and hygiene data.",
)


def row_to_project(row: aiosqlite.Row | dict[str, Any]) -> ProjectOut:
    return ProjectOut(
        id=row["id"],
        path=row["path"],
        name=row["name"],
        root_path=row["root_path"],
        last_scanned=parse_timestamp(row["last_scanned"]),
        status=row["status"],
        created_at=parse_timestamp(row["created_at"]),
        updated_at=parse_timestamp(row["updated_at"]),
    )


def row_to_exclusion(row: aiosqlite.Row | dict[str, Any]) -> ExclusionOut:
    return ExclusionOut(
        id=row["id"],
        project_id=row["project_id"],
        folder_path=row["folder_path"],
        pattern_matched=row["pattern_matched"],
        backup_system=row["backup_system"],
        size_bytes=row["size_bytes"],
        applied_at=parse_timestamp(row["applied_at"]),
        removed_at=parse_timestamp(row["removed_at"]),
        created_at=parse_timestamp(row["created_at"]),
    )


def row_to_rule(row: aiosqlite.Row | dict[str, Any]) -> RuleOut:
    return RuleOut(
        id=row["id"],
        project_id=row["project_id"],
        pattern=row["pattern"],
        max_size_bytes=row["max_size_bytes"],
        action=row["action"],
        priority=row["priority"],
        enabled=bool(row["enabled"]),
        created_at=parse_timestamp(row["created_at"]),
        updated_at=parse_timestamp(row["updated_at"]),
    )


def row_to_watch_root(row: aiosqlite.Row | dict[str, Any]) -> WatchRootOut:
    return WatchRootOut(
        id=row["id"],
        path=row["path"],
        volume_uuid=row["volume_uuid"],
        enabled=bool(row["enabled"]),
        created_at=parse_timestamp(row["created_at"]),
    )


def row_to_junk_category(row: aiosqlite.Row | dict[str, Any]) -> JunkCategoryOut:
    return JunkCategoryOut(
        id=row["id"],
        name=row["name"],
        scanner=row["scanner"],
        verify_required=bool(row["verify_required"]),
        enabled=bool(row["enabled"]),
        created_at=parse_timestamp(row["created_at"]),
    )


def row_to_junk_item(row: aiosqlite.Row | dict[str, Any]) -> JunkItemOut:
    return JunkItemOut(
        id=row["id"],
        category_id=row["category_id"],
        volume_id=row["volume_id"],
        path=row["path"],
        description=row["description"],
        size_bytes=row["size_bytes"],
        last_accessed=parse_timestamp(row["last_accessed"]),
        scan_id=row["scan_id"],
        verified_by_user=row["verified_by_user"],
        cleaned_at=parse_timestamp(row["cleaned_at"]),
        created_at=parse_timestamp(row["created_at"]),
    )


def ensure_db_exists() -> None:
    db_info = get_db_info()
    if not db_info.exists:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"database not found at {db_info.path}",
        )


@app.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    db_info = get_db_info()
    return HealthResponse(
        status="ok" if db_info.exists else "degraded",
        version=api_version(),
        db_path=db_info.path,
        db_exists=db_info.exists,
    )


@app.get("/api/v1/projects", response_model=list[ProjectOut])
async def list_projects(
    status_filter: str | None = Query(default=None, alias="status")
) -> list[ProjectOut]:
    ensure_db_exists()
    query = "SELECT * FROM projects"
    params: list[object] = []
    if status_filter:
        query += " WHERE status = ?"
        params.append(status_filter)
    query += " ORDER BY updated_at DESC, id DESC"
    async with connection_scope() as conn:
        cursor = await conn.execute(query, params)
        rows = await cursor.fetchall()
    return [row_to_project(row) for row in rows]


@app.get("/api/v1/exclusions", response_model=list[ExclusionOut])
async def list_exclusions(active_only: bool = True) -> list[ExclusionOut]:
    ensure_db_exists()
    query = "SELECT * FROM exclusions"
    if active_only:
        query += " WHERE removed_at IS NULL"
    query += " ORDER BY applied_at DESC, id DESC"
    async with connection_scope() as conn:
        cursor = await conn.execute(query)
        rows = await cursor.fetchall()
    return [row_to_exclusion(row) for row in rows]


@app.get("/api/v1/rules", response_model=list[RuleOut])
async def list_rules() -> list[RuleOut]:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT * FROM rules ORDER BY priority DESC, id DESC")
        rows = await cursor.fetchall()
    return [row_to_rule(row) for row in rows]


@app.post("/api/v1/rules", response_model=RuleOut, status_code=status.HTTP_201_CREATED)
async def create_rule(payload: RuleCreate) -> RuleOut:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute(
            """
            INSERT INTO rules (project_id, pattern, max_size_bytes, action, priority, enabled)
            VALUES (?, ?, ?, ?, ?, ?)
            """,
            (
                payload.project_id,
                payload.pattern,
                payload.max_size_bytes,
                payload.action,
                payload.priority,
                int(payload.enabled),
            ),
        )
        cursor = await conn.execute("SELECT * FROM rules WHERE id = ?", (cursor.lastrowid,))
        row = await cursor.fetchone()
    return row_to_rule(row)


@app.patch("/api/v1/rules/{rule_id}", response_model=RuleOut)
async def update_rule(rule_id: int, payload: RuleUpdate) -> RuleOut:
    ensure_db_exists()
    updates = payload.model_dump(exclude_unset=True)
    if not updates:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="no fields provided")

    field_map = {
        "pattern": "pattern",
        "max_size_bytes": "max_size_bytes",
        "action": "action",
        "priority": "priority",
        "enabled": "enabled",
    }
    assignments: list[str] = []
    values: list[object] = []
    for key, value in updates.items():
        assignments.append(f"{field_map[key]} = ?")
        values.append(int(value) if key == "enabled" else value)
    assignments.append("updated_at = datetime('now')")
    values.append(rule_id)

    async with connection_scope() as conn:
        cursor = await conn.execute(
            f"UPDATE rules SET {', '.join(assignments)} WHERE id = ?", values
        )
        if cursor.rowcount == 0:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="rule not found")
        cursor = await conn.execute("SELECT * FROM rules WHERE id = ?", (rule_id,))
        row = await cursor.fetchone()
    return row_to_rule(row)


@app.delete("/api/v1/rules/{rule_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_rule(rule_id: int) -> None:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("DELETE FROM rules WHERE id = ?", (rule_id,))
        if cursor.rowcount == 0:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="rule not found")


@app.get("/api/v1/watch-roots", response_model=list[WatchRootOut])
async def list_watch_roots() -> list[WatchRootOut]:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT * FROM watch_roots ORDER BY created_at DESC, id DESC")
        rows = await cursor.fetchall()
    return [row_to_watch_root(row) for row in rows]


@app.post("/api/v1/watch-roots", response_model=WatchRootOut, status_code=status.HTTP_201_CREATED)
async def create_watch_root(payload: WatchRootCreate) -> WatchRootOut:
    ensure_db_exists()
    normalized_path = str(Path(payload.path).expanduser())
    async with connection_scope() as conn:
        try:
            cursor = await conn.execute(
                "INSERT INTO watch_roots (path, volume_uuid, enabled) VALUES (?, ?, ?)",
                (normalized_path, payload.volume_uuid, int(payload.enabled)),
            )
        except aiosqlite.IntegrityError as exc:
            raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail=str(exc)) from exc
        cursor = await conn.execute("SELECT * FROM watch_roots WHERE id = ?", (cursor.lastrowid,))
        row = await cursor.fetchone()
    return row_to_watch_root(row)


@app.patch("/api/v1/watch-roots/{root_id}", response_model=WatchRootOut)
async def update_watch_root(root_id: int, payload: WatchRootUpdate) -> WatchRootOut:
    ensure_db_exists()
    updates = payload.model_dump(exclude_unset=True)
    if not updates:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="no fields provided")

    assignments: list[str] = []
    values: list[object] = []
    if "enabled" in updates:
        assignments.append("enabled = ?")
        values.append(int(updates["enabled"]))
    if "volume_uuid" in updates:
        assignments.append("volume_uuid = ?")
        values.append(updates["volume_uuid"])
    values.append(root_id)

    async with connection_scope() as conn:
        cursor = await conn.execute(
            f"UPDATE watch_roots SET {', '.join(assignments)} WHERE id = ?",
            values,
        )
        if cursor.rowcount == 0:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="watch root not found")
        cursor = await conn.execute("SELECT * FROM watch_roots WHERE id = ?", (root_id,))
        row = await cursor.fetchone()
    return row_to_watch_root(row)


@app.delete("/api/v1/watch-roots/{root_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_watch_root(root_id: int) -> None:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("DELETE FROM watch_roots WHERE id = ?", (root_id,))
        if cursor.rowcount == 0:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="watch root not found")


@app.get("/api/v1/junk/categories", response_model=list[JunkCategoryOut])
async def list_junk_categories() -> list[JunkCategoryOut]:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT * FROM junk_categories ORDER BY name ASC")
        rows = await cursor.fetchall()
    return [row_to_junk_category(row) for row in rows]


@app.get("/api/v1/junk/items", response_model=list[JunkItemOut])
async def list_junk_items(
    pending_only: bool = False,
    approved_only: bool = False,
    cleaned_only: bool = False,
) -> list[JunkItemOut]:
    ensure_db_exists()
    query = "SELECT * FROM junk_items"
    clauses: list[str] = []
    params: list[object] = []
    if pending_only:
        clauses.append("verified_by_user = 0")
    if approved_only:
        clauses.append("verified_by_user = 1")
    if cleaned_only:
        clauses.append("cleaned_at IS NOT NULL")
    if clauses:
        query += " WHERE " + " AND ".join(clauses)
    query += " ORDER BY created_at DESC, id DESC"
    async with connection_scope() as conn:
        cursor = await conn.execute(query, params)
        rows = await cursor.fetchall()
    return [row_to_junk_item(row) for row in rows]


@app.get("/api/v1/reports/summary", response_model=ReportSummaryOut)
async def report_summary() -> ReportSummaryOut:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT COUNT(*) FROM projects")
        projects_total = (await cursor.fetchone())[0]

        cursor = await conn.execute("SELECT COUNT(*) FROM projects WHERE status = 'active'")
        projects_active = (await cursor.fetchone())[0]

        cursor = await conn.execute(
            "SELECT pattern_matched, backup_system, size_bytes FROM exclusions WHERE removed_at IS NULL"
        )
        exclusion_rows = await cursor.fetchall()

        cursor = await conn.execute(
            "SELECT COUNT(*) FROM daemon_events WHERE date(created_at) = date('now', 'localtime')"
        )
        events_today = (await cursor.fetchone())[0]

        cursor = await conn.execute("SELECT COUNT(*) FROM junk_items WHERE verified_by_user = 0")
        junk_items_pending = (await cursor.fetchone())[0]

    pattern_map: dict[str, dict[str, int]] = defaultdict(lambda: {"count": 0, "size": 0})
    system_map: dict[str, dict[str, int]] = defaultdict(lambda: {"count": 0, "size": 0})
    total_size = 0
    for row in exclusion_rows:
        total_size += row["size_bytes"]
        pattern_map[row["pattern_matched"]]["count"] += 1
        pattern_map[row["pattern_matched"]]["size"] += row["size_bytes"]
        system_map[row["backup_system"]]["count"] += 1
        system_map[row["backup_system"]]["size"] += row["size_bytes"]

    pattern_breakdowns = sorted(
        [
            ReportPatternBreakdown(pattern=pattern, count=data["count"], size_bytes=data["size"])
            for pattern, data in pattern_map.items()
        ],
        key=lambda item: item.size_bytes,
        reverse=True,
    )
    system_breakdowns = sorted(
        [
            ReportSystemBreakdown(system=system, count=data["count"], size_bytes=data["size"])
            for system, data in system_map.items()
        ],
        key=lambda item: item.size_bytes,
        reverse=True,
    )

    return ReportSummaryOut(
        projects_total=projects_total,
        projects_active=projects_active,
        exclusions_active=len(exclusion_rows),
        exclusions_total_size=total_size,
        pattern_breakdowns=pattern_breakdowns,
        system_breakdowns=system_breakdowns,
        events_today=events_today,
        junk_items_pending=junk_items_pending,
    )


@app.get("/api/v1/reports/history", response_model=list[ReportHistoryEntryOut])
async def report_history(
    days: int = Query(default=30, ge=1, le=365)
) -> list[ReportHistoryEntryOut]:
    ensure_db_exists()
    cutoff = date.today().toordinal() - days
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT created_at, removed_at, size_bytes FROM exclusions")
        rows = await cursor.fetchall()

    by_date: dict[str, ReportHistoryEntryOut] = {}
    for row in rows:
        created_at = parse_timestamp(row["created_at"])
        if created_at and created_at.date().toordinal() >= cutoff:
            key = created_at.date().isoformat()
            current = by_date.get(
                key,
                ReportHistoryEntryOut(date=key, added_count=0, removed_count=0, added_size_bytes=0),
            )
            current.added_count += 1
            current.added_size_bytes += row["size_bytes"]
            by_date[key] = current

        removed_at = parse_timestamp(row["removed_at"])
        if removed_at and removed_at.date().toordinal() >= cutoff:
            key = removed_at.date().isoformat()
            current = by_date.get(
                key,
                ReportHistoryEntryOut(date=key, added_count=0, removed_count=0, added_size_bytes=0),
            )
            current.removed_count += 1
            by_date[key] = current

    return sorted(by_date.values(), key=lambda item: item.date, reverse=True)
