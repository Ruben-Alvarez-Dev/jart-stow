"""Integration tests for Jart-Stow REST API endpoints."""
import pytest
from httpx import AsyncClient


@pytest.mark.asyncio
async def test_health_check(client: AsyncClient):
    """Health endpoint returns 200 with expected structure."""
    response = await client.get("/api/v1/health")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "ok"
    assert "database_connected" in data
    assert "daemon_running" in data


@pytest.mark.asyncio
async def test_list_projects_empty(client: AsyncClient):
    """Projects endpoint returns empty list when no projects exist."""
    response = await client.get("/api/v1/projects")
    assert response.status_code == 200
    data = response.json()
    assert data["total"] == 0
    assert data["projects"] == []


@pytest.mark.asyncio
async def test_list_exclusions_empty(client: AsyncClient):
    """Exclusions endpoint returns empty list when no exclusions exist."""
    response = await client.get("/api/v1/exclusions")
    assert response.status_code == 200
    data = response.json()
    assert data["total"] == 0
    assert data["exclusions"] == []


@pytest.mark.asyncio
async def test_list_rules_empty(client: AsyncClient):
    """Rules endpoint returns empty list when no rules exist."""
    response = await client.get("/api/v1/rules")
    assert response.status_code == 200
    data = response.json()
    assert data["total"] == 0
    assert data["rules"] == []


@pytest.mark.asyncio
async def test_junk_categories_exist(client: AsyncClient):
    """Junk categories are seeded from the migration."""
    response = await client.get("/api/v1/junk/categories")
    assert response.status_code == 200
    data = response.json()
    assert len(data["categories"]) == 10


@pytest.mark.asyncio
async def test_watch_roots_empty(client: AsyncClient):
    """Watch roots endpoint returns empty list."""
    response = await client.get("/api/v1/watch-roots")
    assert response.status_code == 200
    data = response.json()
    assert data["watch_roots"] == []


@pytest.mark.asyncio
async def test_report_summary_empty(client: AsyncClient):
    """Report summary returns zero stats with empty DB."""
    response = await client.get("/api/v1/reports/summary")
    assert response.status_code == 200
    data = response.json()
    assert data["projects_total"] == 0
    assert data["exclusions_active"] == 0


@pytest.mark.asyncio
async def test_daemon_events_empty(client: AsyncClient):
    """Daemon events returns empty list."""
    response = await client.get("/api/v1/daemon/events")
    assert response.status_code == 200
    data = response.json()
    assert data["events"] == []


@pytest.mark.asyncio
async def test_junk_items_empty(client: AsyncClient):
    """Junk items returns empty when nothing scanned."""
    response = await client.get("/api/v1/junk/items")
    assert response.status_code == 200
    data = response.json()
    assert data["total"] == 0
    assert data["items"] == []


@pytest.mark.asyncio
async def test_get_project_not_found(client: AsyncClient):
    """Non-existent project returns 404."""
    response = await client.get("/api/v1/projects/99999")
    assert response.status_code == 404


@pytest.mark.asyncio
async def test_get_exclusion_not_found(client: AsyncClient):
    """Non-existent exclusion delete returns 404."""
    response = await client.delete("/api/v1/exclusions/99999")
    assert response.status_code == 404


@pytest.mark.asyncio
async def test_get_rule_not_found(client: AsyncClient):
    """Non-existent rule returns 404."""
    response = await client.patch("/api/v1/rules/99999", json={"enabled": True})
    assert response.status_code == 404
