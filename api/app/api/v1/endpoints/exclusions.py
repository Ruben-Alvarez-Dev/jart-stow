"""Exclusion endpoints — manage backup exclusions."""


import aiosqlite
from fastapi import APIRouter, Depends, HTTPException, Query

from app.api.v1.schemas import ErrorResponse, ExclusionCreate, ExclusionResponse
from app.core.database import get_db
from app.infrastructure.sqlite_repository import (
    create_exclusion,
    get_exclusions,
    get_project,
    remove_exclusion,
)

router = APIRouter(prefix="/exclusions", tags=["exclusions"])


@router.get(
    "",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def list_exclusions(
    project_id: int | None = Query(None, ge=1),
    backup_system: str | None = Query(
        None, pattern=r"^(time_machine|carbon_copy_cloner)$"
    ),
    active_only: bool = Query(True),
    pattern: str | None = Query(None),
    sort_by: str = Query("applied_at", pattern=r"^(applied_at|size_bytes|pattern_matched)$"),
    order: str = Query("desc", pattern=r"^(asc|desc)$"),
    limit: int = Query(50, ge=1, le=500),
    offset: int = Query(0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
):
    """List exclusions with filtering, sorting, and pagination."""
    result = await get_exclusions(
        db,
        project_id=project_id,
        backup_system=backup_system,
        active_only=active_only,
        pattern=pattern,
        sort_by=sort_by,
        order=order,
        limit=limit,
        offset=offset,
    )
    result["exclusions"] = [ExclusionResponse(**e) for e in result["exclusions"]]
    return result


@router.post(
    "",
    status_code=201,
    response_model=ExclusionResponse,
    responses={
        400: {"model": ErrorResponse},
        404: {"model": ErrorResponse},
    },
)
async def add_exclusion(
    body: ExclusionCreate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Manually apply a new exclusion to the configured backup systems."""
    project = await get_project(db, body.project_id)
    if project is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "PROJECT_NOT_FOUND",
                    "message": f"Project with ID {body.project_id} not found",
                    "details": None,
                }
            },
        )
    exclusion = await create_exclusion(
        db,
        body.project_id,
        body.folder_path,
        body.pattern_matched,
        body.backup_systems,
    )
    return ExclusionResponse(**exclusion)


@router.delete(
    "/{exclusion_id}",
    response_model=dict,
    responses={404: {"model": ErrorResponse}},
)
async def delete_exclusion(
    exclusion_id: int,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Remove an exclusion, restoring the folder to backup sets."""
    result = await remove_exclusion(db, exclusion_id)
    if result is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "EXCLUSION_NOT_FOUND",
                    "message": f"Exclusion with ID {exclusion_id} not found",
                    "details": None,
                }
            },
        )
    return result
