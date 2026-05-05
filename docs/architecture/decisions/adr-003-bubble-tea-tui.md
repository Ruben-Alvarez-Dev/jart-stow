# ADR-003: Bubble Tea for Terminal UI

**Status:** Accepted
**Date:** 2026-05-05
**Deciders:** Ruben Alvarez

## Context

Jart-Stow requires a professional, high-performance terminal UI with real-time updates, keyboard-driven navigation, and a polished visual design. The user explicitly required the Charmbracelet ecosystem.

## Decision

Use Bubble Tea + Bubbles + Lipgloss (the Charmbracelet ecosystem) for the TUI layer. All 7 screens are implemented as Bubble Tea models.

## Consequences

**Positive:**
- Elm-like architecture ensures predictable state management.
- Bubbles provides battle-tested components (table, viewport, spinner).
- Lipgloss enables declarative styling without ANSI escape manipulation.
- Harmonica enables smooth animations for transitions.
- The ecosystem is actively maintained and widely adopted.

**Negative:**
- Go-only. Cannot share UI code with the Python API layer.
- Bubble Tea's component model is simple but less flexible than immediate-mode TUIs.

## Alternatives Considered

### Textual (Python)
Rejected. While powerful, Textual would require the TUI to live in the Python runtime, separating it from the daemon's real-time FSEvents stream. The user explicitly chose the Charmbracelet ecosystem.

### Rich (Python)
Rejected as a display library, not a TUI framework. Lacks state management and keyboard navigation primitives.

### tview (Go)
Rejected. Less actively maintained and lacks the component ecosystem of Bubble Tea + Bubbles.
