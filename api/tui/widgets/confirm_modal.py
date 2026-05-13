"""Confirmation modal for destructive actions."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Container, Horizontal
from textual.screen import ModalScreen
from textual.widgets import Button, Label


class ConfirmModal(ModalScreen[bool]):
    """Yes/No confirmation dialog."""

    DEFAULT_CSS = """
    ConfirmModal {
        align: center middle;
    }
    ConfirmModal > Container {
        width: 60;
        height: auto;
        padding: 2 4;
        border: thick $error;
        background: $surface;
    }
    ConfirmModal > Container > Label {
        margin-bottom: 2;
    }
    ConfirmModal > Horizontal {
        height: auto;
        align: center middle;
    }
    ConfirmModal Button {
        margin: 0 2;
    }
    """

    def __init__(self, message: str, **kwargs) -> None:
        super().__init__(**kwargs)
        self.message = message

    def compose(self) -> ComposeResult:
        with Container():
            yield Label(self.message)
            with Horizontal():
                yield Button("Yes", variant="error", id="confirm-yes")
                yield Button("No", variant="primary", id="confirm-no")

    def on_button_pressed(self, event: Button.Pressed) -> None:
        self.dismiss(event.button.id == "confirm-yes")

    def on_key(self, event) -> None:
        if event.key == "escape":
            self.dismiss(False)
