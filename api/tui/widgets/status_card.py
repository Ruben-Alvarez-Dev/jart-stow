"""Reusable status card widget."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Container
from textual.widgets import Label, Static


class StatusCard(Container):
    """Bordered card with a title and free-form content."""

    DEFAULT_CSS = """
    StatusCard {
        width: 100%;
        height: auto;
        padding: 0 1;
        margin: 0 0 1 0;
        border: round $primary;
        background: $surface;
    }
    StatusCard > .card-title {
        color: $primary;
        text-style: bold;
        margin-bottom: 1;
    }
    StatusCard > .card-body {
        height: auto;
    }
    """

    def __init__(self, title: str, **kwargs) -> None:
        super().__init__(**kwargs)
        self.card_title = title

    def compose(self) -> ComposeResult:
        yield Label(self.card_title, classes="card-title")
        yield Static(id="card-body", classes="card-body")

    def set_body(self, text: str) -> None:
        body = self.query_one("#card-body", Static)
        body.update(text)
