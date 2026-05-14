package screens

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

// DashboardModel displays the main dashboard with system status, quick stats,
// recent activity, and top offender patterns.
type DashboardModel struct {
	deps    *core.TUIDependencies
	width   int
	height  int
	loading bool
	summary *services.ReportSummary
	events  []domain.DaemonEvent
	err     error
	spinner spinner.Model
}

// NewDashboardModel creates a dashboard screen model with the given dependencies.
func NewDashboardModel(deps *core.TUIDependencies) *DashboardModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return &DashboardModel{
		deps:    deps,
		loading: true,
		spinner: s,
	}
}

// Init starts loading dashboard data (summary and events).
func (m *DashboardModel) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, m.spinner.Tick)

	cmds = append(cmds, func() tea.Msg {
		loadMsg := core.DashboardLoadMsg{}

		if m.deps.Reporter != nil {
			summary, err := m.deps.Reporter.GenerateSummary(context.Background())
			loadMsg.Summary = summary
			if err != nil {
				loadMsg.Err = err
			}
		}

		if m.deps.EventRepo != nil {
			events, err := m.deps.EventRepo.FindRecent(context.Background(), 20)
			loadMsg.Events = events
			if err != nil && loadMsg.Err == nil {
				loadMsg.Err = err
			}
		}

		return loadMsg
	})

	return tea.Batch(cmds...)
}

// Update handles messages for the dashboard screen.
func (m *DashboardModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case core.ResizeMsg:
		m.width = message.Width
		m.height = message.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd

	case core.DashboardLoadMsg:
		m.summary = message.Summary
		m.events = message.Events
		m.err = message.Err
		m.loading = false
		return m, nil

	case tea.KeyPressMsg:
		switch message.String() {
		case "r":
			m.loading = true
			return m, m.Init()
		}
	}

	return m, nil
}

// View renders the dashboard screen.
func (m *DashboardModel) View() tea.View {
	if m.loading {
		content := fmt.Sprintf("\n  %s Loading dashboard...\n", m.spinner.View())
		return tea.NewView(content)
	}

	if !m.deps.DBAvailable {
		return tea.NewView(m.renderDegraded())
	}

	return tea.NewView(m.renderDashboard())
}

func (m *DashboardModel) renderDegraded() string {
	var b strings.Builder

	b.WriteString(core.DegradeBanner("Database unavailable. Showing limited information."))
	b.WriteString("\n\n")

	if m.deps.QuickExclude != nil {
		volumes := services.GetVolumes()
		b.WriteString(core.HeaderStyle.Width(m.width).Render("Available Volumes"))
		b.WriteString("\n")
		if len(volumes) == 0 {
			b.WriteString("  No volumes detected.\n")
		}
		for _, vol := range volumes {
			icon := core.StatusOK.Render("●")
			b.WriteString(fmt.Sprintf("  %s %s  %s\n", icon, vol.Name, core.LabelStyle.Render(vol.Path)))
		}
	}

	b.WriteString("\n")
	b.WriteString(core.LabelStyle.Render("Press r to retry, or use the Scanner tab (2) for DB-free scanning."))

	return b.String()
}

func (m *DashboardModel) renderDashboard() string {
	layout := core.ResponsiveLayout(m.width)

	// Allocate heights: row1 (status cards), row2 (activity), row3 (offenders)
	row1H := 8
	row2H := 0
	row3H := 0

	if m.height > 0 {
		remaining := m.height - 2 // help line
		row2H = remaining / 3
		row3H = remaining - row1H - row2H
		if row3H < 4 {
			row3H = 4
		}
		if row2H < 4 {
			row2H = 4
		}
	}

	cardWidth := m.width / 2
	if cardWidth < 20 {
		cardWidth = 20
	}

	systemCard := m.renderSystemStatus(cardWidth)
	statsCard := m.renderQuickStats(cardWidth)

	var row1 string
	if layout == core.LayoutCompact {
		row1 = systemCard + "\n" + statsCard
	} else {
		row1 = lipgloss.JoinHorizontal(lipgloss.Top, systemCard, statsCard)
	}

	activityCard := m.renderRecentActivity(m.width, row2H)
	offendersCard := m.renderTopOffenders(m.width, row3H)

	rows := []string{row1, activityCard, offendersCard}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *DashboardModel) renderSystemStatus(width int) string {
	var b strings.Builder

	dbIcon := core.StatusIcon(m.deps.DBAvailable)
	b.WriteString(fmt.Sprintf("%s Database: ", dbIcon))
	if m.deps.DBAvailable {
		b.WriteString(core.StatusOK.Render("Connected"))
	} else {
		b.WriteString(core.StatusErr.Render("Unavailable"))
	}
	b.WriteString("\n")

	if m.summary != nil {
		projIcon := core.RunningIcon()
		b.WriteString(fmt.Sprintf("%s Projects: %s\n", projIcon, core.ValueStyle.Render(
			fmt.Sprintf("%d total, %d active", m.summary.ProjectsTotal, m.summary.ProjectsActive),
		)))

		exclIcon := core.StatusIcon(m.summary.ExclusionsActive > 0)
		b.WriteString(fmt.Sprintf("%s Exclusions: %s\n", exclIcon, core.ValueStyle.Render(
			fmt.Sprintf("%d active (%s)", m.summary.ExclusionsActive, core.FormatBytes(m.summary.ExclusionsTotalSize)),
		)))

		if m.summary.JunkItemsPending > 0 {
			junkIcon := core.StatusIcon(false)
			b.WriteString(fmt.Sprintf("%s Junk Pending: %s\n", junkIcon, core.StatusWarn.Render(
				fmt.Sprintf("%d items", m.summary.JunkItemsPending),
			)))
		}
	} else {
		b.WriteString(core.LabelStyle.Render("  No summary data available.\n"))
	}

	return core.Card("System Status", b.String(), width, false)
}

func (m *DashboardModel) renderQuickStats(width int) string {
	var b strings.Builder

	if m.summary == nil {
		b.WriteString(core.LabelStyle.Render("No data."))
		return core.Card("Quick Stats", b.String(), width, false)
	}

	b.WriteString(fmt.Sprintf("  Events Today: %s\n", core.ValueStyle.Render(
		fmt.Sprintf("%d", m.summary.EventsToday),
	)))

	if len(m.summary.SystemBreakdowns) > 0 {
		b.WriteString("\n  By Backup System:\n")
		for _, sb := range m.summary.SystemBreakdowns {
			bar := core.BarChart(sb.System, sb.SizeBytes, m.summary.ExclusionsTotalSize, width-24)
			b.WriteString(fmt.Sprintf("    %s  %d items\n", bar, sb.Count))
		}
	}

	return core.Card("Quick Stats", b.String(), width, false)
}

func (m *DashboardModel) renderRecentActivity(width, maxH int) string {
	var b strings.Builder

	if len(m.events) == 0 {
		b.WriteString(core.LabelStyle.Render("  No recent events."))
		return core.Card("Recent Activity", b.String(), width, false)
	}

	// Calculate max rows based on available height
	maxRows := 8
	if maxH > 0 {
		inner := maxH - cardPad - 1 // card overhead + title
		if inner < 3 {
			inner = 3
		}
		maxRows = inner
	}
	if len(m.events) < maxRows {
		maxRows = len(m.events)
	}

	for i := 0; i < maxRows; i++ {
		e := m.events[i]
		path := core.TruncatePath(e.FolderPath, 30)
		timeStr := e.CreatedAt.Format("15:04:05")

		var icon string
		switch e.EventType {
		case domain.EventTypeScanCompleted:
			icon = core.StatusOK.Render("S")
		case domain.EventTypeExclusionApplied:
			icon = core.StatusOK.Render("E")
		case domain.EventTypeExclusionRemoved:
			icon = core.StatusWarn.Render("R")
		case domain.EventTypeError:
			icon = core.StatusErr.Render("!")
		default:
			icon = core.LabelStyle.Render("·")
		}

		b.WriteString(fmt.Sprintf("  %s %s %-20s %s\n",
			icon,
			core.LabelStyle.Render(timeStr),
			string(e.EventType),
			core.ValueStyle.Render(path),
		))
	}

	if len(m.events) > maxRows {
		b.WriteString(fmt.Sprintf("\n  %s\n", core.LabelStyle.Render(
			fmt.Sprintf("... and %d more", len(m.events)-maxRows),
		)))
	}

	card := core.Card("Recent Activity", b.String(), width, false)
	if maxH > 0 {
		return lipgloss.NewStyle().MaxHeight(maxH).Render(card)
	}
	return card
}

func (m *DashboardModel) renderTopOffenders(width, maxH int) string {
	var b strings.Builder

	if m.summary == nil || len(m.summary.PatternBreakdowns) == 0 {
		b.WriteString(core.LabelStyle.Render("  No pattern data available."))
		return core.Card("Top Offenders by Pattern", b.String(), width, false)
	}

	breakdowns := make([]services.PatternBreakdown, len(m.summary.PatternBreakdowns))
	copy(breakdowns, m.summary.PatternBreakdowns)
	sort.Slice(breakdowns, func(i, j int) bool {
		return breakdowns[i].SizeBytes > breakdowns[j].SizeBytes
	})

	var maxSize int64
	for _, pb := range breakdowns {
		if pb.SizeBytes > maxSize {
			maxSize = pb.SizeBytes
		}
	}

	maxRows := 10
	if maxH > 0 {
		inner := maxH - cardPad - 1
		if inner < 3 {
			inner = 3
		}
		maxRows = inner
	}
	if len(breakdowns) < maxRows {
		maxRows = len(breakdowns)
	}

	barWidth := width - 40
	if barWidth < 10 {
		barWidth = 10
	}

	for i := 0; i < maxRows; i++ {
		pb := breakdowns[i]
		bar := core.SparkLine([]int64{pb.SizeBytes}, maxSize, 1)
		b.WriteString(fmt.Sprintf("  %-20s %4d items  %s %s\n",
			core.ValueStyle.Render(pb.Pattern),
			pb.Count,
			bar,
			core.FormatBytes(pb.SizeBytes),
		))
	}

	card := core.Card("Top Offenders by Pattern", b.String(), width, false)
	if maxH > 0 {
		return lipgloss.NewStyle().MaxHeight(maxH).Render(card)
	}
	return card
}
