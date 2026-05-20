from __future__ import annotations

from fastapi import FastAPI

from .config import api_version
from .api.v1.router import api_v1_router

app = FastAPI(
    title="Jart-Stow API",
    version=api_version(),
    description="SQLite-backed API for Jart-Stow projects, exclusions, rules, and hygiene data.",
)

# Include the modular API v1 router
app.include_router(api_v1_router)
