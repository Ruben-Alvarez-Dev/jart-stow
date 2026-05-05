"""Application configuration."""

import os
from pathlib import Path


def get_db_path() -> str:
    env = os.environ.get("JART_STOW_DB_PATH", "")
    if env:
        return env
    home = Path.home()
    data_dir = home / ".local" / "share" / "jart-stow"
    data_dir.mkdir(parents=True, exist_ok=True)
    return str(data_dir / "jart-stow.db")


DB_PATH = get_db_path()
