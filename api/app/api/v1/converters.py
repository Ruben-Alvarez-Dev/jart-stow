from typing import Any
import aiosqlite
from ...db import parse_timestamp
from ...models import (
    ExclusionOut,
    JunkCategoryOut,
    JunkItemOut,
    ProjectOut,
    RuleOut,
    WatchRootOut,
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
