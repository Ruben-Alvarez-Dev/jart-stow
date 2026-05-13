"""Hygiene screen — junk categories and items with approve/skip actions."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Container, Horizontal
from textual.screen import Screen
from textual.widgets import DataTable, Label, OptionList, Static

from tui.widgets.nav_bar import NavBar


def _fmt_bytes(n: int) -> str:
    for unit in ("B", "KB", "MB", "GB", "TB"):
        if abs(n) < 1024:
            return f"{n:.1f} {unit}"
        n /= 1024
    return f"{n:.1f} PB"


class HygieneScreen(Screen):

    BINDINGS = [
        Binding("tab", "switch_panel", "Switch"),
        Binding("a", "approve_item", "Approve"),
        Binding("s", "skip_item", "Skip"),
        Binding("A", "approve_all", "Approve All"),
        Binding("escape", "go_back", "Back"),
    ]

    _categories: list[dict] = []
    _focused_panel: str = "categories"

    def compose(self) -> ComposeResult:
        with Horizontal(classes="filter-bar"):
            yield Label("[b]SYSTEM HYGIENE[/b]")
        with Horizontal():
            with Container(classes="split-left"):
                yield Label("[b]Categories[/b]")
                yield OptionList(id="categories-list")
            with Container(classes="split-right"):
                yield Label("[b]Junk Items[/b]")
                yield DataTable(id="items-table")
        yield Static(id="item-detail", classes="detail-panel")
        yield NavBar("Enter:Scan  a:Approve  s:Skip  A:Approve All  Tab:Switch  Esc:Back  q:Quit")

    def on_mount(self) -> None:
        table = self.query_one("#items-table", DataTable)
        table.add_columns("ID", "Path", "Description", "Size", "Status")
        table.cursor_type = "row"
        self._load_categories()
        self.query_one("#categories-list", OptionList).focus()

    def _load_categories(self) -> None:
        self.run_worker(self._fetch_categories(), exclusive=True)

    async def _fetch_categories(self) -> None:
        client = self.app.client
        if not client:
            return
        try:
            result = await client.get_junk_categories()
            self._categories = result.get("categories", [])
            ol = self.query_one("#categories-list", OptionList)
            ol.clear_options()
            for cat in self._categories:
                status = "[green]on[/]" if cat["enabled"] else "[red]off[/]"
                ol.add_option(f"{cat['name']} ({status})")
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def _load_items(self, category_id: int) -> None:
        self.run_worker(self._fetch_items(category_id), exclusive=True)

    async def _fetch_items(self, category_id: int) -> None:
        client = self.app.client
        if not client:
            return
        try:
            result = await client.get_junk_items(category_id=category_id)
            table = self.query_one("#items-table", DataTable)
            table.clear()
            for item in result.get("items", []):
                v = item.get("verified_by_user", 0)
                status = "approved" if v == 1 else "skipped" if v == -1 else "pending"
                table.add_row(
                    item["id"],
                    item["path"],
                    item["description"],
                    _fmt_bytes(item["size_bytes"]),
                    status,
                )
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def on_option_list_option_selected(self, event: OptionList.OptionSelected) -> None:
        idx = event.option_index
        if idx < len(self._categories):
            cat = self._categories[idx]
            self._load_items(cat["id"])
            self.run_worker(self._trigger_scan(cat["id"]))

    async def _trigger_scan(self, category_id: int) -> None:
        client = self.app.client
        if not client:
            return
        try:
            await client.trigger_junk_scan([category_id])
            self.notify("Scan triggered")
            self._load_items(category_id)
        except Exception as exc:
            self.notify(f"Scan error: {exc}", severity="error")

    def on_data_table_row_selected(self, event: DataTable.RowSelected) -> None:
        table = self.query_one("#items-table", DataTable)
        if event.row_index is not None and event.row_index < len(table.rows):
            row_data = table.get_row_at(event.row_index)
            detail = self.query_one("#item-detail", Static)
            detail.update(
                f"ID: {row_data[0]}  |  Path: {row_data[1]}\n"
                f"Description: {row_data[2]}  |  Size: {row_data[3]}  |  Status: {row_data[4]}"
            )

    def action_switch_panel(self) -> None:
        if self._focused_panel == "categories":
            self.query_one("#items-table", DataTable).focus()
            self._focused_panel = "items"
        else:
            self.query_one("#categories-list", OptionList).focus()
            self._focused_panel = "categories"

    def action_approve_item(self) -> None:
        self._update_selected_item(1)

    def action_skip_item(self) -> None:
        self._update_selected_item(-1)

    def _update_selected_item(self, verified: int) -> None:
        table = self.query_one("#items-table", DataTable)
        if table.cursor_row is not None and table.cursor_row < len(table.rows):
            row_data = table.get_row_at(table.cursor_row)
            item_id = row_data[0]
            self.run_worker(self._do_update(item_id, verified))

    async def _do_update(self, item_id: int, verified: int) -> None:
        client = self.app.client
        if not client:
            return
        try:
            await client.update_junk_item(item_id, verified)
            label = "approved" if verified == 1 else "skipped"
            self.notify(f"Item {item_id} {label}")
            if self._categories:
                self._load_items(self._categories[0]["id"])
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def action_approve_all(self) -> None:
        table = self.query_one("#items-table", DataTable)
        ids = []
        for i in range(len(table.rows)):
            row_data = table.get_row_at(i)
            if row_data[4] == "pending":
                ids.append(row_data[0])
        if ids:
            self.run_worker(self._do_batch(ids, 1))

    async def _do_batch(self, ids: list[int], verified: int) -> None:
        client = self.app.client
        if not client:
            return
        try:
            await client.batch_update_junk_items(ids, verified)
            self.notify(f"Batch updated {len(ids)} items")
            if self._categories:
                self._load_items(self._categories[0]["id"])
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def action_go_back(self) -> None:
        self.app.pop_screen()
