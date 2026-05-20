from collections import defaultdict
from datetime import date
from fastapi import APIRouter, Query
from ....db import connection_scope, parse_timestamp
from ....models import (
    ReportHistoryEntryOut,
    ReportPatternBreakdown,
    ReportSummaryOut,
    ReportSystemBreakdown,
)
from ..dependencies import ensure_db_exists

router = APIRouter()

@router.get("/reports/summary", response_model=ReportSummaryOut)
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

@router.get("/reports/history", response_model=list[ReportHistoryEntryOut])
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
