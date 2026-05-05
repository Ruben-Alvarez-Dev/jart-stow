"""Health check endpoint."""

import aiosqlite
from fastapi import APIRouter, Depends

from app.api.v1.schemas import ErrorResponse, HealthResponse
from app.core.database import get_db
from app.infrastructure.sqlite_repository import health_check

router = APIRouter(prefix="/health", tags=["health"])


@router.get(
    "",
    response_model=HealthResponse,
    responses={500: {"model": ErrorResponse}},
)
async def check_health(
    db: aiosqlite.Connection = Depends(get_db),
):
    """Return system health status."""
    result = await health_check(db)
    return HealthResponse(**result)
