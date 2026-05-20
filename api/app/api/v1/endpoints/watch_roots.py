from pathlib import Path
import aiosqlite
from fastapi import APIRouter, HTTPException, status
from ....db import connection_scope
from ....models import WatchRootCreate, WatchRootOut, WatchRootUpdate
from ..converters import row_to_watch_root
from ..dependencies import ensure_db_exists

router = APIRouter()

@router.get("/watch-roots", response_model=list[WatchRootOut])
async def list_watch_roots() -> list[WatchRootOut]:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT * FROM watch_roots ORDER BY created_at DESC, id DESC")
        rows = await cursor.fetchall()
    return [row_to_watch_root(row) for row in rows]

@router.post("/watch-roots", response_model=WatchRootOut, status_code=status.HTTP_201_CREATED)
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

@router.patch("/watch-roots/{root_id}", response_model=WatchRootOut)
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

@router.delete("/watch-roots/{root_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_watch_root(root_id: int) -> None:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("DELETE FROM watch_roots WHERE id = ?", (root_id,))
        if cursor.rowcount == 0:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="watch root not found")
