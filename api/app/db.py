from __future__ import annotations

from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
import aiosqlite

from .config import default_db_path


@dataclass(frozen=True)
class DatabaseInfo:
    path: str
    exists: bool


def get_db_info() -> DatabaseInfo:
    path = default_db_path()
    return DatabaseInfo(path=path, exists=Path(path).exists())


async def get_connection() -> aiosqlite.Connection:
    conn = await aiosqlite.connect(default_db_path())
    conn.row_factory = aiosqlite.Row
    await conn.execute("PRAGMA foreign_keys = ON")
    await conn.execute("PRAGMA journal_mode = WAL")
    await conn.execute("PRAGMA busy_timeout = 5000")
    return conn


@asynccontextmanager
async def connection_scope() -> AsyncGenerator[aiosqlite.Connection, None]:
    conn = await get_connection()
    try:
        yield conn
        await conn.commit()
    finally:
        await conn.close()


def parse_timestamp(value: str | None) -> datetime | None:
    if not value:
        return None
    normalized = value.replace("T", " ")
    try:
        return datetime.fromisoformat(normalized)
    except ValueError:
        return None
