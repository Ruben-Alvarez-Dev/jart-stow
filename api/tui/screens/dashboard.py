"""Dashboard screen — system status, quick stats, and recent activity."""

from __future__ import annotations

import asyncio

from textual.app import ComposeResult
from textual.containers import Container, Horizontal, VerticalScroll
from textual.reactive import reactive
from textual.screen import Screen
from textual.widgets import DataTable, Label, Static

from tui.widgets.nav_bar import NavBar
from tui.widgets.status_card import StatusCard


def _fmt_bytes(n: int) -> str:
    for unit in ("B", "KB", "MB", "GB", "TB"):
        if abs(n) < 1024:
            return f"{n:.1f} {unit}"
        n /= 1024
    return f"{n:.1f} PB"


class DashboardScreen(Screen):
    """Main dashboard showing system overview."""

    BINDINGS = [
        ("escape", "go_back", "Back"),
    ]

    api_connected: reactive[bool] = reactive(False)
    daemon_running: reactive[bool] = reactive(False)
    project_count: reactive[int] = reactive(0)
    exclusion_count: reactive[int] = reactive(0)
    space_saved: reactive[int] = reactive(0)

    def compose(self) -> ComposeResult:
        with Horizontal(classes="filter-bar"):
            yield Label("[b]JART-STOW DASHBOARD[/b]")
        with Horizontal():
            with Container(classes="split-left"):
                yield StatusCard("System Status", id="status-card")
            with Container(classes="split-right"):
                yield StatusCard("Quick Stats", id="stats-card")
        yield StatusCard("Recent Activity", id="activity-card")
        yield DataTable(id="events-table")
        yield NavBar("r:Refresh  Esc:Back  q:Quit")

    def on_mount(self) -> None:
        self._load_data()
        table = self.query_one("#events-table", DataTable)
        table.add_columns("Time", "Type", "Details")

    def _load_data(self) -> None:
        self.run_worker(self._fetch_all(), exclusive=True, name="dashboard-load")

    async def _fetch_all(self) -> None:
        client = self.app.client
        if not client:
            return
        try:
            health, daemon, projects, exclusions, events = await asyncio.gather(
                client.get_health(),
                client.get_daemon_status(),
                client.get_projects(),
                client.get_exclusions(),
                client.get_daemon_events(limit=20),
                return_exceptions=True,
            )

            if not isinstance(health, BaseException):
                self.api_connected = True
                self.daemon_running = health.get("daemon_running", False)
                status_card = self.query_one("#status-card", StatusCard)
                lines = [
                    f"API: {'[green]Connected[/]' if self.api_connected else '[red]Disconnected[/]'}",
                    f"Daemon: {'[green]Running[/]' if self.daemon_running else '[red]Stopped[/]'}",
                    f"Database: {'[green]OK[/]' if health.get('database_connected') else '[red]Error[/]'}",
                    f"Watch Root: {health.get('watched_root', 'None')}",
                ]
                status_card.set_body("\n".join(lines))

            if not isinstance(daemon, BaseException):
                self.daemon_running = daemon.get("running", False)

            if not isinstance(projects, BaseException):
                self.project_count = projects.get("total", len(projects.get("projects", [])))

            if not isinstance(exclusions, BaseException):
                self.exclusion_count = exclusions.get("total", len(exclusions.get("exclusions", [])))

            if not isinstance(health, BaseException) or not isinstance(exclusions, BaseException):
                stats_card = self.query_one("#stats-card", StatusCard)
                stats_card.set_body(
                    f"Projects: {self.project_count}\n"
                    f"Exclusions: {self.exclusion_count}\n"
                    f"Space Saved: {_fmt_bytes(self.space_saved)}"
                )

            if not isinstance(events, BaseException):
                table = self.query_one("#events-table", DataTable)
                table.clear()
                for ev in events.get("events", []):
                    created = ev.get("created_at", "")[:19]
                    etype = ev.get("event_type", "")
                    details = ev.get("details") or ev.get("folder_path") or ""
                    table.add_row(created, etype, details)

        except Exception as exc:
            self.notify(f"Error loading dashboard: {exc}", severity="error")
            self.api_connected = False

    def action_go_back(self) -> None:
        self.app.pop_screen()

    def key_r(self) -> None:
        self._load_data()
