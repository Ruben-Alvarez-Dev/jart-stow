"""Jart-Stow Textual TUI application."""

from __future__ import annotations

import asyncio

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Container, VerticalScroll
from textual.reactive import reactive
from textual.widgets import Footer, Header, Label, Static

from tui.client import JartStowClient
from tui.screens.audit import AuditScreen
from tui.screens.dashboard import DashboardScreen
from tui.screens.exclusions import ExclusionsScreen
from tui.screens.hygiene import HygieneScreen
from tui.screens.report import ReportScreen
from tui.screens.rules import RulesScreen
from tui.widgets.nav_bar import NavBar


MENU_ITEMS = [
    ("1", "Dashboard", "System status, quick stats, and recent activity"),
    ("2", "Exclusions", "View and manage backup exclusions"),
    ("3", "Hygiene", "Detect and review system junk for cleanup"),
    ("4", "Rules", "Manage hygiene and exclusion rules"),
    ("5", "Audit", "Verify exclusion consistency and project health"),
    ("6", "Report", "Generate hygiene and exclusion reports"),
]

SCREEN_MAP = {
    "1": "dashboard",
    "2": "exclusions",
    "3": "hygiene",
    "4": "rules",
    "5": "audit",
    "6": "report",
}


class MainMenuScreen(Static):
    """Main menu with card-based navigation."""

    selected: reactive[int] = reactive(0)

    def compose(self) -> ComposeResult:
        with VerticalScroll(id="menu-list"):
            for i, (_, title, desc) in enumerate(MENU_ITEMS):
                card = Container(
                    Label(f"[b]{title}[/b]", classes="menu-title"),
                    Label(desc, classes="menu-desc"),
                    classes="menu-card highlighted" if i == 0 else "menu-card",
                    id=f"menu-{i}",
                )
                yield card

    def on_mount(self) -> None:
        self._highlight()

    def watch_selected(self, _old: int, _new: int) -> None:
        self._highlight()

    def _highlight(self) -> None:
        for i in range(len(MENU_ITEMS)):
            try:
                card = self.query_one(f"#menu-{i}", Container)
            except Exception:
                continue
            card.set_class(i == self.selected, "highlighted")
        try:
            self.query_one(f"#menu-{self.selected}", Container).focus()
        except Exception:
            pass

    def action_up(self) -> None:
        self.selected = (self.selected - 1) % len(MENU_ITEMS)

    def action_down(self) -> None:
        self.selected = (self.selected + 1) % len(MENU_ITEMS)

    def action_select(self) -> None:
        key = MENU_ITEMS[self.selected][0]
        screen_name = SCREEN_MAP[key]
        self.app.push_screen(screen_name)


class JartStowTUI(App):
    """Jart-Stow Terminal User Interface."""

    TITLE = "Jart-Stow"
    CSS_PATH = "styles.tcss"

    BINDINGS = [
        Binding("q", "quit", "Quit", show=True),
        Binding("escape", "go_back", "Back", show=True),
    ]

    SCREENS = {
        "dashboard": DashboardScreen,
        "exclusions": ExclusionsScreen,
        "hygiene": HygieneScreen,
        "rules": RulesScreen,
        "audit": AuditScreen,
        "report": ReportScreen,
    }

    client: JartStowClient | None = None

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield MainMenuScreen()
        yield NavBar("1-6: Navigate  Enter: Select  Esc: Back  q: Quit")
        yield Footer()

    def on_mount(self) -> None:
        self.client = JartStowClient()

    async def on_unmount(self) -> None:
        if self.client:
            await self.client.close()

    def action_go_back(self) -> None:
        if isinstance(self.screen, MainMenuScreen) or self.screen is self:
            return
        self.pop_screen()
