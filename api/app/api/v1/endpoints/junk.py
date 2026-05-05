"""Junk management endpoints — categories, scanning, items, and cleanup."""


import aiosqlite
from fastapi import APIRouter, Depends, HTTPException, Query

from app.api.v1.schemas import (
    ErrorResponse,
    JunkBatchUpdate,
    JunkCategoryResponse,
    JunkCategoryUpdate,
    JunkCleanResponse,
    JunkItemResponse,
    JunkItemUpdate,
    JunkScanRequest,
    JunkScanResponse,
)
from app.core.database import get_db
from app.infrastructure.sqlite_repository import (
    batch_update_junk_items,
    clean_approved_items,
    get_junk_categories,
    get_junk_items,
    update_junk_category,
    update_junk_item,
)

router = APIRouter(prefix="/junk", tags=["junk"])


@router.get(
    "/categories",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def list_categories(
    db: aiosqlite.Connection = Depends(get_db),
):
    """List all junk categories."""
    result = await get_junk_categories(db)
    result["categories"] = [
        JunkCategoryResponse(**c) for c in result["categories"]
    ]
    return result


@router.patch(
    "/categories/{category_id}",
    response_model=JunkCategoryResponse,
    responses={404: {"model": ErrorResponse}},
)
async def patch_category(
    category_id: int,
    body: JunkCategoryUpdate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Enable or disable a junk category for scanning."""
    result = await update_junk_category(db, category_id, body.enabled)
    if result is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "CATEGORY_NOT_FOUND",
                    "message": f"Junk category with ID {category_id} not found",
                    "details": None,
                }
            },
        )
    return JunkCategoryResponse(**result)


@router.post(
    "/scan",
    status_code=202,
    response_model=JunkScanResponse,
    responses={400: {"model": ErrorResponse}},
)
async def trigger_scan(
    body: JunkScanRequest,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Trigger a junk scan for the specified categories."""
    # Validate that all requested categories exist
    categories = await get_junk_categories(db)
    cat_ids = {c["id"] for c in categories["categories"]}
    missing = set(body.category_ids) - cat_ids
    if missing:
        raise HTTPException(
            status_code=400,
            detail={
                "error": {
                    "code": "VALIDATION_ERROR",
                    "message": f"Unknown category IDs: {sorted(missing)}",
                    "details": None,
                }
            },
        )

    # Create a placeholder scan job. In production the Go daemon picks this
    # up and runs the actual scanners. Here we record intent.
    cursor = await db.execute(
        """
        INSERT INTO scan_jobs (project_id, status, started_at)
        VALUES (0, 'running', datetime('now'))
        """
    )
    await db.commit()
    scan_job_id = cursor.lastrowid

    return JunkScanResponse(
        message="Junk scan started",
        scan_job_id=scan_job_id,
    )


@router.get(
    "/items",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def list_items(
    category_id: int | None = Query(None, ge=1),
    verified: int = Query(0, ge=-1, le=1),
    limit: int = Query(50, ge=1, le=500),
    db: aiosqlite.Connection = Depends(get_db),
):
    """List junk items pending review, with optional filtering."""
    result = await get_junk_items(
        db,
        category_id=category_id,
        verified=verified,
        limit=limit,
    )
    result["items"] = [JunkItemResponse(**i) for i in result["items"]]
    return result


@router.patch(
    "/items/{item_id}",
    response_model=dict,
    responses={404: {"model": ErrorResponse}},
)
async def patch_item(
    item_id: int,
    body: JunkItemUpdate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Approve or skip a single junk item."""
    # Check existence
    check_cursor = await db.execute(
        "SELECT id FROM junk_items WHERE id = ?", (item_id,)
    )
    check_row = await check_cursor.fetchone()
    if check_row is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "ITEM_NOT_FOUND",
                    "message": f"Junk item with ID {item_id} not found",
                    "details": None,
                }
            },
        )
    await update_junk_item(db, item_id, body.verified_by_user)
    return {"id": item_id, "verified_by_user": body.verified_by_user}


@router.post(
    "/items/batch",
    response_model=dict,
    responses={400: {"model": ErrorResponse}},
)
async def batch_update_items(
    body: JunkBatchUpdate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Batch approve or skip multiple junk items."""
    updated = await batch_update_junk_items(
        db, body.item_ids, body.verified_by_user
    )
    return {"updated": updated}


@router.post(
    "/clean",
    response_model=JunkCleanResponse,
    responses={500: {"model": ErrorResponse}},
)
async def clean_items(
    db: aiosqlite.Connection = Depends(get_db),
):
    """Clean all approved junk items and record a cleanup job."""
    result = await clean_approved_items(db)
    return JunkCleanResponse(**result)
