"""Async SQLite database connection using aiosqlite."""

from collections.abc import AsyncGenerator

import aiosqlite

from app.core.config import DB_PATH


async def get_db() -> AsyncGenerator[aiosqlite.Connection, None]:
    db = await aiosqlite.connect(DB_PATH)
    db.row_factory = aiosqlite.Row
    await db.execute("PRAGMA journal_mode=WAL")
    await db.execute("PRAGMA foreign_keys=ON")
    try:
        yield db
    finally:
        await db.close()
