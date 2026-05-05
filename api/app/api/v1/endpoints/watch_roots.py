"""Watch root endpoints — manage watched directories."""

import subprocess

import aiosqlite
from fastapi import APIRouter, Depends, HTTPException
from fastapi.responses import Response

from app.api.v1.schemas import (
    ErrorResponse,
    WatchRootCreate,
    WatchRootResponse,
    WatchRootUpdate,
)
from app.core.database import get_db
from app.infrastructure.sqlite_repository import (
    create_watch_root,
    delete_watch_root,
    get_watch_roots,
    update_watch_root,
)

router = APIRouter(prefix="/watch-roots", tags=["watch_roots"])


def _resolve_volume_uuid(path: str) -> str | None:
    """Resolve the volume UUID for a given path using diskutil."""
    try:
        result = subprocess.run(
            ["diskutil", "info", "-plist", path],
            capture_output=True,
            text=True,
            timeout=10,
        )
        if result.returncode == 0 and "VolumeUUID" in result.stdout:
            import plistlib

            info = plistlib.loads(result.stdout.encode())
            return info.get("VolumeUUID")
    except Exception:
        pass
    return None


@router.get(
    "",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def list_watch_roots(
    db: aiosqlite.Connection = Depends(get_db),
):
    """List all configured watch roots."""
    result = await get_watch_roots(db)
    result["watch_roots"] = [WatchRootResponse(**w) for w in result["watch_roots"]]
    return result


@router.post(
    "",
    status_code=201,
    response_model=WatchRootResponse,
    responses={400: {"model": ErrorResponse}},
)
async def add_watch_root(
    body: WatchRootCreate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Add a new watch root directory."""
    volume_uuid = _resolve_volume_uuid(body.path)
    result = await create_watch_root(db, body.path)
    # Attach the resolved volume UUID
    result["volume_uuid"] = volume_uuid
    # Update the DB with the resolved UUID
    if volume_uuid:
        await db.execute(
            "UPDATE watch_roots SET volume_uuid = ? WHERE id = ?",
            (volume_uuid, result["id"]),
        )
        await db.commit()
    return WatchRootResponse(**result)


@router.patch(
    "/{root_id}",
    response_model=WatchRootResponse,
    responses={404: {"model": ErrorResponse}},
)
async def patch_watch_root(
    root_id: int,
    body: WatchRootUpdate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Enable or disable a watch root."""
    result = await update_watch_root(db, root_id, body.enabled)
    if result is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "WATCH_ROOT_NOT_FOUND",
                    "message": f"Watch root with ID {root_id} not found",
                    "details": None,
                }
            },
        )
    return WatchRootResponse(**result)


@router.delete(
    "/{root_id}",
    status_code=204,
    responses={404: {"model": ErrorResponse}},
)
async def remove_watch_root(
    root_id: int,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Remove a watch root (does not delete project data)."""
    existing = await get_watch_roots(db)
    if not any(w["id"] == root_id for w in existing["watch_roots"]):
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "WATCH_ROOT_NOT_FOUND",
                    "message": f"Watch root with ID {root_id} not found",
                    "details": None,
                }
            },
        )
    await delete_watch_root(db, root_id)
    return Response(status_code=204)
