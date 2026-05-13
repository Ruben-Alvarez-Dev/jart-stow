package screens

import (
	"sort"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/components"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Exclusions Screen
// ============================================================================

// ExclusionFilter represents the current filter state for the exclusions table.
type ExclusionFilter int

const (
	ExclusionFilterAll ExclusionFilter = iota
	ExclusionFilterTM
	ExclusionFilterCCC
)

// ExclusionSort represents the sort order for the exclusions table.
type ExclusionSort int

const (
	ExclusionSortProject ExclusionSort = iota
	ExclusionSortSize
	ExclusionSortDate
)

// ExclusionsModel displays and manages backup exclusions using an interactive
// table with sorting, filtering, and removal capabilities.
type ExclusionsModel struct {
	theme        *theme.Theme
	exclusions   ExclusionLister
	exclusionMgr ScreenExclusionManager

	width  int
	height int

	filter ExclusionFilter
	sortBy ExclusionSort
	table  table.Model
	loaded bool

	navRequest string
}

// NewExclusionsModel creates a new ExclusionsModel with data listing and
// exclusion management capabilities. Both providers may be nil; the screen
// shows appropriate empty states when data is unavailable.
func NewExclusionsModel(t *theme.Theme, exclusions ExclusionLister, exclusionMgr ScreenExclusionManager) *ExclusionsModel {
	return &ExclusionsModel{
		theme:        t,
		exclusions:   exclusions,
		exclusionMgr: exclusionMgr,
		filter:       ExclusionFilterAll,
		sortBy:       ExclusionSortDate,
	}
}

// Init initializes the exclusions screen. Returns nil; data is loaded on
// first render.
func (m *ExclusionsModel) Init() tea.Cmd {
	return nil
}

// Update handles key events and window resize messages for the exclusions
// screen. Navigation keys (up/down) are delegated to the table; action keys
// (r, f, s, enter) are handled directly.
func (m *ExclusionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.table, _ = m.table.Update(msg)
		case "down", "j":
			m.table, _ = m.table.Update(msg)

		case "f":
			m.filter = (m.filter + 1) % 3
			m.rebuildTable()

		case "s":
			m.sortBy = (m.sortBy + 1) % 3
			m.rebuildTable()

		case "r":
			return m, m.handleRemove()

		case "enter":
			m.reloadData()
			m.rebuildTable()

		case "esc", "backspace":
			m.navRequest = "back"

		case "q":
			m.navRequest = "quit"
		}
	}

	return m, nil
}

// handleRemove calls the exclusion manager to remove the currently selected
// exclusion and then refreshes the table.
func (m *ExclusionsModel) handleRemove() tea.Cmd {
	if m.exclusionMgr == nil {
		return nil
	}
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return nil
	}
	// First column is the exclusion ID (hidden in view but stored)
	// We store the ID as metadata. Since rows don't have metadata in the
	// standard table, we track IDs alongside the rows.
	rawExclusions, _ := m.loadFilteredExclusions()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(rawExclusions) {
		return nil
	}
	exclusion := rawExclusions[cursor]
	if err := m.exclusionMgr.RemoveExclusion(exclusion.ID); err != nil {
		return nil
	}
	m.reloadData()
	m.rebuildTable()
	return nil
}

// reloadData marks data as needing a refresh on next render.
func (m *ExclusionsModel) reloadData() {
	m.loaded = false
}

// loadData ensures exclusions data is loaded from the provider.
func (m *ExclusionsModel) loadData() {
	if m.loaded {
		return
	}
	m.loaded = true
	// Data is loaded on demand via loadFilteredExclusions.
}

// loadFilteredExclusions returns the filtered and sorted list of exclusions.
func (m *ExclusionsModel) loadFilteredExclusions() ([]domain.Exclusion, error) {
	if m.exclusions == nil {
		return nil, nil
	}
	all, err := m.exclusions.ListExclusions()
	if err != nil {
		return nil, err
	}

	// Filter by backup system
	var filtered []domain.Exclusion
	for _, ex := range all {
		switch m.filter {
		case ExclusionFilterTM:
			if ex.BackupSystem == domain.BackupSystemTimeMachine || ex.BackupSystem == domain.BackupSystemBoth {
				filtered = append(filtered, ex)
			}
		case ExclusionFilterCCC:
			if ex.BackupSystem == domain.BackupSystemCarbonCopyCloner || ex.BackupSystem == domain.BackupSystemBoth {
				filtered = append(filtered, ex)
			}
		default:
			filtered = append(filtered, ex)
		}
	}

	// Sort
	switch m.sortBy {
	case ExclusionSortProject:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].FolderPath < filtered[j].FolderPath
		})
	case ExclusionSortSize:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].SizeBytes > filtered[j].SizeBytes
		})
	case ExclusionSortDate:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].AppliedAt.After(filtered[j].AppliedAt)
		})
	}

	return filtered, nil
}

// rebuildTable reconstructs the bubbles table from the current exclusions data.
func (m *ExclusionsModel) rebuildTable() {
	exclusions, _ := m.loadFilteredExclusions()

	panelWidth := m.width - 8
	if panelWidth < 40 {
		panelWidth = 40
	}

	// Distribute column widths
	numCol := 6
	// Fixed widths for small columns
	idWidth := 4
	sizeWidth := 10
	sysWidth := 6
	statusWidth := 8
	// Remaining width split between path and pattern
	remaining := panelWidth - numCol - idWidth - sizeWidth - sysWidth - statusWidth
	pathWidth := remaining * 3 / 5
	patternWidth := remaining - pathWidth
	if pathWidth < 12 {
		pathWidth = 12
	}
	if patternWidth < 8 {
		patternWidth = 8
	}

	columns := []table.Column{
		{Title: "#", Width: idWidth},
		{Title: "Path", Width: pathWidth},
		{Title: "Pattern", Width: patternWidth},
		{Title: "Size", Width: sizeWidth},
		{Title: "Sys", Width: sysWidth},
		{Title: "Status", Width: statusWidth},
	}

	var rows []table.Row
	for _, ex := range exclusions {
		status := "active"
		if !ex.IsActive() {
			status = "removed"
		}
		rows = append(rows, table.Row{
			itoa64(ex.ID),
			ex.FolderPath,
			ex.PatternMatched,
			formatBytes(ex.SizeBytes),
			backupSystemAbbr(string(ex.BackupSystem)),
			status,
		})
	}

	// Available rows for the table:
	// m.height - header(3) - filter(2) - detail(8) - nav(1) - padding(2)
	maxRows := m.height - 16
	if maxRows < 3 {
		maxRows = 3
	}
	rowCount := len(rows)
	if rowCount < 1 {
		rowCount = 1
	}
	if rowCount > maxRows {
		rowCount = maxRows
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(rowCount),
		table.WithFocused(false),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("6"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("6")).
		Background(lipgloss.Color("235")).
		Bold(false)
	s.Cell = s.Cell.Foreground(lipgloss.Color("15"))

	tbl.SetStyles(s)
	m.table = tbl
}

// ============================================================================
// Rendering
// ============================================================================

// View renders the exclusions screen layout: header, filter bar, table, and
// navigation bar.
func (m *ExclusionsModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	m.loadData()
	m.ensureTable()

	// Count from header goes into screen subtitle
	subtitle := ""
	if exclusions, err := m.loadFilteredExclusions(); err == nil {
		subtitle = itoa(len(exclusions)) + " active"
	}
	s := components.NewScreen(m.theme, m.width, m.height, "EXCLUSIONS", subtitle)

	panels := lipgloss.JoinVertical(lipgloss.Left,
		m.renderFilterBar(),
		"",
		m.renderTable(),
		m.renderDetailPanel(),
	)
	return s.Render(panels, "", "f:Filter s:Sort r:Remove ↵:Reload Esc:Back q:Quit")
}

func (m *ExclusionsModel) ensureTable() {
	if m.table.Width() == 0 {
		m.rebuildTable()
	}
}

func (m *ExclusionsModel) renderHeader() string {
	title := m.theme.ScreenTitle.Render("EXCLUSIONS")
	exclusions, _ := m.loadFilteredExclusions()
	count := len(exclusions)
	return lipgloss.JoinHorizontal(lipgloss.Left, title, "  "+m.theme.Muted.Render(itoa(count)+" active"))
}

func (m *ExclusionsModel) renderFilterBar() string {
	barWidth := m.width - 4

	tmActive := m.filter == ExclusionFilterTM
	cccActive := m.filter == ExclusionFilterCCC
	allActive := m.filter == ExclusionFilterAll

	tmToggle := makeToggle("[TM]", tmActive, m.theme)
	cccToggle := makeToggle("[CCC]", cccActive, m.theme)
	allToggle := makeToggle("[All]", allActive, m.theme)

	filterGroup := "Filters: " + tmToggle + " " + cccToggle + " " + allToggle

	pActive := m.sortBy == ExclusionSortProject
	sActive := m.sortBy == ExclusionSortSize
	dActive := m.sortBy == ExclusionSortDate

	pToggle := makeToggle("[Project]", pActive, m.theme)
	sToggle := makeToggle("[Size]", sActive, m.theme)
	dToggle := makeToggle("[Date]", dActive, m.theme)

	sortGroup := "Sort: " + pToggle + " " + sToggle + " " + dToggle

	spacer := lipgloss.NewStyle().Width(
		barWidth - lipgloss.Width(filterGroup) - lipgloss.Width(sortGroup),
	).Render("")
	if spacer == "" || lipgloss.Width(spacer) < 0 {
		spacer = " "
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, filterGroup, spacer, sortGroup)
}

func (m *ExclusionsModel) renderTable() string {
	if m.table.Width() == 0 {
		return m.theme.CardBorder.
			Width(m.width - 4).
			Render(m.theme.Muted.Render("  No exclusions yet. Run scan to start."))
	}

	cardWidth := m.width - 4
	tableContent := m.table.View()

	return m.theme.CardBorder.Width(cardWidth).Render(tableContent)
}

func (m *ExclusionsModel) renderDetailPanel() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Selected Exclusion")

	exclusions, _ := m.loadFilteredExclusions()
	cursor := m.table.Cursor()

	var detail string
	if cursor >= 0 && cursor < len(exclusions) {
		ex := exclusions[cursor]
		line1 := m.theme.Muted.Render("ID:") + "     " + itoa64(ex.ID)
		line2 := m.theme.Muted.Render("Path:") + "   " + theme.Truncate(ex.FolderPath, cardWidth-14)
		line3 := m.theme.Muted.Render("Pattern:") + " " + ex.PatternMatched
		line4 := m.theme.Muted.Render("Size:") + "   " + formatBytes(ex.SizeBytes)
		line5 := m.theme.Muted.Render("System:") + " " + string(ex.BackupSystem)
		status := "active"
		if !ex.IsActive() {
			status = "removed"
		}
		line6 := m.theme.Muted.Render("Status:") + " " + status
		line7 := m.theme.Muted.Render("Applied:") + " " + ex.AppliedAt.Format("2006-01-02 15:04:05")
		detail = lipgloss.JoinVertical(lipgloss.Left, line1, line2, line3, line4, line5, line6, line7)
	} else {
		detail = m.theme.Muted.Render("  Press Enter to reload. Use r to remove selected exclusion.")
	}

	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, detail))
}

func (m *ExclusionsModel) renderNavBar() string {
	return m.theme.NavBar.Width(m.width).Padding(0, 1).Render(
		"Up/Down: Navigate  |  f: Filter  |  s: Sort  |  r: Remove  |  Enter: Reload  |  Esc: Back  |  q: Quit",
	)
}

// ============================================================================
// Navigation interface
// ============================================================================

// NavRequest returns the pending navigation request, or "" if none.
func (m *ExclusionsModel) NavRequest() string { return m.navRequest }

// NavTarget returns the target model for navigation (nil for back/quit).
func (m *ExclusionsModel) NavTarget() tea.Model { return nil }

// ClearNav clears the navigation request.
func (m *ExclusionsModel) ClearNav() { m.navRequest = "" }

// ============================================================================
// Helpers
// ============================================================================

// makeToggle returns a styled toggle button. Active toggles use the Selected
// style; inactive toggles use the Muted style.
func makeToggle(label string, active bool, t *theme.Theme) string {
	if active {
		return t.Selected.Render(label)
	}
	return t.Muted.Render(label)
}

// backupSystemAbbr abbreviates a BackupSystem string for compact display.
func backupSystemAbbr(bs string) string {
	switch bs {
	case "time_machine":
		return "TM"
	case "carbon_copy_cloner":
		return "CCC"
	case "both":
		return "TM+"
	default:
		return bs
	}
}
