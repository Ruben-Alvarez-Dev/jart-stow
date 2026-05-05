"""Project endpoints — CRUD for tracked projects."""


import aiosqlite
from fastapi import APIRouter, Depends, HTTPException, Query

from app.api.v1.schemas import ErrorResponse, ProjectDetail, ProjectResponse, ProjectUpdate
from app.core.database import get_db
from app.infrastructure.sqlite_repository import get_project, get_projects, update_project_status

router = APIRouter(prefix="/projects", tags=["projects"])


@router.get(
    "",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def list_projects(
    status: str | None = Query(None, pattern=r"^(active|ignored|archived)$"),
    sort_by: str = Query("name", pattern=r"^(name|last_scanned|created_at|status)$"),
    order: str = Query("asc", pattern=r"^(asc|desc)$"),
    limit: int = Query(50, ge=1, le=500),
    offset: int = Query(0, ge=0),
    db: aiosqlite.Connection = Depends(get_db),
):
    """List all tracked projects with optional filtering and pagination."""
    result = await get_projects(
        db,
        status=status,
        sort_by=sort_by,
        order=order,
        limit=limit,
        offset=offset,
    )
    result["projects"] = [ProjectResponse(**p) for p in result["projects"]]
    return result


@router.get(
    "/{project_id}",
    response_model=ProjectDetail,
    responses={404: {"model": ErrorResponse}},
)
async def read_project(
    project_id: int,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Get a single project by ID, including exclusion statistics."""
    project = await get_project(db, project_id)
    if project is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "PROJECT_NOT_FOUND",
                    "message": f"Project with ID {project_id} not found",
                    "details": None,
                }
            },
        )
    return ProjectDetail(**project)


@router.patch(
    "/{project_id}",
    response_model=ProjectResponse,
    responses={
        404: {"model": ErrorResponse},
        400: {"model": ErrorResponse},
    },
)
async def patch_project(
    project_id: int,
    body: ProjectUpdate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Update a project's status."""
    existing = await get_project(db, project_id)
    if existing is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "PROJECT_NOT_FOUND",
                    "message": f"Project with ID {project_id} not found",
                    "details": None,
                }
            },
        )
    updated = await update_project_status(db, project_id, body.status)
    return ProjectResponse(**updated)
