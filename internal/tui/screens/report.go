package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Report Screen
// ============================================================================

// ReportModel displays aggregate statistics, space saved, and daemon event history.
type ReportModel struct {
	theme      *theme.Theme
	exclusions ExclusionLister
	events     EventLister

	width  int
	height int

	loaded      bool
	exclusionsList []domain.Exclusion
}

// NewReportModel creates a new ReportModel.
func NewReportModel(t *theme.Theme, exclusions ExclusionLister, events EventLister) *ReportModel {
	return &ReportModel{
		theme:      t,
		exclusions: exclusions,
		events:     events,
	}
}

// Init initializes the report screen.
func (m *ReportModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the report screen.
func (m *ReportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the report screen layout.
func (m *ReportModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	m.loadData()

	header := m.renderHeader()
	spaceSavedSection := m.renderSpaceSaved()
	panels := m.renderPanels()
	daemonSection := m.renderDaemonSection()
	navBar := m.renderNavBar()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		spaceSavedSection,
		panels,
		daemonSection,
	)

	mainArea := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 3).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, navBar)
}

func (m *ReportModel) loadData() {
	if m.loaded {
		return
	}
	m.loaded = true
}

func (m *ReportModel) renderHeader() string {
	title := m.theme.Header.Render("REPORT")
	return title
}

func (m *ReportModel) renderSpaceSaved() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Space Saved Over Time")

	var totalSpace int64
	if m.exclusions != nil {
		size, err := m.exclusions.TotalSpaceSaved()
		if err == nil {
			totalSpace = size
		}
	}

	if totalSpace == 0 && (m.exclusions == nil) {
		emptyMsg := m.theme.Muted.Render("  No data available. Run the daemon to collect statistics.")
		return m.theme.CardBorder.
			Width(cardWidth).
			Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, emptyMsg))
	}

	// Text-based sparkline bar
	bar := m.renderTextBar(totalSpace)
	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, bar))
}

func (m *ReportModel) renderTextBar(totalBytes int64) string {
	// Simple text-based bar showing space saved
	unit := "GB"
	val := float64(totalBytes) / (1024 * 1024 * 1024)
	if val < 0.01 && totalBytes > 0 {
		unit = "MB"
		val = float64(totalBytes) / (1024 * 1024)
	}
	if val < 0.01 && totalBytes > 0 {
		unit = "KB"
		val = float64(totalBytes) / 1024
	}

	// Build a proportional bar
	bar := m.theme.Primary.Render("Total Space Saved: " + formatBytes(totalBytes))
	bar += "\n"
	bar += m.renderHorizontalBar(totalBytes, totalBytes, 40)
	bar += "\n  " + formatFloat(val) + " " + unit

	return bar
}

func (m *ReportModel) renderHorizontalBar(value int64, max int64, width int) string {
	if max <= 0 {
		max = 1
	}
	filled := int(float64(value) / float64(max) * float64(width))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += m.theme.Primary.Render("\u2588") // Full block
	}
	for i := filled; i < width; i++ {
		bar += m.theme.Muted.Render("\u2591") // Light shade
	}
	return bar
}

func (m *ReportModel) renderPanels() string {
	leftWidth := m.width/2 - 3
	rightWidth := m.width - leftWidth - 6

	leftPanel := m.renderBreakdownPanel(leftWidth)
	rightPanel := m.renderBySystemPanel(rightWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m *ReportModel) renderBreakdownPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Breakdown by Pattern")

	// Count exclusions by pattern
	patternCounts := make(map[string]int64)
	var totalCount int64 = 0

	if m.exclusions != nil {
		exclusions, err := m.exclusions.ListExclusions()
		if err == nil {
			for _, e := range exclusions {
				patternCounts[e.PatternMatched] += e.SizeBytes
				totalCount += e.SizeBytes
			}
		}
	}

	if len(patternCounts) == 0 {
		return m.theme.CardBorder.
			Width(width).
			Render(lipgloss.JoinVertical(lipgloss.Left, titleBar,
				m.theme.Muted.Render("  No data available.")))
	}

	var lines []string
	for pattern, size := range patternCounts {
		percent := 0
		if totalCount > 0 {
			percent = int(float64(size) / float64(totalCount) * 100)
		}
		barWidth := width - 24
		if barWidth < 5 {
			barWidth = 5
		}
		filled := int(float64(size) / float64(totalCount) * float64(barWidth))
		if filled < 1 {
			filled = 1
		}
		bar := m.theme.Primary.Render(stringOfChar('|', filled))
		line := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(14).Render(theme.Truncate(pattern, 12)),
			lipgloss.NewStyle().Width(barWidth).Render(bar),
			" "+itoa(percent)+"%",
		)
		lines = append(lines, line)
	}

	return m.theme.CardBorder.
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func (m *ReportModel) renderBySystemPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("By System")

	// Count by backup system
	systemCounts := make(map[string]int64)
	var totalCount int64 = 0

	if m.exclusions != nil {
		exclusions, err := m.exclusions.ListExclusions()
		if err == nil {
			for _, e := range exclusions {
				key := string(e.BackupSystem)
				systemCounts[key] += e.SizeBytes
				totalCount += e.SizeBytes
			}
		}
	}

	if len(systemCounts) == 0 {
		return m.theme.CardBorder.
			Width(width).
			Render(lipgloss.JoinVertical(lipgloss.Left, titleBar,
				m.theme.Muted.Render("  No data available.")))
	}

	var lines []string
	for system, size := range systemCounts {
		percent := 0
		if totalCount > 0 {
			percent = int(float64(size) / float64(totalCount) * 100)
		}
		line := m.theme.Primary.Render(backupSystemAbbr(system)+"  ") + itoa(percent) + "%  " + formatBytes(size)
		lines = append(lines, line)
	}

	return m.theme.CardBorder.
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func (m *ReportModel) renderDaemonSection() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Daemon Uptime & Events")

	eventsToday := 0
	if m.events != nil {
		count, err := m.events.ListRecentEvents(1000)
		if err == nil {
			eventsToday = len(count)
		}
	}

	lines := []string{
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.theme.Muted.Render("Events today:"),
			" "+itoa(eventsToday),
			m.theme.Muted.Render("  |  Projects detected:"),
			" N/A",
			m.theme.Muted.Render("  |  Errors:"),
			" 0",
		),
	}

	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func (m *ReportModel) renderNavBar() string {
	return m.theme.HelpText.Render("1-7 Screens  |  E: Export  |  q Quit")
}

func formatFloat(v float64) string {
	intPart := int(v)
	decPart := int((v - float64(intPart)) * 10)
	if decPart > 0 {
		return itoa(intPart) + "." + itoa(decPart)
	}
	return itoa(intPart)
}
