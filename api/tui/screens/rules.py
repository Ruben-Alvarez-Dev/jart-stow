"""Rules screen — global and project-scoped hygiene rules."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Container, VerticalScroll
from textual.screen import Screen
from textual.widgets import DataTable, Label

from tui.widgets.nav_bar import NavBar


class RulesScreen(Screen):

    BINDINGS = [
        Binding("tab", "switch_table", "Switch"),
        Binding("escape", "go_back", "Back"),
    ]

    _focused: str = "global"

    def compose(self) -> ComposeResult:
        yield Label("[b]HYGIENE RULES[/b]", classes="card-title")
        with VerticalScroll():
            yield Label("[b]Global Defaults[/b]")
            yield DataTable(id="global-table")
            yield Label("[b]Project Overrides[/b]", id="project-label")
            yield DataTable(id="project-table")
        yield NavBar("Tab:Switch  r:Reload  Esc:Back  q:Quit")

    def on_mount(self) -> None:
        for tid in ("global-table", "project-table"):
            t = self.query_one(f"#{tid}", DataTable)
            t.add_columns("ID", "Pattern", "Max Size", "Action", "Priority", "Enabled")
            t.cursor_type = "row"
        self._load_data()
        self.query_one("#global-table", DataTable).focus()

    def _load_data(self) -> None:
        self.run_worker(self._fetch_rules(), exclusive=True)

    async def _fetch_rules(self) -> None:
        client = self.app.client
        if not client:
            return
        try:
            result = await client.get_rules(enabled_only=False)
            rules = result.get("rules", [])

            global_table = self.query_one("#global-table", DataTable)
            project_table = self.query_one("#project-table", DataTable)
            global_table.clear()
            project_table.clear()

            for rule in rules:
                row = (
                    rule["id"],
                    rule["pattern"],
                    str(rule["max_size_bytes"]),
                    rule["action"],
                    str(rule["priority"]),
                    "[green]on[/]" if rule["enabled"] else "[red]off[/]",
                )
                if rule.get("project_id") is None:
                    global_table.add_row(*row)
                else:
                    project_table.add_row(
                        rule["id"],
                        rule.get("project_name", str(rule["project_id"])),
                        rule["pattern"],
                        rule["action"],
                        str(rule["priority"]),
                        "[green]on[/]" if rule["enabled"] else "[red]off[/]",
                    )
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def action_switch_table(self) -> None:
        if self._focused == "global":
            self.query_one("#project-table", DataTable).focus()
            self._focused = "project"
        else:
            self.query_one("#global-table", DataTable).focus()
            self._focused = "global"

    def key_r(self) -> None:
        self._load_data()

    def action_go_back(self) -> None:
        self.app.pop_screen()
