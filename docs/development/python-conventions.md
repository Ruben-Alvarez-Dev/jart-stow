# Python Conventions

## Style

- `ruff` for linting and formatting (enforced by pre-commit).
- Type hints on all public functions and methods.
- Maximum line length: 100 characters.

## Naming

- **Modules:** lowercase, underscores. `sqlite_repository`, `junk_scanner`.
- **Classes:** PascalCase. `ProjectService`, `ExclusionRepository`.
- **Functions/Methods:** snake_case. `find_by_path`, `list_active`.
- **Constants:** UPPER_SNAKE_CASE. `DEFAULT_PORT`, `MAX_PAGE_SIZE`.
- **Private members:** prefix `_`. `_db`, `_validate_path`.

## Project Structure

```
api/
├── app/
│   ├── main.py             # FastAPI app factory
│   ├── api/v1/
│   │   ├── router.py       # Route aggregation
│   │   ├── endpoints/      # One file per resource
│   │   └── schemas/        # Pydantic models
│   ├── core/               # Config, database connection
│   ├── domain/             # Domain models, services
│   └── infrastructure/     # SQLite repository implementation
└── tests/                  # Mirrors app structure
```

## FastAPI Patterns

- Endpoints return Pydantic response models (never dicts).
- Use dependency injection for database sessions.
- Validate all inputs with Pydantic `BaseModel`.
- Error responses use a consistent `ErrorResponse` schema.

```python
@router.get("/{project_id}", response_model=ProjectResponse)
async def get_project(
    project_id: int,
    db: aiosqlite.Connection = Depends(get_db),
) -> ProjectResponse:
    ...
```

## Database Access

- All SQL queries go through the repository layer.
- Use `aiosqlite` for async SQLite access.
- Parameterize all queries — never use f-strings for SQL.
- Use context managers for connections: `async with db.execute(...) as cursor:`.

## Testing

- Use `pytest` with `pytest-asyncio`.
- Use `httpx.AsyncClient` for endpoint testing.
- Test database is a temporary SQLite file, not an in-memory database.
- Never mock the repository in API tests — test against a real SQLite.
