// Package components provides reusable layout primitives that enforce viewport
// constraints. Every screen uses these instead of raw lipgloss, guaranteeing
// that no content overflows the terminal.
package components

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Screen represents the standard screen layout:
//
//	┌─ ScreenTitle ──────────────────────┐
//	│                                     │  header (fixed: 5 lines)
//	├─────────────────────────────────────┤
//	│   panels / main content             │  panels (variable)
//	├─────────────────────────────────────┤
//	│   action bar (optional)             │  actions (0 or 5 lines)
//	├─────────────────────────────────────┤
//	└─ nav bar ───────────────────────────┘  nav (1-2 lines)
//
// Total height never exceeds viewportHeight - enforced by MaxHeight().
type Screen struct {
	theme          *theme.Theme
	viewportWidth  int
	viewportHeight int
	title          string
	subtitle       string
}

// NewScreen creates a screen layout component.
func NewScreen(t *theme.Theme, width, height int, title, subtitle string) *Screen {
	return &Screen{
		theme:          t,
		viewportWidth:  width,
		viewportHeight: height,
		title:          title,
		subtitle:       subtitle,
	}
}

// Render assembles the full screen: header + panels + actions + navBar.
// All sections are rendered within viewportHeight.
func (s *Screen) Render(panels, actions, navText string) string {
	header := s.renderHeader()
	navBar := s.renderNavBar(navText)

	headerH := lipgloss.Height(header)
	navH := lipgloss.Height(navBar)
	actionsH := lipgloss.Height(actions)

	// Available height for panels = remaining after fixed elements
	panelsMaxH := s.viewportHeight - headerH - actionsH - navH
	if panelsMaxH < 4 {
		panelsMaxH = 4
	}

	// Clip panels to available space
	if lipgloss.Height(panels) > panelsMaxH {
		panels = lipgloss.NewStyle().MaxHeight(panelsMaxH).Render(panels)
	}

	// Build and enforce total ≤ viewport
	full := lipgloss.JoinVertical(lipgloss.Left, header, panels, actions, navBar)
	return lipgloss.NewStyle().MaxHeight(s.viewportHeight).Render(full)
}

// ContentHeight returns available lines for ALL panels content
// (after fixed header and nav bar, before clipping).
func (s *Screen) ContentHeight() int {
	headerH := 5 // ScreenTitle(border+margin) + subtitle + blank
	navH := lipgloss.Height(s.renderNavBar(""))
	return s.viewportHeight - headerH - navH
}

// PanelHeight returns the exact number of lines available for the
// panels area, accounting for an optional action bar.
// This is ContentHeight minus actions height.
func (s *Screen) PanelHeight(actionsH int) int {
	ch := s.ContentHeight() - actionsH
	if ch < 4 {
		ch = 4
	}
	return ch
}

// CardContentHeight returns lines available INSIDE a Card component,
// after card chrome (5 lines: top border, top padding, title,
// bottom padding, bottom border). Use this to size table rows / list items.
// actionsH = 0 if there's no action bar, or 5 if there is one.
func (s *Screen) CardContentHeight(actionsH int) int {
	ph := s.PanelHeight(actionsH)
	// Card: border_top(1) + pad_top(1) + title(1) + pad_bot(1) + border_bot(1) = 5
	ch := ph - 5
	if ch < 1 {
		ch = 1
	}
	return ch
}

// ── Header ────────────────────────────────────────────────────────────────

func (s *Screen) renderHeader() string {
	title := s.theme.ScreenTitle.Render(s.title)
	subtitle := s.theme.Muted.Render(s.subtitle)
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "")
}

func (s *Screen) renderNavBar(navText string) string {
	if navText == "" {
		navText = "Esc: Menu | q: Quit"
	}
	return s.theme.NavBar.Width(s.viewportWidth).Padding(0, 1).Render(navText)
}

// ── Card ──────────────────────────────────────────────────────────────────

// Card wraps content in a themed card with border and title.
type Card struct {
	theme *theme.Theme
	title string
	width int
}

// NewCard creates a card with the given title and width.
func NewCard(t *theme.Theme, title string, width int) *Card {
	return &Card{theme: t, title: title, width: width}
}

// Render renders content inside the card with consistent sizing.
func (c *Card) Render(content string) string {
	full := lipgloss.JoinVertical(lipgloss.Left,
		c.theme.CardTitle.Render(c.title),
		content,
	)
	return c.theme.CardBorder.Width(c.width).Render(full)
}

// ── ActionBar ─────────────────────────────────────────────────────────────

// ActionBar renders a horizontal action hint bar using card styling.
// Returns "" if actionText is empty. When non-empty, height is always 5 lines.
func ActionBar(t *theme.Theme, width int, actionText string) string {
	if actionText == "" {
		return ""
	}
	w := width - 2
	if w < 10 {
		w = 10
	}
	return t.CardBorder.Width(w).Render(t.Primary.Render(actionText))
}

// ActionBarHeight returns the height of an action bar (0 or 5).
func ActionBarHeight(actionText string) int {
	if actionText == "" {
		return 0
	}
	return 5
}
