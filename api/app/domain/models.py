"""Pydantic v2 domain models for the Jart-Stow API.

All models mirror the Go domain entities and map to the SQLite schema
defined in internal/adapters/sqlite/migrations/001_initial_schema.sql.
"""

from datetime import datetime

from pydantic import BaseModel, Field

# ---------------------------------------------------------------------------
# Project models
# ---------------------------------------------------------------------------


class ProjectResponse(BaseModel):
    """Public project representation returned in list endpoints."""

    id: int
    path: str
    name: str
    root_path: str
    last_scanned: datetime | None = None
    status: str
    created_at: datetime
    updated_at: datetime


class ProjectDetail(ProjectResponse):
    """Extended project representation with exclusion statistics."""

    exclusion_count: int
    total_excluded_size_bytes: int


class ProjectUpdate(BaseModel):
    """Payload for updating a project's status."""

    status: str = Field(..., pattern=r"^(active|ignored|archived)$")


# ---------------------------------------------------------------------------
# Exclusion models
# ---------------------------------------------------------------------------


class ExclusionResponse(BaseModel):
    """Exclusion record as returned by the API."""

    id: int
    project_id: int
    project_name: str
    folder_path: str
    pattern_matched: str
    backup_system: str
    size_bytes: int
    applied_at: datetime
    removed_at: datetime | None = None


class ExclusionCreate(BaseModel):
    """Payload for creating a new exclusion."""

    project_id: int = Field(..., gt=0, description="Existing project ID")
    folder_path: str = Field(..., min_length=1, description="Absolute path to exclude")
    pattern_matched: str = Field(..., min_length=1, description="Pattern that triggered exclusion")
    backup_systems: list[str] = Field(
        default=["time_machine", "carbon_copy_cloner"],
        description="Target backup systems",
    )


# ---------------------------------------------------------------------------
# Rule models
# ---------------------------------------------------------------------------


class RuleResponse(BaseModel):
    """Hygiene rule as returned by the API."""

    id: int
    project_id: int | None = None
    project_name: str | None = None
    pattern: str
    max_size_bytes: int
    action: str
    priority: int
    enabled: bool
    created_at: datetime
    updated_at: datetime


class RuleCreate(BaseModel):
    """Payload for creating a new rule."""

    project_id: int | None = Field(None, description="NULL for global rule")
    pattern: str = Field(..., min_length=1)
    max_size_bytes: int = Field(..., ge=0)
    action: str = Field(..., pattern=r"^(warn|alert|exclude|clean)$")
    priority: int = Field(10, ge=0)


class RuleUpdate(BaseModel):
    """Payload for updating an existing rule. All fields optional."""

    max_size_bytes: int | None = Field(None, ge=0)
    action: str | None = Field(None, pattern=r"^(warn|alert|exclude|clean)$")
    enabled: bool | None = None


# ---------------------------------------------------------------------------
# Watch Root models
# ---------------------------------------------------------------------------


class WatchRootResponse(BaseModel):
    """Watch root as returned by the API."""

    id: int
    path: str
    volume_uuid: str | None = None
    enabled: bool
    projects_count: int
    created_at: datetime


class WatchRootCreate(BaseModel):
    """Payload for adding a new watch root."""

    path: str = Field(..., min_length=1, description="Absolute path to watch")


class WatchRootUpdate(BaseModel):
    """Payload for toggling a watch root."""

    enabled: bool


# ---------------------------------------------------------------------------
# Daemon models
# ---------------------------------------------------------------------------


class DaemonStatusResponse(BaseModel):
    """Current daemon status."""

    running: bool
    pid: int | None = None
    watched_root: str | None = None
    uptime_seconds: int | None = None
    projects_watched: int
    events_today: int
    last_event_at: datetime | None = None


class DaemonEventResponse(BaseModel):
    """Individual daemon event from the audit log."""

    id: int
    event_type: str
    project_id: int | None = None
    folder_path: str | None = None
    details: str | None = None
    created_at: datetime


# ---------------------------------------------------------------------------
# Junk models
# ---------------------------------------------------------------------------


class JunkCategoryResponse(BaseModel):
    """Junk category configuration."""

    id: int
    name: str
    scanner: str
    verify_required: bool
    enabled: bool


class JunkCategoryUpdate(BaseModel):
    """Payload for toggling a junk category."""

    enabled: bool


class JunkItemResponse(BaseModel):
    """Individual junk item pending review."""

    id: int
    category_name: str
    path: str
    description: str
    size_bytes: int
    verified_by_user: int
    created_at: datetime


class JunkItemUpdate(BaseModel):
    """Payload for approving or skipping a single junk item."""

    verified_by_user: int = Field(..., ge=-1, le=1)


class JunkBatchUpdate(BaseModel):
    """Payload for batch approve/skip of junk items."""

    item_ids: list[int] = Field(..., min_length=1)
    verified_by_user: int = Field(..., ge=-1, le=1)


class JunkScanRequest(BaseModel):
    """Payload for triggering a junk scan."""

    category_ids: list[int] = Field(..., min_length=1)


class JunkScanResponse(BaseModel):
    """Response after triggering a junk scan."""

    message: str
    scan_job_id: int


class JunkCleanResponse(BaseModel):
    """Response after cleaning approved junk items."""

    cleaned: int
    total_size_freed_bytes: int
    cleanup_job_id: int


# ---------------------------------------------------------------------------
# Report models
# ---------------------------------------------------------------------------


class ReportSummaryResponse(BaseModel):
    """Aggregated system statistics."""

    projects_total: int
    projects_active: int
    exclusions_active: int
    exclusions_total_size_bytes: int
    exclusions_total_size_human: str
    breakdown_by_pattern: dict[str, int]
    breakdown_by_system: dict[str, int]


class ReportHistoryPoint(BaseModel):
    """Single day in the exclusion history timeline."""

    date: str
    exclusions_added: int
    exclusions_removed: int
    size_added_bytes: int
    size_removed_bytes: int


# ---------------------------------------------------------------------------
# Health / Error models
# ---------------------------------------------------------------------------


class HealthResponse(BaseModel):
    """System health check result."""

    status: str
    daemon_running: bool
    database_connected: bool
    watched_root: str | None = None
    uptime_seconds: int | None = None


class ErrorDetail(BaseModel):
    """Structured error information."""

    code: str
    message: str
    details: str | None = None


class ErrorResponse(BaseModel):
    """Consistent error envelope returned by all endpoints."""

    error: ErrorDetail
