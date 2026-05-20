from fastapi import APIRouter
from .endpoints import (
    health,
    projects,
    exclusions,
    rules,
    watch_roots,
    junk,
    reports,
)

api_v1_router = APIRouter()

api_v1_router.include_router(health.router, tags=["health"])
api_v1_router.include_router(projects.router, tags=["projects"])
api_v1_router.include_router(exclusions.router, tags=["exclusions"])
api_v1_router.include_router(rules.router, tags=["rules"])
api_v1_router.include_router(watch_roots.router, tags=["watch-roots"])
api_v1_router.include_router(junk.router, tags=["junk"])
api_v1_router.include_router(reports.router, tags=["reports"])
