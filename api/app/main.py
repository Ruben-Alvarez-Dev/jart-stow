from __future__ import annotations

import time
import logging
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from .config import api_version
from .api.v1.router import api_v1_router

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Jart-Stow API",
    version=api_version(),
    description="SQLite-backed API for Jart-Stow projects, exclusions, rules, and hygiene data.",
)

@app.middleware("http")
async def log_requests(request: Request, call_next):
    start_time = time.time()
    response = await call_next(request)
    duration = time.time() - start_time
    logger.info(
        f"method={request.method} path={request.url.path} status={response.status_code} duration={duration:.4f}s"
    )
    return response

@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    return JSONResponse(
        status_code=500,
        content={"message": "An unexpected error occurred", "detail": str(exc)},
    )

# Include the modular API v1 router
app.include_router(api_v1_router)
