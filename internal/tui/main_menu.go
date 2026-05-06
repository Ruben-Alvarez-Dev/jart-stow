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
	// Menu items start after header (approx 5 lines for header + spacing)
	startY := 6
	itemHeight := 4 // card: border top + title + desc + border bottom
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

// View renders the main menu with a professional card-based layout.
func (m *MainMenuModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	minWidth := 50
	if m.width < minWidth {
		return "Terminal too narrow (" + itoa(m.width) + " < " + itoa(minWidth) + ")"
	}

	header := m.renderHeader()

	// Render menu item cards
	var items []string
	for i, item := range m.items {
		items = append(items, m.renderItem(i, item))
	}
	menuArea := lipgloss.JoinVertical(lipgloss.Left, items...)

	// Spacer to push footer to bottom
	footer := m.renderFooter()
	usedHeight := lipgloss.Height(header) + lipgloss.Height(menuArea) + lipgloss.Height(footer)
	remainingHeight := m.height - usedHeight
	var spacer string
	if remainingHeight > 0 {
		spacer = lipgloss.NewStyle().Height(remainingHeight).Render("")
	}

	return lipgloss.JoinVertical(lipgloss.Top,
		header,
		menuArea,
		spacer,
		footer,
	)
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
		Render(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "macOS development hygiene & backup exclusion manager"))

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		divider,
		subtitle,
		"",
	)
}

func (m *MainMenuModel) renderItem(i int, item menuItem) string {
	selected := i == m.cursor

	// Determine card width: min(68, width-4)
	cardW := m.width - 4
	if cardW > 68 {
		cardW = 68
	}

	// Icon + title line
	iconStyled := m.theme.Primary.Render(item.icon)
	titleText := item.title
	descText := item.desc

	var titleLine, descLine string
	innerW := cardW - 4 // inside borders

	if selected {
		// Selected: highlighted background with ♢ marker, bold title
		titleLine = m.theme.SelectedItem.
			Width(innerW).
			Render(" ▸ " + iconStyled + "  " + titleText)
		descLine = m.theme.SelectedItem.
			Width(innerW).
			Render("    " + descText)
	} else {
		// Normal: cyan title on dark background, muted description
		normalTitle := lipgloss.NewStyle().
			Foreground(theme.ColorPrimary).
			Bold(true)

		titleLine = lipgloss.NewStyle().
			Width(innerW).
			Render("   " + iconStyled + "  " + normalTitle.Render(titleText))
		descLine = m.theme.Muted.
			Width(innerW).
			Render("    " + descText)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine)

	// Card border: highlighted cyan border for selected, muted for others
	var cardStyle lipgloss.Style
	if selected {
		cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorPrimary).
			Padding(0, 1).
			Width(cardW)
	} else {
		cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorMuted).
			Padding(0, 1).
			Width(cardW)
	}

	// Center the card horizontally
	return lipgloss.NewStyle().Width(m.width).Render(
		lipgloss.PlaceHorizontal(m.width, lipgloss.Center, cardStyle.Render(inner)),
	)
}

func (m *MainMenuModel) renderFooter() string {
	hint := m.theme.HelpText.Render("↑↓ select  ·  ↵ enter  ·  1-7 quick  ·  q quit  ·  click")
	return m.theme.NavBar.Width(m.width).Padding(0, 1).Render(hint)
}

// repeatStr creates a string of repeated characters.
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

// itoa converts int to string without importing fmt.
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
