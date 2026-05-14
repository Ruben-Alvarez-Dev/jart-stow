package screens

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

// ReportModel is the Bubble Tea model for the Report screen.
type ReportModel struct {
	deps    *core.TUIDependencies
	width   int
	height  int
	summary *services.ReportSummary
	history []services.HistoryEntry
	loading bool
	spinner spinner.Model
}

// NewReportModel creates a new ReportModel with the given dependencies.
func NewReportModel(deps *core.TUIDependencies) *ReportModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return &ReportModel{
		deps:    deps,
		spinner: s,
	}
}

// Init loads the report summary and history.
func (m *ReportModel) Init() tea.Cmd {
	if m.deps == nil || !m.deps.DBAvailable || m.deps.Reporter == nil {
		return nil
	}
	m.loading = true
	return func() tea.Msg {
		summary, err := m.deps.Reporter.GenerateSummary(context.Background())
		if err != nil {
			return core.ReportLoadMsg{Summary: nil, History: nil, Err: err}
		}
		history, histErr := m.deps.Reporter.GenerateHistory(context.Background(), 30)
		if histErr != nil {
			return core.ReportLoadMsg{Summary: summary, History: nil, Err: histErr}
		}
		return core.ReportLoadMsg{Summary: summary, History: history, Err: nil}
	}
}

// Update handles messages for the Report screen.
func (m *ReportModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case core.ResizeMsg:
		m.width = message.Width
		m.height = message.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd

	case core.ReportLoadMsg:
		m.loading = false
		if message.Err != nil {
			return m, nil
		}
		m.summary = message.Summary
		m.history = message.History
		return m, nil

	case tea.KeyPressMsg:
		switch message.String() {
		case "r":
			return m, m.Init()
		}
	}

	return m, nil
}

// View renders the Report screen.
func (m *ReportModel) View() tea.View {
	if m.deps == nil || !m.deps.DBAvailable {
		return tea.NewView(core.DegradeBanner("Database unavailable -- report data cannot be loaded."))
	}

	if m.loading {
		return tea.NewView(fmt.Sprintf("\n  %s Loading report...", m.spinner.View()))
	}

	if m.summary == nil {
		return tea.NewView(core.DegradeBanner("No report data available. Press 'r' to reload."))
	}

	if m.width == 0 || m.height == 0 {
		return tea.NewView("Initializing report...")
	}

	layout := core.ResponsiveLayout(m.width)

	// Allocate heights: sparkline(6), charts(~half), stats(3), help(1)
	helpH := 1
	statsH := 3
	sparkH := 6
	remaining := m.height - helpH - statsH - sparkH
	if remaining < 6 {
		remaining = 6
	}
	chartH := remaining

	halfW := m.width/2 - 2
	if layout == core.LayoutCompact {
		halfW = m.width
	}

	sparkView := m.renderSparkline(m.width - 4)

	patternPanel := m.renderPatternBreakdown(halfW, chartH)
	systemPanel := m.renderSystemBreakdown(halfW, chartH)

	var middle string
	if layout == core.LayoutCompact {
		middle = patternPanel + "\n" + systemPanel
	} else {
		middle = lipgloss.JoinHorizontal(lipgloss.Top, patternPanel, systemPanel)
	}

	statsGrid := m.renderStatsGrid(m.width)

	helpLine := core.LabelStyle.Render("  r:reload")

	rows := []string{sparkView, middle, statsGrid, helpLine}
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m *ReportModel) renderSparkline(width int) string {
	var values []int64
	var maxVal int64
	for _, h := range m.history {
		values = append(values, h.AddedSizeBytes)
		if h.AddedSizeBytes > maxVal {
			maxVal = h.AddedSizeBytes
		}
	}

	if len(values) == 0 {
		return core.Card("Exclusion History (30 days)", core.LabelStyle.Render("  No history data available."), width, false)
	}

	reversed := make([]int64, len(values))
	for i, v := range values {
		reversed[len(values)-1-i] = v
	}

	spark := core.SparkLine(reversed, maxVal, width-6)

	var bodyLines []string
	bodyLines = append(bodyLines, "  "+spark)
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, core.LabelStyle.Render(fmt.Sprintf("  Peak: %s  |  Total entries: %d", core.FormatBytes(maxVal), len(m.history))))

	return core.Card("Exclusion History (30 days)", strings.Join(bodyLines, "\n"), width, false)
}

func (m *ReportModel) renderPatternBreakdown(width, maxH int) string {
	var lines []string

	if len(m.summary.PatternBreakdowns) == 0 {
		lines = append(lines, core.LabelStyle.Render("  No pattern data."))
		return core.Card("By Pattern", strings.Join(lines, "\n"), width, true)
	}

	var maxSize int64
	for _, pb := range m.summary.PatternBreakdowns {
		if pb.SizeBytes > maxSize {
			maxSize = pb.SizeBytes
		}
	}

	barW := width - 32
	if barW < 10 {
		barW = 10
	}

	// Limit entries to fit panel height
	maxRows := len(m.summary.PatternBreakdowns)
	if maxH > 0 {
		inner := maxH - cardPad - 1
		if inner < 3 {
			inner = 3
		}
		if maxRows > inner {
			maxRows = inner
		}
	}

	for i := 0; i < maxRows; i++ {
		pb := m.summary.PatternBreakdowns[i]
		chart := core.BarChart(pb.Pattern, pb.SizeBytes, maxSize, barW)
		lines = append(lines, chart)
	}

	card := core.Card("By Pattern", strings.Join(lines, "\n"), width, true)
	if maxH > 0 {
		return lipgloss.NewStyle().MaxHeight(maxH).Render(card)
	}
	return card
}

func (m *ReportModel) renderSystemBreakdown(width, maxH int) string {
	var lines []string

	if len(m.summary.SystemBreakdowns) == 0 {
		lines = append(lines, core.LabelStyle.Render("  No system data."))
		return core.Card("By System", strings.Join(lines, "\n"), width, false)
	}

	var maxSize int64
	for _, sb := range m.summary.SystemBreakdowns {
		if sb.SizeBytes > maxSize {
			maxSize = sb.SizeBytes
		}
	}

	barW := width - 32
	if barW < 10 {
		barW = 10
	}

	maxRows := len(m.summary.SystemBreakdowns)
	if maxH > 0 {
		inner := maxH - cardPad - 1
		if inner < 3 {
			inner = 3
		}
		if maxRows > inner {
			maxRows = inner
		}
	}

	for i := 0; i < maxRows; i++ {
		sb := m.summary.SystemBreakdowns[i]
		chart := core.BarChart(sb.System, sb.SizeBytes, maxSize, barW)
		lines = append(lines, chart)
	}

	card := core.Card("By System", strings.Join(lines, "\n"), width, false)
	if maxH > 0 {
		return lipgloss.NewStyle().MaxHeight(maxH).Render(card)
	}
	return card
}

func (m *ReportModel) renderStatsGrid(width int) string {
	colW := width/4 - 2
	if colW < 12 {
		colW = 12
	}

	statStyle := lipgloss.NewStyle().Width(colW).Align(lipgloss.Left)

	cells := []string{
		statStyle.Render(fmt.Sprintf("  Projects\n  %s", core.ValueStyle.Render(fmt.Sprintf("%d / %d active", m.summary.ProjectsActive, m.summary.ProjectsTotal)))),
		statStyle.Render(fmt.Sprintf("  Exclusions\n  %s", core.ValueStyle.Render(fmt.Sprintf("%d (%s)", m.summary.ExclusionsActive, core.FormatBytes(m.summary.ExclusionsTotalSize))))),
		statStyle.Render(fmt.Sprintf("  Events Today\n  %s", core.ValueStyle.Render(fmt.Sprintf("%d", m.summary.EventsToday)))),
		statStyle.Render(fmt.Sprintf("  Pending Junk\n  %s", core.ValueStyle.Render(fmt.Sprintf("%d", m.summary.JunkItemsPending)))),
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	return lipgloss.NewStyle().Width(width).Render(row)
}
