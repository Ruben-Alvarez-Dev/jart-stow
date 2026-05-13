// Package tui — Main Menu screen.
// Now uses components.Screen for viewport-safe layout.
package tui

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/components"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem struct {
	icon  string
	title string
	desc  string
	build func() tea.Model
}

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

type MainMenuModel struct {
	theme      *theme.Theme
	items      []menuItem
	cursor     int
	width      int
	height     int
	navRequest string
	navTarget  tea.Model
}

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

func (m *MainMenuModel) Init() tea.Cmd { return nil }

func (m *MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.navRequest = "quit"
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
			}
		case "1", "2", "3", "4", "5", "6", "7":
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(m.items) {
				m.navRequest = "forward"
				m.navTarget = m.items[idx].build()
			}
		}

	case tea.MouseMsg:
		if msg.Type == tea.MouseLeft {
			m.handleMouseClick(msg.X, msg.Y)
		}
	}
	return m, nil
}

func (m *MainMenuModel) handleMouseClick(x, y int) {
	startY := 6
	itemH := 4
	for i := range m.items {
		top := startY + i*itemH
		bot := top + itemH
		if y >= top && y < bot && x >= 2 && x < m.width-2 {
			m.cursor = i
			m.navRequest = "forward"
			m.navTarget = m.items[i].build()
		}
	}
}

func (m *MainMenuModel) NavRequest() string   { return m.navRequest }
func (m *MainMenuModel) NavTarget() tea.Model { return m.navTarget }
func (m *MainMenuModel) ClearNav()            { m.navRequest = ""; m.navTarget = nil }

func (m *MainMenuModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	minW := 50
	if m.width < minW {
		return "Terminal too narrow (" + itoa(m.width) + " < " + itoa(minW) + ")"
	}

	// Use Screen component for viewport-safe layout
	s := components.NewScreen(m.theme, m.width, m.height, "JART-STOW",
		"macOS development hygiene & backup exclusion manager")

	// Each item is 4 lines. Calculate how many fit in available content height.
	// ContentHeight includes everything between header and nav bar.
	navTH := "↑↓ select · ↵ enter · 1-7 quick · q quit · click"
	navH := lipgloss.Height(m.theme.NavBar.Width(m.width).Padding(0, 1).Render(navTH))
	availH := s.ContentHeight() - navH // subtract nav hint since we include it manually
	maxItems := availH / 4
	if maxItems < 1 {
		maxItems = 1
	}
	if maxItems > len(m.items) {
		maxItems = len(m.items)
	}

	// Build only the items that fit
	var itemViews []string
	for i := 0; i < maxItems; i++ {
		itemViews = append(itemViews, m.renderItem(i))
	}

	// If some items don't fit, note it
	if maxItems < len(m.items) {
		more := m.theme.Muted.Render("  ... and " + itoa(len(m.items)-maxItems) + " more options")
		itemViews = append(itemViews, more)
	}

	panels := lipgloss.JoinVertical(lipgloss.Left, itemViews...)
	return s.Render(panels, "", navTH)
}

func (m *MainMenuModel) renderHeader() string {
	title := m.theme.Title.Padding(0, 3).Render("JART-STOW")
	divider := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Width(m.width).
		Render(repeatStr("─", m.width))
	subtitle := m.theme.Muted.
		Padding(0, 0).
		Width(m.width).
		Render(lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			"macOS development hygiene & backup exclusion manager"))
	return lipgloss.JoinVertical(lipgloss.Center, title, divider, subtitle, "")
}

func (m *MainMenuModel) renderItem(i int) string {
	item := m.items[i]
	selected := i == m.cursor
	cardW := m.width - 4
	if cardW > 68 {
		cardW = 68
	}
	cardW = min(cardW, m.width-4)
	innerW := cardW - 4

	icon := m.theme.Primary.Render(item.icon)
	desc := item.desc

	var titleLine, descLine string
	if selected {
		titleLine = m.theme.SelectedItem.Width(innerW).Render(" ▸ " + icon + "  " + item.title)
		descLine = m.theme.SelectedItem.Width(innerW).Render("    " + desc)
	} else {
		t1 := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true)
		titleLine = lipgloss.NewStyle().Width(innerW).Render("   " + icon + "  " + t1.Render(item.title))
		descLine = m.theme.Muted.Width(innerW).Render("    " + desc)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine)

	var border lipgloss.Color
	if selected {
		border = theme.ColorPrimary
	} else {
		border = theme.ColorMuted
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(cardW).
		Render(inner)

	return lipgloss.NewStyle().Width(m.width).Render(
		lipgloss.PlaceHorizontal(m.width, lipgloss.Center, card),
	)
}

func repeatStr(s string, count int) string {
	if count <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
