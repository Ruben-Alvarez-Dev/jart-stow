from fastapi import APIRouter, HTTPException, status
from ....db import connection_scope
from ....models import RuleCreate, RuleOut, RuleUpdate
from ..converters import row_to_rule
from ..dependencies import ensure_db_exists

router = APIRouter()

@router.get("/rules", response_model=list[RuleOut])
async def list_rules() -> list[RuleOut]:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("SELECT * FROM rules ORDER BY priority DESC, id DESC")
        rows = await cursor.fetchall()
    return [row_to_rule(row) for row in rows]

@router.post("/rules", response_model=RuleOut, status_code=status.HTTP_201_CREATED)
async def create_rule(payload: RuleCreate) -> RuleOut:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute(
            """
            INSERT INTO rules (project_id, pattern, max_size_bytes, action, priority, enabled)
            VALUES (?, ?, ?, ?, ?, ?)
            """,
            (
                payload.project_id,
                payload.pattern,
                payload.max_size_bytes,
                payload.action,
                payload.priority,
                int(payload.enabled),
            ),
        )
        cursor = await conn.execute("SELECT * FROM rules WHERE id = ?", (cursor.lastrowid,))
        row = await cursor.fetchone()
    return row_to_rule(row)

@router.patch("/rules/{rule_id}", response_model=RuleOut)
async def update_rule(rule_id: int, payload: RuleUpdate) -> RuleOut:
    ensure_db_exists()
    updates = payload.model_dump(exclude_unset=True)
    if not updates:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="no fields provided")

    field_map = {
        "pattern": "pattern",
        "max_size_bytes": "max_size_bytes",
        "action": "action",
        "priority": "priority",
        "enabled": "enabled",
    }
    assignments: list[str] = []
    values: list[object] = []
    for key, value in updates.items():
        assignments.append(f"{field_map[key]} = ?")
        values.append(int(value) if key == "enabled" else value)
    assignments.append("updated_at = datetime('now')")
    values.append(rule_id)

    async with connection_scope() as conn:
        cursor = await conn.execute(
            f"UPDATE rules SET {', '.join(assignments)} WHERE id = ?", values
        )
        if cursor.rowcount == 0:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="rule not found")
        cursor = await conn.execute("SELECT * FROM rules WHERE id = ?", (rule_id,))
        row = await cursor.fetchone()
    return row_to_rule(row)

@router.delete("/rules/{rule_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_rule(rule_id: int) -> None:
    ensure_db_exists()
    async with connection_scope() as conn:
        cursor = await conn.execute("DELETE FROM rules WHERE id = ?", (rule_id,))
        if cursor.rowcount == 0:
            raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="rule not found")
