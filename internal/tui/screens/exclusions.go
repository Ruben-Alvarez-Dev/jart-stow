package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
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

// ExclusionsModel displays and manages backup exclusions.
type ExclusionsModel struct {
	theme      *theme.Theme
	exclusions ExclusionLister

	width  int
	height int

	filter     ExclusionFilter
	sort       ExclusionSort
	selected   int
	cursor     int
	loaded     bool
}

// NewExclusionsModel creates a new ExclusionsModel.
func NewExclusionsModel(t *theme.Theme, exclusions ExclusionLister) *ExclusionsModel {
	return &ExclusionsModel{
		theme:      t,
		exclusions: exclusions,
		filter:     ExclusionFilterAll,
		sort:       ExclusionSortDate,
	}
}

// Init initializes the exclusions screen.
func (m *ExclusionsModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the exclusions screen.
func (m *ExclusionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor++
		case "f":
			// Cycle filter: All -> TM -> CCC -> All
			m.filter = (m.filter + 1) % 3
			m.cursor = 0
		case "s":
			// Cycle sort: Project -> Size -> Date -> Project
			m.sort = (m.sort + 1) % 3
		case "enter":
			m.selected = m.cursor
		}
	}
	return m, nil
}

// View renders the exclusions screen layout.
func (m *ExclusionsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	m.loadData()

	header := m.renderHeader()
	filterBar := m.renderFilterBar()
	table := m.renderTable()
	detailPanel := m.renderDetailPanel()
	navBar := m.renderNavBar()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		filterBar,
		"",
		table,
		detailPanel,
	)

	mainArea := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 3).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, navBar)
}

func (m *ExclusionsModel) loadData() {
	if m.loaded {
		return
	}
	m.loaded = true
	// Data is accessed via View; no preloading needed beyond the provider reference.
}

func (m *ExclusionsModel) renderHeader() string {
	title := m.theme.Header.Render("EXCLUSIONS")
	count := m.countExclusions()
	subtitle := m.theme.Muted.Render(itoa(count) + " active")
	return lipgloss.JoinHorizontal(lipgloss.Left, title, "  "+subtitle)
}

func (m *ExclusionsModel) countExclusions() int {
	if m.exclusions == nil {
		return 0
	}
	count, err := m.exclusions.CountExclusions()
	if err != nil {
		return 0
	}
	return count
}

func (m *ExclusionsModel) renderFilterBar() string {
	barWidth := m.width - 4

	// Filter toggles
	tmActive := m.filter == ExclusionFilterTM
	cccActive := m.filter == ExclusionFilterCCC
	allActive := m.filter == ExclusionFilterAll

	tmToggle := "[TM]"
	cccToggle := "[CCC]"
	allToggle := "[All]"
	if tmActive {
		tmToggle = m.theme.Selected.Render("[TM]")
	} else {
		tmToggle = m.theme.Muted.Render("[TM]")
	}
	if cccActive {
		cccToggle = m.theme.Selected.Render("[CCC]")
	} else {
		cccToggle = m.theme.Muted.Render("[CCC]")
	}
	if allActive {
		allToggle = m.theme.Selected.Render("[All]")
	} else {
		allToggle = m.theme.Muted.Render("[All]")
	}

	filterGroup := "Filters: " + tmToggle + " " + cccToggle + " " + allToggle

	// Sort toggles
	pActive := m.sort == ExclusionSortProject
	sActive := m.sort == ExclusionSortSize
	dActive := m.sort == ExclusionSortDate

	pToggle := "[Project]"
	sToggle := "[Size]"
	dToggle := "[Date]"
	if pActive {
		pToggle = m.theme.Selected.Render("[Project]")
	} else {
		pToggle = m.theme.Muted.Render("[Project]")
	}
	if sActive {
		sToggle = m.theme.Selected.Render("[Size]")
	} else {
		sToggle = m.theme.Muted.Render("[Size]")
	}
	if dActive {
		dToggle = m.theme.Selected.Render("[Date]")
	} else {
		dToggle = m.theme.Muted.Render("[Date]")
	}

	sortGroup := "Sort: " + pToggle + " " + sToggle + " " + dToggle

	spacer := lipgloss.NewStyle().Width(barWidth - lipgloss.Width(filterGroup) - lipgloss.Width(sortGroup)).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Left, filterGroup, spacer, sortGroup)
}

func (m *ExclusionsModel) renderTable() string {
	cardWidth := m.width - 4

	// Table header
	headerStyle := m.theme.CardTitle
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(4).Render("#"),
		headerStyle.Width(16).Render("Project"),
		headerStyle.Width(28).Render("Path"),
		headerStyle.Width(10).Render("Size"),
		headerStyle.Width(6).Render("System"),
	)

	// Divider line
	divider := m.theme.Muted.Render(stringOfChar('-', cardWidth-4))

	content := lipgloss.JoinVertical(lipgloss.Left, header, divider)

	// Table rows - load from provider
	if m.exclusions != nil {
		exclusions, err := m.exclusions.ListExclusions()
		if err == nil && len(exclusions) > 0 {
			for i, ex := range exclusions {
				if i >= 10 {
					// Show ellipsis
					content = lipgloss.JoinVertical(lipgloss.Left, content, m.theme.Muted.Render("  ..."))
					break
				}
				rowStyle := lipgloss.NewStyle()
				if i == m.cursor {
					rowStyle = m.theme.Selected
				}
				row := lipgloss.JoinHorizontal(lipgloss.Left,
					rowStyle.Width(4).Render(itoa(i+1)),
					rowStyle.Width(16).Render(theme.Truncate(itoa64(ex.ProjectID), 14)),
					rowStyle.Width(28).Render(theme.Truncate(ex.FolderPath, 26)),
					rowStyle.Width(10).Render(formatBytes(ex.SizeBytes)),
					rowStyle.Width(6).Render(backupSystemAbbr(string(ex.BackupSystem))),
				)
				content = lipgloss.JoinVertical(lipgloss.Left, content, row)
			}
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left, content,
				m.theme.Muted.Render("  No exclusions yet. Run scan to start."))
		}
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left, content,
			m.theme.Muted.Render("  No exclusions yet. Run scan to start."))
	}

	return m.theme.CardBorder.Width(cardWidth).Render(content)
}

func (m *ExclusionsModel) renderDetailPanel() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Selected Exclusion")
	emptyMsg := m.theme.Muted.Render("  Press Enter on a row to select an exclusion for details.")

	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, emptyMsg))
}

func (m *ExclusionsModel) renderNavBar() string {
	return m.theme.HelpText.Render("up/down Navigate  |  Enter: Select  |  R: Restore  |  F: Filter")
}

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
