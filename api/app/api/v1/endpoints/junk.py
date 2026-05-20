from fastapi import APIRouter
from ....db import connection_scope
from ....models import JunkCategoryOut, JunkItemOut
from ..converters import row_to_junk_category, row_to_junk_item
from ..dependencies import ensure_db_exists

router = APIRouter()

@router.get("/junk/categories", response_model=list[JunkCategoryOut])
async def list_junk_categories() -> list[JunkCategoryOut]:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT * FROM junk_categories ORDER BY name ASC")
        rows = await cursor.fetchall()
    return [row_to_junk_category(row) for row in rows]

@router.get("/junk/items", response_model=list[JunkItemOut])
async def list_junk_items(
    pending_only: bool = False,
    approved_only: bool = False,
    cleaned_only: bool = False,
) -> list[JunkItemOut]:
    ensure_db_exists()
    query = "SELECT * FROM junk_items"
    clauses: list[str] = []
    params: list[object] = []
    if pending_only:
        clauses.append("verified_by_user = 0")
    if approved_only:
        clauses.append("verified_by_user = 1")
    if cleaned_only:
        clauses.append("cleaned_at IS NOT NULL")
    if clauses:
        query += " WHERE " + " AND ".join(clauses)
    query += " ORDER BY created_at DESC, id DESC"
    async with connection_scope() as conn:
        cursor = await conn.execute(query, params)
        rows = await cursor.fetchall()
    return [row_to_junk_item(row) for row in rows]
