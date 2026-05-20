from fastapi import HTTPException, status
from ...db import get_db_info

def ensure_db_exists() -> None:
    db_info = get_db_info()
    if not db_info.exists:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"database not found at {db_info.path}",
        )
