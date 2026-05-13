package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/components"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Audit Screen
// ============================================================================

// AuditModel displays a project tree with status icons and a detail panel
// for workspace inspection.
type AuditModel struct {
	theme      *theme.Theme
	projects   ProjectLister
	exclusions ExclusionLister

	width  int
	height int

	cursor      int
	loaded      bool
	projectsLen int

	navRequest string
}

// NewAuditModel creates a new AuditModel.
func NewAuditModel(t *theme.Theme, projects ProjectLister, exclusions ExclusionLister) *AuditModel {
	return &AuditModel{
		theme:      t,
		projects:   projects,
		exclusions: exclusions,
	}
}

// Init initializes the audit screen.
func (m *AuditModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the audit screen.
func (m *AuditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < m.projectsLen-1 {
				m.cursor++
			}
		case "esc":
			m.navRequest = "back"
		case "backspace":
			m.navRequest = "back"
		case "q":
			m.navRequest = "quit"
		}
	}
	return m, nil
}

// View renders the audit screen layout.
func (m *AuditModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	m.loadData()
	s := components.NewScreen(m.theme, m.width, m.height, "AUDIT", "")
	panels := lipgloss.JoinVertical(lipgloss.Left, m.renderPanels(), m.renderSummaryBar())
	return s.Render(panels, "", "← Esc:Back  ·  q:Quit")
}

func (m *AuditModel) loadData() {
	if m.loaded {
		return
	}
	m.loaded = true

	if m.projects != nil {
		projects, err := m.projects.ListProjects()
		if err == nil {
			m.projectsLen = len(projects)
		}
	}
}

func (m *AuditModel) renderHeader() string {
	title := m.theme.ScreenTitle.Render("AUDIT")
	return title
}

func (m *AuditModel) renderPanels() string {
	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 6

	leftPanel := m.renderProjectTree(leftWidth)
	rightPanel := m.renderDetailPanel(rightWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m *AuditModel) renderProjectTree(width int) string {
	titleBar := m.theme.CardTitle.Render("Project Tree")

	var lines []string

	if m.projects != nil {
		projects, err := m.projects.ListProjects()
		if err == nil && len(projects) > 0 {
			// Group by root path
			rootGroups := groupByRoot(projects)
			for rootPath, projs := range rootGroups {
				lines = append(lines, m.theme.Primary.Render(theme.Truncate(rootPath, width-6)))
				for i, p := range projs {
					prefix := "   "
					if i == len(projs)-1 {
						prefix = "   "
					}
					marker := "  "
					if len(lines)-1 == m.cursor {
						marker = m.theme.Selected.Render("> ")
					}
					statusIcon := statusIconForProject(p)
					totalSize := calculateProjectExclusionSize(p, m.exclusions)
					sizeStr := ""
					if totalSize > 0 {
						sizeStr = " " + formatBytes(totalSize)
					}
					line := prefix + marker + statusIcon + " " + theme.Truncate(p.Name, width-20) + sizeStr
					lines = append(lines, line)
				}
			}
		} else {
			lines = append(lines, m.theme.Muted.Render("  No projects to audit."))
		}
	} else {
		lines = append(lines, m.theme.Muted.Render("  No projects to audit."))
	}

	// Legend
	lines = append(lines, "")
	lines = append(lines, m.theme.Muted.Render("  Legend:"))
	lines = append(lines, m.theme.Success.Render("  "+checkMark+" = clean"))
	lines = append(lines, m.theme.Warning.Render("  "+warnMark+" = warnings"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.theme.CardBorder.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *AuditModel) renderDetailPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Project Detail")
	emptyMsg := m.theme.Muted.Render("  Use up/down to navigate the project tree.\n  Press Space to inspect a project.")

	return m.theme.CardBorder.
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, emptyMsg))
}

func (m *AuditModel) renderSummaryBar() string {
	cardWidth := m.width - 4

	total := m.projectsLen
	clean := m.projectsLen // Simplified; real calculation would scan exclusions
	warnings := 0
	violations := 0

	summary := lipgloss.JoinHorizontal(lipgloss.Left,
		m.theme.Muted.Render("Projects: ")+itoa(total),
		m.theme.Muted.Render("  |  "),
		m.theme.Success.Render("Clean: "+itoa(clean)),
		m.theme.Muted.Render("  |  "),
		m.theme.Warning.Render("Warnings: "+itoa(warnings)),
		m.theme.Muted.Render("  |  "),
		m.theme.Danger.Render("Violations: "+itoa(violations)),
	)

	return m.theme.CardBorder.Width(cardWidth).Render(summary)
}

func (m *AuditModel) renderNavBar() string {
	return m.theme.NavBar.Width(m.width).Padding(0, 1).Render("← Esc:Back  q:Quit  ·  up/down Navigate  ·  Right Expand  ·  Space: Inspect  ·  A: Audit All")
}

// NavRequest returns the pending navigation request, or "" if none.
func (m *AuditModel) NavRequest() string { return m.navRequest }

// NavTarget returns the target model for navigation (nil for back/quit).
func (m *AuditModel) NavTarget() tea.Model { return nil }

// ClearNav clears the navigation request.
func (m *AuditModel) ClearNav() { m.navRequest = "" }

// Helper functions for the audit screen

const checkMark = "\u2713" // ✓
const warnMark = "\u26A0"  // ⚠

func statusIconForProject(p domain.Project) string {
	if p.Status == domain.ProjectStatusActive {
		return checkMark
	}
	if p.Status == domain.ProjectStatusArchived {
		return "A"
	}
	return warnMark
}

// groupByRoot groups projects by their root path.
func groupByRoot(projects []domain.Project) map[string][]domain.Project {
	result := make(map[string][]domain.Project)
	for _, p := range projects {
		result[p.RootPath] = append(result[p.RootPath], p)
	}
	return result
}

// calculateProjectExclusionSize sums the total exclusion size for a project.
func calculateProjectExclusionSize(project domain.Project, provider ExclusionLister) int64 {
	if provider == nil {
		return 0
	}
	exclusions, err := provider.ListExclusions()
	if err != nil {
		return 0
	}
	var total int64
	for _, e := range exclusions {
		if e.ProjectID == project.ID {
			total += e.SizeBytes
		}
	}
	return total
}
