"""Reusable bottom navigation bar widget."""

from __future__ import annotations

from textual.widgets import Static


class NavBar(Static):
    """A thin bar docked at the bottom showing keyboard shortcuts."""

    DEFAULT_CSS = """
    NavBar {
        dock: bottom;
        height: 1;
        background: $primary 20%;
        color: $text-muted;
        padding: 0 1;
        content-align: left middle;
    }
    """

    def __init__(self, hints: str = "Esc:Back  q:Quit", **kwargs) -> None:
        super().__init__(hints, **kwargs)

    def update_hints(self, hints: str) -> None:
        self.update(hints)
