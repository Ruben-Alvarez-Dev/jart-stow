"""Report screen — summary statistics, breakdowns, and history."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Container, Horizontal, VerticalScroll
from textual.screen import Screen
from textual.widgets import DataTable, Label, Static

from tui.widgets.nav_bar import NavBar


def _fmt_bytes(n: int) -> str:
    for unit in ("B", "KB", "MB", "GB", "TB"):
        if abs(n) < 1024:
            return f"{n:.1f} {unit}"
        n /= 1024
    return f"{n:.1f} PB"


def _bar(pct: float, width: int = 30) -> str:
    filled = int(pct / 100 * width)
    return f"[on $primary]{' ' * filled}[/]{' ' * (width - filled)} {pct:.1f}%"


class ReportScreen(Screen):

    BINDINGS = [
        Binding("escape", "go_back", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Horizontal(classes="filter-bar"):
            yield Label("[b]REPORTS[/b]")
        with VerticalScroll():
            yield Static(id="summary-section")
            with Horizontal():
                with Container(classes="split-left"):
                    yield Label("[b]Breakdown by Pattern[/b]")
                    yield Static(id="pattern-breakdown")
                with Container(classes="split-right"):
                    yield Label("[b]Breakdown by System[/b]")
                    yield Static(id="system-breakdown")
            yield Label("[b]Exclusion History (30 days)[/b]")
            yield DataTable(id="history-table")
        yield NavBar("r:Refresh  Esc:Back  q:Quit")

    def on_mount(self) -> None:
        table = self.query_one("#history-table", DataTable)
        table.add_columns("Date", "Added", "Removed", "Size Added", "Size Removed")
        self._load_data()

    def _load_data(self) -> None:
        self.run_worker(self._fetch_all(), exclusive=True)

    async def _fetch_all(self) -> None:
        client = self.app.client
        if not client:
            return
        try:
            import asyncio
            summary, history = await asyncio.gather(
                client.get_report_summary(),
                client.get_report_history(days=30),
                return_exceptions=True,
            )

            if not isinstance(summary, BaseException):
                summary_widget = self.query_one("#summary-section", Static)
                summary_widget.update(
                    f"[b]Projects:[/b] {summary.get('projects_total', 0)} "
                    f"({summary.get('projects_active', 0)} active)\n"
                    f"[b]Active Exclusions:[/b] {summary.get('exclusions_active', 0)}\n"
                    f"[b]Total Space Excluded:[/b] {summary.get('exclusions_total_size_human', '0 B')}\n"
                )

                pattern_widget = self.query_one("#pattern-breakdown", Static)
                bp = summary.get("breakdown_by_pattern", {})
                total = max(sum(bp.values()), 1)
                lines = []
                for pattern, size in sorted(bp.items(), key=lambda x: -x[1]):
                    pct = size / total * 100
                    lines.append(f"{pattern[:20]:20} {_bar(pct)}")
                pattern_widget.update("\n".join(lines) or "No data")

                system_widget = self.query_one("#system-breakdown", Static)
                bs = summary.get("breakdown_by_system", {})
                total_s = max(sum(bs.values()), 1)
                slines = []
                for system, size in sorted(bs.items(), key=lambda x: -x[1]):
                    pct = size / total_s * 100
                    slines.append(f"{system[:20]:20} {_bar(pct)}")
                system_widget.update("\n".join(slines) or "No data")

            if not isinstance(history, BaseException):
                table = self.query_one("#history-table", DataTable)
                table.clear()
                for h in history.get("history", []):
                    table.add_row(
                        h["date"],
                        str(h["exclusions_added"]),
                        str(h["exclusions_removed"]),
                        _fmt_bytes(h["size_added_bytes"]),
                        _fmt_bytes(h["size_removed_bytes"]),
                    )

        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def key_r(self) -> None:
        self._load_data()

    def action_go_back(self) -> None:
        self.app.pop_screen()
