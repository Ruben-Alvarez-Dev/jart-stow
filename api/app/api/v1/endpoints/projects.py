from fastapi import APIRouter, Query
from ....db import connection_scope
from ....models import ProjectOut
from ..converters import row_to_project
from ..dependencies import ensure_db_exists

router = APIRouter()

@router.get("/projects", response_model=list[ProjectOut])
async def list_projects(
    status_filter: str | None = Query(default=None, alias="status")
) -> list[ProjectOut]:
    ensure_db_exists()
    query = "SELECT * FROM projects"
    params: list[object] = []
    if status_filter:
        query += " WHERE status = ?"
        params.append(status_filter)
    query += " ORDER BY updated_at DESC, id DESC"
    async with connection_scope() as conn:
        cursor = await conn.execute(query, params)
        rows = await cursor.fetchall()
    return [row_to_project(row) for row in rows]
