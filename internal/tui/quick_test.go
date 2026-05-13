package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
)

func TestAllScreens(t *testing.T) {
	tm := theme.NewTheme()
	dummy := func() tea.Model { return &dummyM{} }
	
	// Test main menu at user's terminal size
	menu := NewMainMenu(tm, dummy, dummy, dummy, dummy, dummy, dummy, dummy)
	menu.Update(tea.WindowSizeMsg{Width: 112, Height: 32})
	view := menu.View()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	t.Logf("MAIN MENU 112x32: %d/32 lines", len(lines))
	for i, l := range lines {
		s := l
		if len(s) > 100 { s = s[:97] + "..." }
		mark := "✓"
		if i >= 32 { mark = "✗" }
		t.Logf("  %s %2d: %s", mark, i, s)
	}
	if len(lines) > 32 {
		t.Errorf("MAIN MENU OVERFLOW: %d lines", len(lines))
	}
}

type dummyM struct{}
func (d *dummyM) Init() tea.Cmd { return nil }
func (d *dummyM) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return d, nil }
func (d *dummyM) View() string { return "dummy" }
