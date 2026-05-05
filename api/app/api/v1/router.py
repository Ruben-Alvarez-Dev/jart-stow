"""API v1 router — includes all endpoint routers with proper prefixes and tags."""

from fastapi import APIRouter

from app.api.v1.endpoints import (
    daemon,
    exclusions,
    health,
    junk,
    projects,
    reports,
    rules,
    watch_roots,
)

api_router = APIRouter()

api_router.include_router(projects.router)
api_router.include_router(exclusions.router)
api_router.include_router(rules.router)
api_router.include_router(daemon.router)
api_router.include_router(watch_roots.router)
api_router.include_router(junk.router)
api_router.include_router(reports.router)
api_router.include_router(health.router)
