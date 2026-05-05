"""Daemon endpoints — status, start, stop, and event log."""

import subprocess

import aiosqlite
from fastapi import APIRouter, Depends, HTTPException, Query

from app.api.v1.schemas import DaemonEventResponse, DaemonStatusResponse, ErrorResponse
from app.core.database import get_db
from app.infrastructure.sqlite_repository import get_daemon_events, get_daemon_status

router = APIRouter(prefix="/daemon", tags=["daemon"])


def _daemon_label() -> str:
    """Return the launchd service label for the Jart-Stow daemon."""
    import os

    return os.environ.get("JART_STOW_LABEL", "com.jart-stow.daemon")


@router.get(
    "/status",
    response_model=DaemonStatusResponse,
    responses={500: {"model": ErrorResponse}},
)
async def daemon_status(
    db: aiosqlite.Connection = Depends(get_db),
):
    """Return the current daemon status."""
    status = await get_daemon_status(db)
    return DaemonStatusResponse(**status)


@router.post(
    "/start",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def daemon_start(
    db: aiosqlite.Connection = Depends(get_db),
):
    """Start the Jart-Stow daemon via launchctl."""
    try:
        result = subprocess.run(
            ["launchctl", "load", f"/Library/LaunchDaemons/{_daemon_label()}.plist"],
            capture_output=True,
            text=True,
            timeout=15,
        )
        if result.returncode != 0:
            raise HTTPException(
                status_code=500,
                detail={
                    "error": {
                        "code": "DAEMON_ERROR",
                        "message": f"Failed to start daemon: {result.stderr.strip()}",
                        "details": None,
                    }
                },
            )
        # Log the event
        await db.execute(
            """
            INSERT INTO daemon_events (event_type, details, created_at)
            VALUES ('daemon_started', 'Started via API', datetime('now'))
            """
        )
        await db.commit()
        return {"message": "Daemon started", "pid": None}
    except HTTPException:
        raise
    except Exception as exc:
        raise HTTPException(
            status_code=500,
            detail={
                "error": {
                    "code": "DAEMON_ERROR",
                    "message": str(exc),
                    "details": None,
                }
            },
        ) from exc


@router.post(
    "/stop",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def daemon_stop(
    db: aiosqlite.Connection = Depends(get_db),
):
    """Stop the Jart-Stow daemon via launchctl."""
    try:
        result = subprocess.run(
            ["launchctl", "unload", f"/Library/LaunchDaemons/{_daemon_label()}.plist"],
            capture_output=True,
            text=True,
            timeout=15,
        )
        if result.returncode != 0:
            raise HTTPException(
                status_code=500,
                detail={
                    "error": {
                        "code": "DAEMON_ERROR",
                        "message": f"Failed to stop daemon: {result.stderr.strip()}",
                        "details": None,
                    }
                },
            )
        # Log the event
        await db.execute(
            """
            INSERT INTO daemon_events (event_type, details, created_at)
            VALUES ('daemon_stopped', 'Stopped via API', datetime('now'))
            """
        )
        await db.commit()
        return {"message": "Daemon stopped"}
    except HTTPException:
        raise
    except Exception as exc:
        raise HTTPException(
            status_code=500,
            detail={
                "error": {
                    "code": "DAEMON_ERROR",
                    "message": str(exc),
                    "details": None,
                }
            },
        ) from exc


@router.get(
    "/events",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def daemon_events(
    limit: int = Query(50, ge=1, le=500),
    event_type: str | None = Query(
        None,
        pattern=r"^(project_detected|scan_completed|exclusion_applied|exclusion_removed|error|daemon_started|daemon_stopped)$",
    ),
    db: aiosqlite.Connection = Depends(get_db),
):
    """Return recent daemon events, optionally filtered by type."""
    result = await get_daemon_events(db, limit=limit, event_type=event_type)
    result["events"] = [DaemonEventResponse(**e) for e in result["events"]]
    return result
