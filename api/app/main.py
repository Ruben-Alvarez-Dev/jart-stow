"""FastAPI application factory for the Jart-Stow REST API."""

from fastapi import FastAPI

from app.api.v1.router import api_router


def create_app() -> FastAPI:
    app = FastAPI(
        title="Jart-Stow API",
        version="0.1.0",
        description="macOS development hygiene & backup exclusion manager",
    )
    app.include_router(api_router, prefix="/api/v1")
    return app


app = create_app()
