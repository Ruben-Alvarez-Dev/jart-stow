package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Rules Screen
// ============================================================================

// RuleLister provides access to hygiene rules for the TUI.
type RuleLister interface {
	ListGlobalRules() ([]domain.Rule, error)
	ListProjectRules() ([]domain.Rule, error)
}

// RulesModel displays and manages hygiene rules (global defaults and per-project overrides).
type RulesModel struct {
	theme *theme.Theme
	rules RuleLister

	width  int
	height int

	cursor  int
	focus   int // 0 = global section, 1 = project overrides section
	loaded  bool
}

// NewRulesModel creates a new RulesModel.
func NewRulesModel(t *theme.Theme, rules RuleLister) *RulesModel {
	return &RulesModel{
		theme: t,
		rules: rules,
	}
}

// Init initializes the rules screen.
func (m *RulesModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the rules screen.
func (m *RulesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "tab":
			m.focus = (m.focus + 1) % 2
			m.cursor = 0
		}
	}
	return m, nil
}

// View renders the rules screen layout.
func (m *RulesModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	m.loadData()

	header := m.renderHeader()
	globalSection := m.renderGlobalSection()
	projectSection := m.renderProjectSection()
	navBar := m.renderNavBar()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		globalSection,
		projectSection,
	)

	mainArea := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 3).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, navBar)
}

func (m *RulesModel) loadData() {
	if m.loaded {
		return
	}
	m.loaded = true
}

func (m *RulesModel) renderHeader() string {
	title := m.theme.Header.Render("RULES")
	count := m.countRules()
	subtitle := m.theme.Muted.Render(itoa(count) + " active")
	return lipgloss.JoinHorizontal(lipgloss.Left, title, "  "+subtitle)
}

func (m *RulesModel) countRules() int {
	if m.rules == nil {
		return 0
	}
	global, _ := m.rules.ListGlobalRules()
	project, _ := m.rules.ListProjectRules()
	return len(global) + len(project)
}

func (m *RulesModel) renderGlobalSection() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Global Defaults")

	// Table header
	headerStyle := m.theme.CardTitle
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(20).Render("Pattern"),
		headerStyle.Width(12).Render("Max Size"),
		headerStyle.Width(12).Render("Action"),
		headerStyle.Width(10).Render("Enabled"),
	)
	divider := m.theme.Muted.Render(stringOfChar('-', cardWidth-4))

	var content string
	if m.rules != nil {
		rules, err := m.rules.ListGlobalRules()
		if err == nil && len(rules) > 0 {
			rows := []string{header, divider}
			for i, rule := range rules {
				rowStyle := lipgloss.NewStyle()
				if m.focus == 0 && i == m.cursor {
					rowStyle = m.theme.Selected
				}
				enabledStr := "x"
				if !rule.Enabled {
					enabledStr = " "
				}
				row := lipgloss.JoinHorizontal(lipgloss.Left,
					rowStyle.Width(20).Render(theme.Truncate(rule.Pattern, 18)),
					rowStyle.Width(12).Render(formatBytes(rule.MaxSizeBytes)),
					rowStyle.Width(12).Render(string(rule.Action)),
					rowStyle.Width(10).Render("["+enabledStr+"]"),
				)
				rows = append(rows, row)
			}
			content = lipgloss.JoinVertical(lipgloss.Left, rows...)
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left,
				header, divider,
				m.theme.Muted.Render("  No global rules defined yet."))
		}
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			header, divider,
			m.theme.Muted.Render("  No global rules defined yet."))
	}

	border := m.theme.CardBorder
	if m.focus == 0 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.Width(cardWidth).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *RulesModel) renderProjectSection() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Per-Project Overrides")

	headerStyle := m.theme.CardTitle
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		headerStyle.Width(16).Render("Project"),
		headerStyle.Width(20).Render("Pattern"),
		headerStyle.Width(12).Render("Max Size"),
		headerStyle.Width(12).Render("Action"),
	)
	divider := m.theme.Muted.Render(stringOfChar('-', cardWidth-4))

	var content string
	if m.rules != nil {
		rules, err := m.rules.ListProjectRules()
		if err == nil && len(rules) > 0 {
			rows := []string{header, divider}
			for i, rule := range rules {
				rowStyle := lipgloss.NewStyle()
				if m.focus == 1 && i == m.cursor {
					rowStyle = m.theme.Selected
				}
				projectID := "-"
				if rule.ProjectID != nil {
					projectID = itoa64(*rule.ProjectID)
				}
				row := lipgloss.JoinHorizontal(lipgloss.Left,
					rowStyle.Width(16).Render(theme.Truncate(projectID, 14)),
					rowStyle.Width(20).Render(theme.Truncate(rule.Pattern, 18)),
					rowStyle.Width(12).Render(formatBytes(rule.MaxSizeBytes)),
					rowStyle.Width(12).Render(string(rule.Action)),
				)
				rows = append(rows, row)
			}
			content = lipgloss.JoinVertical(lipgloss.Left, rows...)
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left,
				header, divider,
				m.theme.Muted.Render("  No per-project overrides defined yet."))
		}
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			header, divider,
			m.theme.Muted.Render("  No per-project overrides defined yet."))
	}

	border := m.theme.CardBorder
	if m.focus == 1 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.Width(cardWidth).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *RulesModel) renderNavBar() string {
	return m.theme.HelpText.Render("A: Add Rule  |  E: Edit  |  D: Delete  |  up/down: Navigate")
}
