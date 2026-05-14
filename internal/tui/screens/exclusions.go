package screens

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

// filterNames maps filterIdx to a human-readable label and optional backup system filter.
var filterNames = []string{"All", "TM", "CCC"}
var filterSystems = []domain.BackupSystem{"", domain.BackupSystemTimeMachine, domain.BackupSystemCarbonCopyCloner}

// ExclusionsModel displays active exclusions in a filterable, navigable table
// with a detail panel for the selected row.
type ExclusionsModel struct {
	deps      *core.TUIDependencies
	width     int
	height    int
	table     table.Model
	exclusions []domain.Exclusion
	filterIdx int
	selected  *domain.Exclusion
	loading   bool
	err       error
}

// NewExclusionsModel creates an ExclusionsModel with a configured bubbles table.
func NewExclusionsModel(deps *core.TUIDependencies) *ExclusionsModel {
	columns := []table.Column{
		{Title: "Project", Width: 12},
		{Title: "Path", Width: 30},
		{Title: "Pattern", Width: 18},
		{Title: "Size", Width: 10},
		{Title: "System", Width: 6},
		{Title: "Status", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithWidth(90),
	)

	return &ExclusionsModel{
		deps:  deps,
		table: t,
	}
}

// Init kicks off the initial data load.
func (m *ExclusionsModel) Init() tea.Cmd {
	if m.deps == nil || m.deps.ExclusionRepo == nil {
		return nil
	}
	m.loading = true
	return m.loadExclusions()
}

// loadExclusions fetches exclusions from the repository, optionally filtered by backup system.
func (m *ExclusionsModel) loadExclusions() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		all, err := m.deps.ExclusionRepo.FindActive(ctx)
		if err != nil {
			return core.ExclusionsLoadMsg{Err: err}
		}

		sys := filterSystems[m.filterIdx]
		if sys == "" {
			return core.ExclusionsLoadMsg{Exclusions: all}
		}

		var filtered []domain.Exclusion
		for _, ex := range all {
			if ex.BackupSystem == sys || ex.BackupSystem == domain.BackupSystemBoth {
				filtered = append(filtered, ex)
			}
		}
		return core.ExclusionsLoadMsg{Exclusions: filtered}
	}
}

// Update handles all incoming messages for the exclusions screen.
func (m *ExclusionsModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {

	case core.ResizeMsg:
		m.width = message.Width
		m.height = message.Height
		tableHeight := m.height / 2
		if tableHeight < 4 {
			tableHeight = 4
		}
		m.table.SetWidth(m.width - 4)
		m.table.SetHeight(tableHeight)
		return m, nil

	case core.ExclusionsLoadMsg:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.exclusions = message.Exclusions
		m.selected = nil
		m.table.SetRows(m.buildRows())
		return m, nil

	case core.ExclusionRemovedMsg:
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		return m, m.loadExclusions()

	case tea.KeyPressMsg:
		switch message.String() {
		case "j", "down":
			m.table, _ = m.table.Update(message)
			m.syncSelected()
			return m, nil
		case "k", "up":
			m.table, _ = m.table.Update(message)
			m.syncSelected()
			return m, nil
		case "f":
			m.filterIdx = (m.filterIdx + 1) % len(filterNames)
			m.loading = true
			return m, m.loadExclusions()
		case "r":
			if m.selected != nil && m.selected.IsActive() {
				id := m.selected.ID
				return m, func() tea.Msg {
					err := m.deps.ExclusionRepo.MarkRemoved(context.Background(), id)
					return core.ExclusionRemovedMsg{ID: id, Err: err}
				}
			}
			return m, nil
		case "esc":
			m.selected = nil
			return m, nil
		}
	}

	return m, nil
}

// syncSelected updates the selected exclusion based on the current table cursor.
func (m *ExclusionsModel) syncSelected() {
	if len(m.exclusions) == 0 {
		m.selected = nil
		return
	}
	idx := m.table.Cursor()
	if idx < len(m.exclusions) {
		m.selected = &m.exclusions[idx]
	} else {
		m.selected = nil
	}
}

// buildRows converts exclusions into table rows.
func (m *ExclusionsModel) buildRows() []table.Row {
	rows := make([]table.Row, 0, len(m.exclusions))
	for _, ex := range m.exclusions {
		status := "Active"
		if !ex.IsActive() {
			status = "Removed"
		}
		systemLabel := "Both"
		switch ex.BackupSystem {
		case domain.BackupSystemTimeMachine:
			systemLabel = "TM"
		case domain.BackupSystemCarbonCopyCloner:
			systemLabel = "CCC"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", ex.ProjectID),
			core.TruncatePath(ex.FolderPath, 28),
			ex.PatternMatched,
			core.FormatBytes(ex.SizeBytes),
			systemLabel,
			status,
		})
	}
	return rows
}

// View renders the exclusions screen.
func (m *ExclusionsModel) View() tea.View {
	// Degraded mode when database is unavailable
	if m.deps == nil || !m.deps.DBAvailable {
		return tea.NewView(core.DegradeBanner("Database unavailable. Exclusions require an active database connection."))
	}

	if m.loading {
		return tea.NewView("Loading exclusions...")
	}

	// Recalculate table height to fit within terminal
	if m.height > 0 {
		detailH := 0
		if m.selected != nil {
			detailH = 9 // card with 7 lines of detail + border
		}
		filterH := 3
		helpH := 1
		errH := 0
		if m.err != nil {
			errH = 2
		}
		remaining := m.height - filterH - helpH - errH - detailH
		if remaining < 4 {
			remaining = 4
		}
		m.table.SetHeight(remaining)
	}

	var sections []string

	if m.err != nil {
		sections = append(sections, core.StatusErr.Render(fmt.Sprintf("Error: %v", m.err)))
		sections = append(sections, "")
	}

	filterBar := m.renderFilterBar()
	sections = append(sections, filterBar)

	tableView := m.table.View()
	sections = append(sections, tableView)

	if m.selected != nil {
		detail := m.renderDetail()
		sections = append(sections, detail)
	}

	hint := core.LabelStyle.Render("  f:Filter  r:Remove  esc:Deselect  j/k:Navigate")
	sections = append(sections, hint)

	joined := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if m.height > 0 {
		return tea.NewView(lipgloss.NewStyle().MaxHeight(m.height).Render(joined))
	}
	return tea.NewView(joined)
}

// renderFilterBar shows the current filter selection.
func (m *ExclusionsModel) renderFilterBar() string {
	var parts []string
	for i, name := range filterNames {
		if i == m.filterIdx {
			parts = append(parts, core.ValueStyle.Render(fmt.Sprintf("[%s]", name)))
		} else {
			parts = append(parts, core.LabelStyle.Render(fmt.Sprintf(" %s ", name)))
		}
	}
	filter := strings.Join(parts, " ")
	return core.CardStyle.Width(m.width - 4).Render(
		core.LabelStyle.Render("Filter: ") + filter +
			core.LabelStyle.Render(fmt.Sprintf("    (%d exclusions)", len(m.exclusions))),
	)
}

// renderDetail shows information about the currently selected exclusion.
func (m *ExclusionsModel) renderDetail() string {
	if m.selected == nil {
		return ""
	}
	ex := m.selected

	systemLabel := "Both"
	switch ex.BackupSystem {
	case domain.BackupSystemTimeMachine:
		systemLabel = "Time Machine"
	case domain.BackupSystemCarbonCopyCloner:
		systemLabel = "Carbon Copy Cloner"
	}

	var lines []string
	lines = append(lines, core.LabelStyle.Render("ID: ")+fmt.Sprintf("%d", ex.ID))
	lines = append(lines, core.LabelStyle.Render("Path: ")+ex.FolderPath)
	lines = append(lines, core.LabelStyle.Render("Pattern: ")+ex.PatternMatched)
	lines = append(lines, core.LabelStyle.Render("Size: ")+core.FormatBytes(ex.SizeBytes))
	lines = append(lines, core.LabelStyle.Render("System: ")+systemLabel)
	lines = append(lines, core.LabelStyle.Render("Applied: ")+ex.AppliedAt.Format("2006-01-02 15:04"))
	statusIcon := core.StatusIcon(ex.IsActive())
	lines = append(lines, core.LabelStyle.Render("Status: ")+statusIcon)

	body := strings.Join(lines, "\n")
	return core.Card("Selected Exclusion", body, m.width-4, true)
}
