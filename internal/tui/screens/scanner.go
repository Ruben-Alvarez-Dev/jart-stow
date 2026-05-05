package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Scanner Screen
// ============================================================================

// ScannerModel displays the project scanner screen with volume/path selection
// and scan results.
type ScannerModel struct {
	theme      *theme.Theme
	watchRoots WatchRootProvider

	width  int
	height int

	// Focus: left panel or right panel
	focus int // 0 = left, 1 = right

	// Selected watch root index (in the left panel)
	selectedPath int
}

// NewScannerModel creates a new ScannerModel.
func NewScannerModel(t *theme.Theme, watchRoots WatchRootProvider) *ScannerModel {
	return &ScannerModel{
		theme:      t,
		watchRoots: watchRoots,
		focus:      0,
	}
}

// Init initializes the scanner screen.
func (m *ScannerModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the scanner screen.
func (m *ScannerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 2
		case "up", "k":
			if m.focus == 0 && m.selectedPath > 0 {
				m.selectedPath--
			}
		case "down", "j":
			if m.focus == 0 {
				m.selectedPath++
			}
		}
	}
	return m, nil
}

// View renders the scanner layout.
func (m *ScannerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	panels := m.renderPanels()
	logPanel := m.renderLogPanel()
	navBar := m.renderNavBar()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		panels,
		logPanel,
	)

	mainArea := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 3).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, navBar)
}

func (m *ScannerModel) renderHeader() string {
	title := m.theme.Header.Render("SCANNER")
	return title
}

func (m *ScannerModel) renderPanels() string {
	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 6

	leftPanel := m.renderLeftPanel(leftWidth)
	rightPanel := m.renderRightPanel(rightWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m *ScannerModel) renderLeftPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Volumes & Paths")

	var lines []string
	lines = append(lines, "[ ] Home")
	lines = append(lines, "[ ] Macintosh HD")

	// Show configured watch roots
	if m.watchRoots != nil {
		roots, err := m.watchRoots.ListWatchRoots()
		if err == nil {
			for _, root := range roots {
				marker := "[ ]"
				lines = append(lines, marker+" "+theme.Truncate(root.Path, width-6))
			}
		}
	}

	if len(lines) == 0 {
		lines = append(lines, m.theme.Muted.Render("No watch roots configured"))
	}

	divider := m.theme.Muted.Render(stringOfChar('-', width-4))
	lines = append(lines, divider)
	lines = append(lines, "Patterns:")
	lines = append(lines, "[x] node_modules")
	lines = append(lines, "[x] .venv")
	lines = append(lines, "[x] target")
	lines = append(lines, "[x] build")

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	border := m.theme.CardBorder
	if m.focus == 0 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *ScannerModel) renderRightPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Scan Results")

	emptyMsg := m.theme.Muted.Render("  No scan results yet.\n  Select paths and press Enter to scan.")

	border := m.theme.CardBorder
	if m.focus == 1 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, emptyMsg))
}

func (m *ScannerModel) renderLogPanel() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Scan Log")
	emptyMsg := m.theme.Muted.Render("  No scans performed yet. Press Enter to start scanning.")

	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, emptyMsg))
}

func (m *ScannerModel) renderNavBar() string {
	return m.theme.HelpText.Render("Space: Toggle  |  Enter: Scan  |  Tab: Switch panels")
}

func stringOfChar(c byte, n int) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = c
	}
	return string(buf)
}
