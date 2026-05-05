"""Report endpoints — summary statistics and history."""

import aiosqlite
from fastapi import APIRouter, Depends, Query

from app.api.v1.schemas import ErrorResponse, ReportHistoryPoint, ReportSummaryResponse
from app.core.database import get_db
from app.infrastructure.sqlite_repository import get_report_history, get_report_summary

router = APIRouter(prefix="/reports", tags=["reports"])


@router.get(
    "/summary",
    response_model=ReportSummaryResponse,
    responses={500: {"model": ErrorResponse}},
)
async def report_summary(
    db: aiosqlite.Connection = Depends(get_db),
):
    """Return aggregated system statistics."""
    result = await get_report_summary(db)
    return ReportSummaryResponse(**result)


@router.get(
    "/history",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def report_history(
    days: int = Query(30, ge=1, le=365),
    db: aiosqlite.Connection = Depends(get_db),
):
    """Return daily exclusion history for the specified number of days."""
    result = await get_report_history(db, days=days)
    result["history"] = [ReportHistoryPoint(**h) for h in result["history"]]
    return result
