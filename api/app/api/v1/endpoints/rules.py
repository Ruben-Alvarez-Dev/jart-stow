"""Rule endpoints — CRUD for hygiene rules."""


import aiosqlite
from fastapi import APIRouter, Depends, HTTPException, Query
from fastapi.responses import Response

from app.api.v1.schemas import ErrorResponse, RuleCreate, RuleResponse, RuleUpdate
from app.core.database import get_db
from app.infrastructure.sqlite_repository import (
    create_rule,
    delete_rule,
    get_rules,
    update_rule,
)

router = APIRouter(prefix="/rules", tags=["rules"])


@router.get(
    "",
    response_model=dict,
    responses={500: {"model": ErrorResponse}},
)
async def list_rules(
    project_id: int | None = Query(None, ge=1),
    enabled_only: bool = Query(True),
    db: aiosqlite.Connection = Depends(get_db),
):
    """List hygiene rules, optionally filtered by project and enabled state."""
    result = await get_rules(db, project_id=project_id, enabled_only=enabled_only)
    result["rules"] = [RuleResponse(**r) for r in result["rules"]]
    return result


@router.post(
    "",
    status_code=201,
    response_model=RuleResponse,
    responses={400: {"model": ErrorResponse}},
)
async def add_rule(
    body: RuleCreate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Create a new hygiene rule (global or project-scoped)."""
    rule = await create_rule(
        db,
        body.project_id,
        body.pattern,
        body.max_size_bytes,
        body.action,
        body.priority,
    )
    return RuleResponse(**rule)


@router.patch(
    "/{rule_id}",
    response_model=RuleResponse,
    responses={404: {"model": ErrorResponse}},
)
async def patch_rule(
    rule_id: int,
    body: RuleUpdate,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Update fields on an existing rule."""
    updates = body.model_dump(exclude_none=True)
    result = await update_rule(db, rule_id, **updates)
    if result is None:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "RULE_NOT_FOUND",
                    "message": f"Rule with ID {rule_id} not found",
                    "details": None,
                }
            },
        )
    return RuleResponse(**result)


@router.delete(
    "/{rule_id}",
    status_code=204,
    responses={404: {"model": ErrorResponse}},
)
async def remove_rule(
    rule_id: int,
    db: aiosqlite.Connection = Depends(get_db),
):
    """Delete a rule by ID."""
    # First check existence
    rules_result = await get_rules(db, project_id=None, enabled_only=False)
    existing = any(r["id"] == rule_id for r in rules_result["rules"])
    if not existing:
        raise HTTPException(
            status_code=404,
            detail={
                "error": {
                    "code": "RULE_NOT_FOUND",
                    "message": f"Rule with ID {rule_id} not found",
                    "details": None,
                }
            },
        )
    await delete_rule(db, rule_id)
    return Response(status_code=204)
