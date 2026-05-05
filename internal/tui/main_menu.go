// Package tui — Main Menu screen.
// This is the central hub of the TUI. All decision branches radiate from here.
package tui

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// menuItem represents one entry in the main menu.
type menuItem struct {
	icon  string
	title string
	desc  string
	build func() tea.Model
}

// mainMenuItems defines the decision tree from the main menu.
func mainMenuItems(buildDashboard, buildScanner, buildExclusions, buildRules, buildAudit, buildHygiene, buildReport func() tea.Model) []menuItem {
	return []menuItem{
		{"🏠", "Dashboard", "Overview: daemon, projects, space saved", buildDashboard},
		{"📂", "Scanner", "Scan projects for dev artifacts to exclude", buildScanner},
		{"🚫", "Exclusions", "View and manage backup exclusions", buildExclusions},
		{"🧹", "Hygiene", "Detect and review system junk for cleanup", buildHygiene},
		{"📋", "Rules", "Manage custom hygiene and exclusion rules", buildRules},
		{"🔍", "Audit", "Verify exclusion consistency and health", buildAudit},
		{"📊", "Report", "Generate hygiene and exclusion reports", buildReport},
	}
}

// MainMenuModel is the central hub screen of the TUI.
type MainMenuModel struct {
	theme *theme.Theme
	items []menuItem

	cursor int

	width  int
	height int

	navRequest string
	navTarget  tea.Model
}

// NewMainMenu creates the main menu screen.
func NewMainMenu(t *theme.Theme, builders ...func() tea.Model) *MainMenuModel {
	if len(builders) < 7 {
		panic("main menu requires 7 sub-screen builders")
	}
	return &MainMenuModel{
		theme: t,
		items: mainMenuItems(builders[0], builders[1], builders[2],
			builders[3], builders[4], builders[5], builders[6]),
	}
}

// Init initializes the main menu (no-op, static screen).
func (m *MainMenuModel) Init() tea.Cmd { return nil }

// Update handles keyboard and mouse input for the main menu.
func (m *MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.navRequest = "quit"
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", "right", "l":
			if m.cursor >= 0 && m.cursor < len(m.items) {
				m.navRequest = "forward"
				m.navTarget = m.items[m.cursor].build()
				return m, nil
			}
		case "1", "2", "3", "4", "5", "6", "7":
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(m.items) {
				m.navRequest = "forward"
				m.navTarget = m.items[idx].build()
				return m, nil
			}
		}

	case tea.MouseMsg:
		if msg.Type == tea.MouseLeft {
			m.handleMouseClick(msg.X, msg.Y)
		}
	}

	return m, nil
}

// handleMouseClick maps a click coordinate to a menu item.
func (m *MainMenuModel) handleMouseClick(x, y int) {
	// Menu items start after header (5 lines) and title area (2 lines)
	startY := 7
	itemHeight := 3 // each item: title + desc + spacing
	for i := range m.items {
		itemTop := startY + i*itemHeight
		itemBot := itemTop + itemHeight
		if y >= itemTop && y < itemBot && x >= 2 && x < m.width-2 {
			m.cursor = i
			m.navRequest = "forward"
			m.navTarget = m.items[i].build()
			return
		}
	}
}

// NavRequest returns the pending navigation request.
func (m *MainMenuModel) NavRequest() string   { return m.navRequest }
func (m *MainMenuModel) NavTarget() tea.Model { return m.navTarget }
func (m *MainMenuModel) ClearNav()            { m.navRequest = ""; m.navTarget = nil }

// View renders the main menu.
func (m *MainMenuModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Header
	header := m.renderHeader()

	// Menu items
	var itemViews []string
	for i, item := range m.items {
		itemViews = append(itemViews, m.renderItem(i, item))
	}
	menuArea := lipgloss.JoinVertical(lipgloss.Left, itemViews...)

	// Fill remaining space
	remainingHeight := m.height - lipgloss.Height(header) - lipgloss.Height(menuArea) - 3
	var spacer string
	if remainingHeight > 0 {
		spacer = lipgloss.NewStyle().Height(remainingHeight).Render("")
	}

	// Footer with controls
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Top,
		header,
		menuArea,
		spacer,
		footer,
	)
}

func (m *MainMenuModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorPrimary).
		Padding(0, 2).
		Render("JART-STOW")
	subtitle := m.theme.Muted.Render("macOS development hygiene & backup exclusion manager")

	header := lipgloss.JoinVertical(lipgloss.Center, title, subtitle)
	return m.theme.NavBar.Width(m.width).Render(header) + "\n"
}

func (m *MainMenuModel) renderItem(i int, item menuItem) string {
	const itemWidth = 60

	isSelected := i == m.cursor

	icon := item.icon + " "
	title := item.title
	desc := "  " + item.desc

	var line1, line2 string
	if isSelected {
		cursor := m.theme.Highlight.Render("▸")
		line1 = cursor + " " + m.theme.Selected.Render(icon+title)
		line2 = m.theme.Selected.Render(desc)
	} else {
		line1 = "  " + m.theme.Primary.Render(icon) + m.theme.Primary.Render(title)
		line2 = m.theme.Muted.Render(desc)
	}

	itemContent := lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	box := lipgloss.NewStyle().Padding(0, 4).Width(itemWidth).Render(itemContent)
	return box
}

func (m *MainMenuModel) renderFooter() string {
	hint := m.theme.HelpText.Render("↑↓ select  ↵ enter  q quit  mouse click")
	return m.theme.NavBar.Width(m.width).Render(hint)
}
