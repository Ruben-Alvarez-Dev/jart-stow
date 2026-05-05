"""Test fixtures for the Jart-Stow REST API.

Provides an async HTTP client backed by a temporary SQLite database
initialised with the full production schema (but no project data).
"""

from pathlib import Path

import aiosqlite
import pytest_asyncio
from httpx import ASGITransport, AsyncClient

PROJECT_ROOT = Path(__file__).resolve().parents[2]
MIGRATION_FILE = (
    PROJECT_ROOT
    / "internal"
    / "adapters"
    / "sqlite"
    / "migrations"
    / "001_initial_schema.sql"
)


def _read_migration() -> str:
    """Read the production schema SQL from the Go migration file."""
    return MIGRATION_FILE.read_text()


@pytest_asyncio.fixture
async def client(tmp_path):
    """Return an async httpx client connected to a temp-SQLite-backed app.

    The database contains the full schema (tables, indexes, seed data for
    junk_categories) but zero project-level data.  Tests must populate the
    database exclusively through the API.
    """
    db_path = tmp_path / "test.db"
    migration_sql = _read_migration()

    # Initialise the schema
    db = await aiosqlite.connect(str(db_path))
    await db.executescript(migration_sql)
    await db.close()

    # Import app internals lazily so the test DB path is isolated
    from app.core.database import get_db
    from app.main import app

    async def override_get_db():
        db = await aiosqlite.connect(str(db_path))
        db.row_factory = aiosqlite.Row
        await db.execute("PRAGMA journal_mode=WAL")
        await db.execute("PRAGMA foreign_keys=ON")
        try:
            yield db
        finally:
            await db.close()

    app.dependency_overrides[get_db] = override_get_db

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        yield ac

    app.dependency_overrides.clear()
