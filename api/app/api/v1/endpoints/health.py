from fastapi import APIRouter
from ....db import get_db_info
from ....models import HealthResponse
from ....config import api_version

router = APIRouter()

@router.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    db_info = get_db_info()
    return HealthResponse(
        status="ok" if db_info.exists else "degraded",
        version=api_version(),
        db_path=db_info.path,
        db_exists=db_info.exists,
    )
