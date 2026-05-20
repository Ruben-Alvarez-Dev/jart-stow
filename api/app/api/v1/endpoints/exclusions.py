from fastapi import APIRouter
from ....db import connection_scope
from ....models import ExclusionOut
from ..converters import row_to_exclusion
from ..dependencies import ensure_db_exists

router = APIRouter()

@router.get("/exclusions", response_model=list[ExclusionOut])
async def list_exclusions(active_only: bool = True) -> list[ExclusionOut]:
    ensure_db_exists()
    query = "SELECT * FROM exclusions"
    if active_only:
        query += " WHERE removed_at IS NULL"
    query += " ORDER BY applied_at DESC, id DESC"
    async with connection_scope() as conn:
        cursor = await conn.execute(query)
        rows = await cursor.fetchall()
    return [row_to_exclusion(row) for row in rows]
