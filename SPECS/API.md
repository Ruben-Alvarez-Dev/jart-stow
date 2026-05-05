# Jart-Stow API Specification

**Version:** 1.0.0  
**Status:** Draft  
**Author:** Ruben Alvarez  
**Date:** 2026-05-05  

---

## 1. Overview

The Jart-Stow REST API is built with **FastAPI 3.0** (Python) and serves as the programmatic interface to the system. It shares the same SQLite database as the Go daemon and TUI, enabling both internal (TUI) and external (any HTTP client) consumption.

### Technology Stack

| Component | Technology |
|---|---|
| Framework | FastAPI 3.0 |
| ASGI Server | Uvicorn |
| Data Validation | Pydantic v2 |
| API Documentation | OpenAPI 3.0 (auto-generated) |
| Database Access | aiosqlite (async) |
| Testing | pytest + httpx (async) |

---

## 2. Base URL

```
Development:  http://localhost:8420
Production:   http://localhost:8420
```

### Startup

```bash
jart-stow api start           # Starts Uvicorn on port 8420
jart-stow api start --port 9000
jart-stow api stop
jart-stow api status
```

---

## 3. API Versioning

All endpoints are prefixed with `/api/v1/`.

```
/api/v1/projects
/api/v1/exclusions
/api/v1/rules
/api/v1/daemon
/api/v1/reports
```

---

## 4. Endpoints

### 4.1 Health & Metadata

#### `GET /api/v1/health`

System health check.

```
Response 200:
{
  "status": "ok",
  "daemon_running": true,
  "database_connected": true,
  "watched_root": "/Users/ruben/Code",
  "uptime_seconds": 482910
}
```

---

### 4.2 Projects

#### `GET /api/v1/projects`

List all tracked projects.

| Query Param | Type | Default | Description |
|---|---|---|---|
| `status` | string | — | Filter: `active`, `ignored`, `archived` |
| `sort_by` | string | `name` | `name`, `last_scanned`, `created_at` |
| `order` | string | `asc` | `asc`, `desc` |
| `limit` | int | 50 | Max results |
| `offset` | int | 0 | Pagination offset |

```
Response 200:
{
  "total": 47,
  "limit": 50,
  "offset": 0,
  "projects": [
    {
      "id": 1,
      "path": "/Users/ruben/Code/CAAL",
      "name": "CAAL",
      "root_path": "/Users/ruben/Code",
      "last_scanned": "2026-05-05T18:30:00Z",
      "status": "active",
      "created_at": "2026-04-10T12:00:00Z",
      "updated_at": "2026-05-05T18:30:00Z"
    }
  ]
}
```

#### `GET /api/v1/projects/{project_id}`

Get a single project by ID.

```
Response 200:
{
  "id": 1,
  "path": "/Users/ruben/Code/CAAL",
  "name": "CAAL",
  "root_path": "/Users/ruben/Code",
  "last_scanned": "2026-05-05T18:30:00Z",
  "status": "active",
  "exclusion_count": 12,
  "total_excluded_size_bytes": 4100000000,
  "created_at": "2026-04-10T12:00:00Z",
  "updated_at": "2026-05-05T18:30:00Z"
}
```

#### `PATCH /api/v1/projects/{project_id}`

Update project status.

```
Request:
{
  "status": "ignored"
}

Response 200:
{
  "id": 1,
  "status": "ignored",
  "updated_at": "2026-05-05T20:00:00Z"
}
```

---

### 4.3 Exclusions

#### `GET /api/v1/exclusions`

List exclusions with filtering.

| Query Param | Type | Default | Description |
|---|---|---|---|
| `project_id` | int | — | Filter by project |
| `backup_system` | string | — | `time_machine`, `carbon_copy_cloner` |
| `active_only` | bool | `true` | Only show active (not removed) |
| `pattern` | string | — | Filter by pattern matched |
| `sort_by` | string | `applied_at` | `applied_at`, `size_bytes`, `pattern_matched` |
| `order` | string | `desc` | — |
| `limit` | int | 50 | — |
| `offset` | int | 0 | — |

```
Response 200:
{
  "total": 312,
  "total_size_bytes": 8400000000,
  "exclusions": [
    {
      "id": 1,
      "project_id": 1,
      "project_name": "CAAL",
      "folder_path": "/Users/ruben/Code/CAAL/node_modules",
      "pattern_matched": "node_modules",
      "backup_system": "both",
      "size_bytes": 2300000000,
      "applied_at": "2026-05-01T14:30:00Z",
      "removed_at": null
    }
  ]
}
```

#### `POST /api/v1/exclusions`

Manually apply an exclusion.

```
Request:
{
  "project_id": 1,
  "folder_path": "/Users/ruben/Code/CAAL/.venv",
  "pattern_matched": ".venv",
  "backup_systems": ["time_machine", "carbon_copy_cloner"]
}

Response 201:
{
  "id": 313,
  "project_id": 1,
  "folder_path": "/Users/ruben/Code/CAAL/.venv",
  "pattern_matched": ".venv",
  "backup_system": "both",
  "size_bytes": 850000000,
  "applied_at": "2026-05-05T20:05:00Z"
}
```

#### `DELETE /api/v1/exclusions/{exclusion_id}`

Restore (remove) an exclusion.

```
Response 200:
{
  "id": 313,
  "removed_at": "2026-05-05T20:10:00Z",
  "message": "Exclusion removed from Time Machine and CCC"
}
```

---

### 4.4 Rules

#### `GET /api/v1/rules`

List hygiene rules.

| Query Param | Type | Default | Description |
|---|---|---|---|
| `project_id` | int | — | Filter by project (null = global) |
| `enabled_only` | bool | `true` | Only show enabled rules |

```
Response 200:
{
  "total": 8,
  "rules": [
    {
      "id": 1,
      "project_id": null,
      "project_name": null,
      "pattern": "node_modules",
      "max_size_bytes": 524288000,
      "action": "exclude",
      "priority": 10,
      "enabled": true,
      "created_at": "2026-05-01T00:00:00Z",
      "updated_at": "2026-05-01T00:00:00Z"
    }
  ]
}
```

#### `POST /api/v1/rules`

Create a new rule.

```
Request:
{
  "project_id": null,
  "pattern": "node_modules",
  "max_size_bytes": 524288000,
  "action": "exclude",
  "priority": 10
}

Response 201:
{
  "id": 9,
  "project_id": null,
  "pattern": "node_modules",
  "max_size_bytes": 524288000,
  "action": "exclude",
  "priority": 10,
  "enabled": true,
  "created_at": "2026-05-05T20:15:00Z"
}
```

#### `PATCH /api/v1/rules/{rule_id}`

Update a rule.

```
Request:
{
  "max_size_bytes": 1073741824,
  "action": "warn"
}

Response 200:
{
  "id": 9,
  "max_size_bytes": 1073741824,
  "action": "warn",
  "updated_at": "2026-05-05T20:20:00Z"
}
```

#### `DELETE /api/v1/rules/{rule_id}`

Delete a rule.

```
Response 204: (no body)
```

---

### 4.5 Daemon

#### `GET /api/v1/daemon/status`

Current daemon status.

```
Response 200:
{
  "running": true,
  "pid": 12345,
  "watched_root": "/Users/ruben/Code",
  "uptime_seconds": 482910,
  "projects_watched": 47,
  "events_today": 23,
  "last_event_at": "2026-05-05T20:25:00Z"
}
```

#### `POST /api/v1/daemon/start`

Start the daemon.

```
Response 200:
{
  "message": "Daemon started",
  "pid": 12345
}
```

#### `POST /api/v1/daemon/stop`

Stop the daemon.

```
Response 200:
{
  "message": "Daemon stopped"
}
```

#### `GET /api/v1/daemon/events`

Recent daemon events.

| Query Param | Type | Default | Description |
|---|---|---|---|
| `limit` | int | 50 | Max results |
| `event_type` | string | — | Filter by type |

```
Response 200:
{
  "events": [
    {
      "id": 5001,
      "event_type": "exclusion_applied",
      "project_id": 1,
      "folder_path": "/Users/ruben/Code/CAAL/node_modules",
      "details": "{\"size\": \"2.3GB\", \"pattern\": \"node_modules\"}",
      "created_at": "2026-05-05T20:25:00Z"
    }
  ]
}
```

---

### 4.6 Watch Roots

#### `GET /api/v1/watch-roots`

List all configured watch roots.

```
Response 200:
{
  "watch_roots": [
    {
      "id": 1,
      "path": "/Users/ruben/Code",
      "volume_uuid": null,
      "enabled": true,
      "projects_count": 47,
      "created_at": "2026-05-01T00:00:00Z"
    }
  ]
}
```

#### `POST /api/v1/watch-roots`

Add a new watch root.

```
Request:
{
  "path": "/Volumes/ExternalSSD/Projects"
}

Response 201:
{
  "id": 2,
  "path": "/Volumes/ExternalSSD/Projects",
  "volume_uuid": "A1B2C3D4-1234-5678",
  "enabled": true,
  "projects_count": 0,
  "created_at": "2026-05-05T20:30:00Z"
}
```

#### `PATCH /api/v1/watch-roots/{root_id}`

Enable/disable a watch root.

```
Request:
{
  "enabled": false
}

Response 200:
{
  "id": 2,
  "enabled": false
}
```

#### `DELETE /api/v1/watch-roots/{root_id}`

Remove a watch root (does not delete project data).

```
Response 204:
```

---

### 4.7 Junk Management

#### `GET /api/v1/junk/categories`

List available junk categories.

```
Response 200:
{
  "categories": [
    {
      "id": 1,
      "name": "unused_docker_images",
      "scanner": "docker",
      "verify_required": true,
      "enabled": false
    }
  ]
}
```

#### `PATCH /api/v1/junk/categories/{category_id}`

Enable/disable a category for scanning.

```
Request:
{
  "enabled": true
}

Response 200:
{
  "id": 1,
  "enabled": true
}
```

#### `POST /api/v1/junk/scan`

Trigger a junk scan for enabled categories.

```
Request:
{
  "category_ids": [1, 2]
}

Response 202:
{
  "message": "Junk scan started",
  "scan_job_id": 42
}
```

#### `GET /api/v1/junk/items`

List junk items pending review.

| Query Param | Type | Default | Description |
|---|---|---|---|
| `category_id` | int | — | Filter by category |
| `verified` | int | `0` | `-1` (skipped), `0` (pending), `1` (approved) |
| `limit` | int | 50 | Max results |

```
Response 200:
{
  "total": 23,
  "total_size_bytes": 6200000000,
  "items": [
    {
      "id": 1,
      "category_name": "unused_docker_images",
      "path": "docker://ubuntu:18.04",
      "description": "Docker image ubuntu:18.04 (unused, 3 years old)",
      "size_bytes": 127000000,
      "verified_by_user": 0,
      "created_at": "2026-05-05T20:30:00Z"
    }
  ]
}
```

#### `PATCH /api/v1/junk/items/{item_id}`

Approve or skip an item.

```
Request:
{
  "verified_by_user": 1
}

Response 200:
{
  "id": 1,
  "verified_by_user": 1
}
```

#### `POST /api/v1/junk/items/batch`

Batch approve or skip items.

```
Request:
{
  "item_ids": [1, 2, 3, 5, 8],
  "verified_by_user": 1
}

Response 200:
{
  "updated": 5
}
```

#### `POST /api/v1/junk/clean`

Clean all approved items.

```
Response 200:
{
  "cleaned": 8,
  "total_size_freed_bytes": 2400000000,
  "cleanup_job_id": 10
}
```

---

### 4.8 Reports

#### `GET /api/v1/reports/summary`

Aggregated statistics.

```
Response 200:
{
  "projects_total": 47,
  "projects_active": 47,
  "exclusions_active": 312,
  "exclusions_total_size_bytes": 8400000000,
  "exclusions_total_size_human": "8.4 GB",
  "breakdown_by_pattern": {
    "node_modules": 4368000000,
    ".venv": 2016000000,
    ".next": 1008000000,
    "build": 588000000,
    "other": 420000000
  },
  "breakdown_by_system": {
    "both": 5712000000,
    "carbon_copy_cloner": 1680000000,
    "time_machine": 1008000000
  }
}
```

#### `GET /api/v1/reports/history`

Exclusion history over time.

| Query Param | Type | Default | Description |
|---|---|---|---|
| `days` | int | 30 | Number of days to include |

```
Response 200:
{
  "history": [
    {
      "date": "2026-05-05",
      "exclusions_added": 4,
      "exclusions_removed": 0,
      "size_added_bytes": 1200000000,
      "size_removed_bytes": 0
    }
  ]
}
```

---

## 5. Error Handling

All errors return a consistent JSON structure:

```
Response 4xx/5xx:
{
  "error": {
    "code": "PROJECT_NOT_FOUND",
    "message": "Project with ID 999 not found",
    "details": null
  }
}
```

### Error Codes

| HTTP Status | Code | Description |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Invalid request body or params |
| 404 | `PROJECT_NOT_FOUND` | Project ID does not exist |
| 404 | `EXCLUSION_NOT_FOUND` | Exclusion ID does not exist |
| 404 | `RULE_NOT_FOUND` | Rule ID does not exist |
| 409 | `EXCLUSION_ALREADY_EXISTS` | Path already excluded |
| 500 | `DAEMON_ERROR` | Daemon operation failed |
| 500 | `DATABASE_ERROR` | SQLite operation failed |
| 503 | `DAEMON_NOT_RUNNING` | Daemon is not active |

---

## 6. OpenAPI Documentation

FastAPI auto-generates:

- **Swagger UI:** `http://localhost:8420/docs`
- **ReDoc:** `http://localhost:8420/redoc`
- **OpenAPI JSON:** `http://localhost:8420/openapi.json`

---

## 7. Pydantic Schemas

All request/response models are defined in `api/app/api/v1/schemas/` using Pydantic v2 with strict validation, no defaults that imply mock data, and comprehensive field descriptions.

```python
from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional

class ProjectResponse(BaseModel):
    id: int
    path: str
    name: str
    root_path: str
    last_scanned: Optional[datetime] = None
    status: str
    created_at: datetime
    updated_at: datetime

class ExclusionCreate(BaseModel):
    project_id: int = Field(..., gt=0, description="Existing project ID")
    folder_path: str = Field(..., min_length=1, description="Absolute path to exclude")
    pattern_matched: str = Field(..., min_length=1)
    backup_systems: list[str] = Field(
        default=["time_machine", "carbon_copy_cloner"],
        description="Target backup systems"
    )
```

---

## 8. Testing

API tests use `pytest` + `httpx.AsyncClient` against a temporary SQLite database populated with real fixtures **only** from actual system state — never hardcoded mock data.

```python
# api/tests/conftest.py
@pytest.fixture
async def client(tmp_path):
    db_path = tmp_path / "test.db"
    # DB is initialized empty, tests populate via API calls
    app = create_app(db_path=str(db_path))
    async with httpx.AsyncClient(app=app, base_url="http://test") as ac:
        yield ac
```

Test data is generated by calling the real API endpoints against a clean database, ensuring 100% production-realistic behavior.
