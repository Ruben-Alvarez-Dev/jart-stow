"""Async API client for the Jart-Stow FastAPI backend."""

from __future__ import annotations

import httpx


class JartStowClient:
    """Thin async wrapper around all /api/v1 endpoints."""

    def __init__(self, base_url: str = "http://localhost:8420") -> None:
        self._client = httpx.AsyncClient(base_url=base_url, timeout=10.0)

    async def close(self) -> None:
        await self._client.aclose()

    # --- helpers ---

    async def _get(self, path: str, **params: object) -> dict:
        r = await self._client.get(f"/api/v1{path}", params=params)
        r.raise_for_status()
        return r.json()

    async def _post(self, path: str, json: dict | None = None) -> dict:
        r = await self._client.post(f"/api/v1{path}", json=json)
        r.raise_for_status()
        return r.json()

    async def _patch(self, path: str, json: dict) -> dict:
        r = await self._client.patch(f"/api/v1{path}", json=json)
        r.raise_for_status()
        return r.json()

    async def _delete(self, path: str) -> dict | None:
        r = await self._client.delete(f"/api/v1{path}")
        r.raise_for_status()
        return r.json() if r.content else None

    # --- health ---

    async def get_health(self) -> dict:
        return await self._get("/health")

    # --- daemon ---

    async def get_daemon_status(self) -> dict:
        return await self._get("/daemon/status")

    async def start_daemon(self) -> dict:
        return await self._post("/daemon/start")

    async def stop_daemon(self) -> dict:
        return await self._post("/daemon/stop")

    async def get_daemon_events(self, *, limit: int = 50, event_type: str | None = None) -> dict:
        params: dict = {"limit": limit}
        if event_type:
            params["event_type"] = event_type
        return await self._get("/daemon/events", **params)

    # --- projects ---

    async def get_projects(
        self,
        *,
        status: str | None = None,
        sort_by: str = "name",
        order: str = "asc",
        limit: int = 50,
        offset: int = 0,
    ) -> dict:
        params: dict = {"sort_by": sort_by, "order": order, "limit": limit, "offset": offset}
        if status:
            params["status"] = status
        return await self._get("/projects", **params)

    async def get_project(self, project_id: int) -> dict:
        return await self._get(f"/projects/{project_id}")

    async def update_project(self, project_id: int, status: str) -> dict:
        return await self._patch(f"/projects/{project_id}", json={"status": status})

    # --- exclusions ---

    async def get_exclusions(
        self,
        *,
        project_id: int | None = None,
        backup_system: str | None = None,
        active_only: bool = True,
        pattern: str | None = None,
        sort_by: str = "applied_at",
        order: str = "desc",
        limit: int = 50,
        offset: int = 0,
    ) -> dict:
        params: dict = {
            "active_only": active_only,
            "sort_by": sort_by,
            "order": order,
            "limit": limit,
            "offset": offset,
        }
        if project_id is not None:
            params["project_id"] = project_id
        if backup_system:
            params["backup_system"] = backup_system
        if pattern:
            params["pattern"] = pattern
        return await self._get("/exclusions", **params)

    async def create_exclusion(
        self,
        project_id: int,
        folder_path: str,
        pattern_matched: str,
        backup_systems: list[str] | None = None,
    ) -> dict:
        body: dict = {
            "project_id": project_id,
            "folder_path": folder_path,
            "pattern_matched": pattern_matched,
        }
        if backup_systems:
            body["backup_systems"] = backup_systems
        return await self._post("/exclusions", json=body)

    async def delete_exclusion(self, exclusion_id: int) -> dict | None:
        return await self._delete(f"/exclusions/{exclusion_id}")

    # --- rules ---

    async def get_rules(self, *, project_id: int | None = None, enabled_only: bool = True) -> dict:
        params: dict = {"enabled_only": enabled_only}
        if project_id is not None:
            params["project_id"] = project_id
        return await self._get("/rules", **params)

    async def create_rule(
        self,
        pattern: str,
        max_size_bytes: int,
        action: str,
        priority: int = 10,
        project_id: int | None = None,
    ) -> dict:
        body: dict = {
            "pattern": pattern,
            "max_size_bytes": max_size_bytes,
            "action": action,
            "priority": priority,
        }
        if project_id is not None:
            body["project_id"] = project_id
        return await self._post("/rules", json=body)

    async def update_rule(self, rule_id: int, **fields: object) -> dict:
        return await self._patch(f"/rules/{rule_id}", json=fields)

    async def delete_rule(self, rule_id: int) -> None:
        await self._delete(f"/rules/{rule_id}")

    # --- watch roots ---

    async def get_watch_roots(self) -> dict:
        return await self._get("/watch-roots")

    async def create_watch_root(self, path: str) -> dict:
        return await self._post("/watch-roots", json={"path": path})

    async def update_watch_root(self, root_id: int, enabled: bool) -> dict:
        return await self._patch(f"/watch-roots/{root_id}", json={"enabled": enabled})

    async def delete_watch_root(self, root_id: int) -> None:
        await self._delete(f"/watch-roots/{root_id}")

    # --- junk ---

    async def get_junk_categories(self) -> dict:
        return await self._get("/junk/categories")

    async def update_junk_category(self, category_id: int, enabled: bool) -> dict:
        return await self._patch(f"/junk/categories/{category_id}", json={"enabled": enabled})

    async def trigger_junk_scan(self, category_ids: list[int]) -> dict:
        return await self._post("/junk/scan", json={"category_ids": category_ids})

    async def get_junk_items(
        self,
        *,
        category_id: int | None = None,
        verified: int = 0,
        limit: int = 50,
    ) -> dict:
        params: dict = {"verified": verified, "limit": limit}
        if category_id is not None:
            params["category_id"] = category_id
        return await self._get("/junk/items", **params)

    async def update_junk_item(self, item_id: int, verified_by_user: int) -> dict:
        return await self._patch(f"/junk/items/{item_id}", json={"verified_by_user": verified_by_user})

    async def batch_update_junk_items(self, item_ids: list[int], verified_by_user: int) -> dict:
        return await self._post("/junk/items/batch", json={"item_ids": item_ids, "verified_by_user": verified_by_user})

    async def clean_approved_items(self) -> dict:
        return await self._post("/junk/clean")

    # --- reports ---

    async def get_report_summary(self) -> dict:
        return await self._get("/reports/summary")

    async def get_report_history(self, *, days: int = 30) -> dict:
        return await self._get("/reports/history", days=days)
