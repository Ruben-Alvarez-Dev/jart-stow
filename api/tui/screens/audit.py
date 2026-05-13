"""Audit screen — project tree with status icons and detail panel."""

from __future__ import annotations

from collections import defaultdict

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Container, Horizontal
from textual.screen import Screen
from textual.widgets import Label, Static, Tree

from tui.widgets.nav_bar import NavBar


STATUS_ICONS = {
    "active": "[green]●[/]",
    "archived": "[dim]◉[/]",
    "ignored": "[yellow]▲[/]",
}


class AuditScreen(Screen):

    BINDINGS = [
        Binding("escape", "go_back", "Back"),
    ]

    def compose(self) -> ComposeResult:
        with Horizontal(classes="filter-bar"):
            yield Label("[b]PROJECT AUDIT[/b]")
        with Horizontal():
            with Container(classes="split-left"):
                yield Label("[b]Projects[/b]")
                yield Tree("All Projects", id="project-tree")
            with Container(classes="split-right"):
                yield Label("[b]Project Details[/b]")
                yield Static(id="project-detail")
        yield NavBar("Enter:Select  r:Reload  Esc:Back  q:Quit")

    def on_mount(self) -> None:
        self._load_data()

    def _load_data(self) -> None:
        self.run_worker(self._fetch_projects(), exclusive=True)

    async def _fetch_projects(self) -> None:
        client = self.app.client
        if not client:
            return
        try:
            result = await client.get_projects()
            projects = result.get("projects", [])

            tree = self.query_one("#project-tree", Tree)
            tree.root.remove_children()

            by_root: dict[str, list[dict]] = defaultdict(list)
            for p in projects:
                by_root[p["root_path"]].append(p)

            for root_path, projs in sorted(by_root.items()):
                branch = tree.root.add(root_path, expand=True)
                for p in projs:
                    icon = STATUS_ICONS.get(p["status"], "?")
                    branch.add_leaf(f"{icon} {p['name']}", data=p)

            tree.root.expand()
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def on_tree_node_selected(self, event: Tree.NodeSelected) -> None:
        node = event.node
        if node.data and isinstance(node.data, dict):
            p = node.data
            self.run_worker(self._load_detail(p["id"]), exclusive=True)

    async def _load_detail(self, project_id: int) -> None:
        client = self.app.client
        if not client:
            return
        try:
            detail = await client.get_project(project_id)
            detail_widget = self.query_one("#project-detail", Static)
            status_icon = STATUS_ICONS.get(detail["status"], "?")
            last_scanned = detail.get("last_scanned", "Never") or "Never"

            def _fmt_bytes(n: int) -> str:
                for unit in ("B", "KB", "MB", "GB", "TB"):
                    if abs(n) < 1024:
                        return f"{n:.1f} {unit}"
                    n /= 1024
                return f"{n:.1f} PB"

            detail_widget.update(
                f"Name: {detail['name']}\n"
                f"Path: {detail['path']}\n"
                f"Status: {status_icon} {detail['status']}\n"
                f"Root: {detail['root_path']}\n"
                f"Last Scanned: {last_scanned}\n"
                f"Exclusions: {detail.get('exclusion_count', 'N/A')}\n"
                f"Total Excluded: {_fmt_bytes(detail.get('total_excluded_size_bytes', 0))}\n"
                f"Created: {detail.get('created_at', 'N/A')}\n"
                f"Updated: {detail.get('updated_at', 'N/A')}"
            )
        except Exception as exc:
            self.notify(f"Error: {exc}", severity="error")

    def key_r(self) -> None:
        self._load_data()

    def action_go_back(self) -> None:
        self.app.pop_screen()
