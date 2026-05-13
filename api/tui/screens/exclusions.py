"""Exclusions screen — interactive table with filtering and sorting."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Container, Horizontal
from textual.reactive import reactive
from textual.screen import Screen
from textual.widgets import Button, DataTable, Label, Static

from tui.widgets.confirm_modal import ConfirmModal
from tui.widgets.nav_bar import NavBar


SYSTEM_FILTERS = ["all", "time_machine", "carbon_copy_cloner"]
SORT_OPTIONS = ["applied_at", "size_bytes", "pattern_matched"]


def _fmt_bytes(n: int) -> str:
    for unit in ("B", "KB", "MB", "GB", "TB"):
        if abs(n) < 1024:
            return f"{n:.1f} {unit}"
        n /= 1024
    return f"{n:.1f} PB"


class ExclusionsScreen(Screen):

    BINDINGS = [
        Binding("f", "cycle_filter", "Filter"),
        Binding("s", "cycle_sort", "Sort"),
        Binding("r", "remove_selected", "Remove"),
        Binding("escape", "go_back", "Back"),
    ]

    filter_idx: reactive[int] = reactive(0)
    sort_idx: reactive[int] = reactive(0)

    def compose(self) -> ComposeResult:
        with Horizontal(classes="filter-bar"):
            yield Label("[b]EXCLUSIONS[/b]", id="title")
            yield Label(id="filter-label")
            yield Label(id="sort-label")
        yield DataTable(id="exclusions-table")
        yield Static(id="detail-panel", classes="detail-panel")
        yield NavBar("f:Filter  s:Sort  r:Remove  Enter:Reload  Esc:Back  q:Quit")

    def on_mount(self) -> None:
        table = self.query_one("#exclusions-table", DataTable)
        table.add_columns("ID", "Path", "Pattern", "Size", "System", "Status")
        table.cursor_type = "row"
        self._update_labels()
        self._load_data()

    def _update_labels(self) -> None:
        fl = self.query_one("#filter-label", Label)
        sl = self.query_one("#sort-label", Label)
        fl.update(f"Filter: [bold]{SYSTEM_FILTERS[self.filter_idx]}[/bold]")
        sl.update(f"Sort: [bold]{SORT_OPTIONS[self.sort_idx]}[/bold]")

    def watch_filter_idx(self, *_: object) -> None:
        self._update_labels()
        self._load_data()

    def watch_sort_idx(self, *_: object) -> None:
        self._update_labels()
        self._load_data()

    def _load_data(self) -> None:
        self.run_worker(self._fetch_exclusions(), exclusive=True)

    async def _fetch_exclusions(self) -> None:
        client = self.app.client
        if not client:
            return
        try:
            system = SYSTEM_FILTERS[self.filter_idx]
            bs = None if system == "all" else system
            result = await client.get_exclusions(
                backup_system=bs,
                sort_by=SORT_OPTIONS[self.sort_idx],
                order="desc",
            )
            table = self.query_one("#exclusions-table", DataTable)
            table.clear()
            for ex in result.get("exclusions", []):
                table.add_row(
                    ex["id"],
                    ex["folder_path"],
                    ex["pattern_matched"],
                    _fmt_bytes(ex["size_bytes"]),
                    ex["backup_system"],
                    "active" if ex.get("removed_at") is None else "removed",
                )
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def on_data_table_row_selected(self, event: DataTable.RowSelected) -> None:
        table = self.query_one("#exclusions-table", DataTable)
        if event.row_index is not None and event.row_index < len(table.rows):
            row_data = table.get_row_at(event.row_index)
            detail = self.query_one("#detail-panel", Static)
            detail.update(
                f"ID: {row_data[0]}  |  Path: {row_data[1]}\n"
                f"Pattern: {row_data[2]}  |  Size: {row_data[3]}  |  System: {row_data[4]}"
            )

    def action_cycle_filter(self) -> None:
        self.filter_idx = (self.filter_idx + 1) % len(SYSTEM_FILTERS)

    def action_cycle_sort(self) -> None:
        self.sort_idx = (self.sort_idx + 1) % len(SORT_OPTIONS)

    def action_remove_selected(self) -> None:
        table = self.query_one("#exclusions-table", DataTable)
        if table.cursor_row is not None and table.cursor_row < len(table.rows):
            row_data = table.get_row_at(table.cursor_row)
            exc_id = row_data[0]

            async def on_confirm(confirmed: bool) -> None:
                if confirmed and self.app.client:
                    try:
                        await self.app.client.delete_exclusion(exc_id)
                        self.notify(f"Exclusion {exc_id} removed")
                        self._load_data()
                    except Exception as exc:
                        self.notify(f"Error: {exc}", severity="error")

            self.app.push_screen(
                ConfirmModal(f"Remove exclusion #{exc_id}?"),
                on_confirm,
            )

    def action_go_back(self) -> None:
        self.app.pop_screen()
