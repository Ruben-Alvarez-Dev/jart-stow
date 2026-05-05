// Package theme defines the visual styling system for the Jart-Stow TUI.
// It provides a Theme struct with lipgloss styles and helper functions
// that all screens use for consistent visual presentation.
package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Color constants used throughout the TUI.
const (
	ColorPrimary   = lipgloss.Color("6") // Cyan
	ColorSuccess   = lipgloss.Color("2") // Green
	ColorWarning   = lipgloss.Color("3") // Yellow
	ColorDanger    = lipgloss.Color("1") // Red
	ColorMuted     = lipgloss.Color("8") // Bright black (gray)
	ColorHighlight = lipgloss.Color("5") // Magenta
)

// Theme holds all the styled elements used across the TUI.
// Zero-allocation: create once at startup and reuse.
type Theme struct {
	Primary   lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Danger    lipgloss.Style
	Muted     lipgloss.Style
	Highlight lipgloss.Style

	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Header   lipgloss.Style

	StatusRunning lipgloss.Style
	StatusStopped lipgloss.Style

	CardBorder lipgloss.Style
	CardTitle  lipgloss.Style

	Selected  lipgloss.Style
	HelpText  lipgloss.Style
	ErrorText lipgloss.Style
}

// NewTheme creates a Theme with all styles initialized from the color palette.
func NewTheme() *Theme {
	t := &Theme{}

	// Base color styles (foreground-only, no background)
	t.Primary = lipgloss.NewStyle().Foreground(ColorPrimary)
	t.Success = lipgloss.NewStyle().Foreground(ColorSuccess)
	t.Warning = lipgloss.NewStyle().Foreground(ColorWarning)
	t.Danger = lipgloss.NewStyle().Foreground(ColorDanger)
	t.Muted = lipgloss.NewStyle().Foreground(ColorMuted)
	t.Highlight = lipgloss.NewStyle().Foreground(ColorHighlight)

	// Title: bold, primary foreground + background
	t.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(ColorPrimary).
		Padding(0, 1)

	// Subtitle: primary foreground, no bold
	t.Subtitle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(false)

	// Header: bold, primary foreground, underlined with muted color
	t.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorMuted).
		PaddingBottom(1).
		MarginBottom(1)

	// Status indicators
	t.StatusRunning = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)
	t.StatusStopped = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Card styles
	t.CardBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(1, 2)
	t.CardTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Padding(0, 1)

	// Selection / highlight
	t.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHighlight)

	// Help text: muted, small
	t.HelpText = lipgloss.NewStyle().
		Foreground(ColorMuted)

	// Error text: danger foreground, bold
	t.ErrorText = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	return t
}

// StatusDot returns a colored dot for running/stopped status.
// Green dot (running) or gray dot (stopped).
func (t *Theme) StatusDot(running bool) string {
	if running {
		return t.StatusRunning.Render("\u25CF") // ●
	}
	return t.StatusStopped.Render("\u25CB") // ○
}

// Truncate shortens a string to max characters, appending "..." if truncated.
// If max ≤ 0, the original string is returned unchanged.
func Truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
