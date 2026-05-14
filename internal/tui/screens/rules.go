package screens

import (
	"context"
	"fmt"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

// RulesModel displays global and per-project hygiene rules in two split tables.
// The user presses tab to switch focus between the panels.
type RulesModel struct {
	deps         *core.TUIDependencies
	width        int
	height       int
	globalTable  table.Model
	projectTable table.Model
	rules        []domain.Rule
	focusPanel   int // 0 = global, 1 = project
	loading      bool
	err          error
}

// NewRulesModel creates a RulesModel with two configured bubbles tables.
func NewRulesModel(deps *core.TUIDependencies) *RulesModel {
	columns := []table.Column{
		{Title: "Pattern", Width: 24},
		{Title: "Max Size", Width: 12},
		{Title: "Action", Width: 10},
		{Title: "Priority", Width: 10},
		{Title: "Enabled", Width: 8},
	}

	globalT := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(8),
		table.WithWidth(70),
	)

	projectT := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(8),
		table.WithWidth(70),
	)

	return &RulesModel{
		deps:         deps,
		globalTable:  globalT,
		projectTable: projectT,
		focusPanel:   0,
	}
}

// Init loads rules from the repository.
func (m *RulesModel) Init() tea.Cmd {
	if m.deps == nil || m.deps.RuleRepo == nil {
		return nil
	}
	m.loading = true
	return m.loadRules()
}

// loadRules fetches all rules from the repository.
func (m *RulesModel) loadRules() tea.Cmd {
	return func() tea.Msg {
		rules, err := m.deps.RuleRepo.FindAll(context.Background())
		return core.RulesLoadMsg{Rules: rules, Err: err}
	}
}

// Update handles all incoming messages for the rules screen.
func (m *RulesModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {

	case core.ResizeMsg:
		m.width = message.Width
		m.height = message.Height
		panelHeight := m.height / 2
		if panelHeight < 4 {
			panelHeight = 4
		}
		tableWidth := m.width - 4
		if tableWidth < 40 {
			tableWidth = 40
		}
		m.globalTable.SetWidth(tableWidth)
		m.globalTable.SetHeight(panelHeight)
		m.projectTable.SetWidth(tableWidth)
		m.projectTable.SetHeight(panelHeight)
		return m, nil

	case core.RulesLoadMsg:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.rules = message.Rules
		var globals, projects []domain.Rule
		for _, r := range m.rules {
			if r.IsGlobal() {
				globals = append(globals, r)
			} else {
				projects = append(projects, r)
			}
		}
		m.globalTable.SetRows(buildRuleRows(globals))
		m.projectTable.SetRows(buildRuleRows(projects))
		return m, nil

	case core.RuleDeletedMsg:
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		return m, m.loadRules()

	case core.RuleSavedMsg:
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		return m, m.loadRules()

	case tea.KeyPressMsg:
		switch message.String() {
		case "tab":
			if m.focusPanel == 0 {
				m.focusPanel = 1
				m.globalTable.Blur()
				m.projectTable.Focus()
			} else {
				m.focusPanel = 0
				m.projectTable.Blur()
				m.globalTable.Focus()
			}
			return m, nil
		case "j", "down":
			if m.focusPanel == 0 {
				m.globalTable, _ = m.globalTable.Update(message)
			} else {
				m.projectTable, _ = m.projectTable.Update(message)
			}
			return m, nil
		case "k", "up":
			if m.focusPanel == 0 {
				m.globalTable, _ = m.globalTable.Update(message)
			} else {
				m.projectTable, _ = m.projectTable.Update(message)
			}
			return m, nil
		case "r":
			m.loading = true
			return m, m.loadRules()
		}
	}

	return m, nil
}

// View renders the rules screen with two stacked table panels.
func (m *RulesModel) View() tea.View {
	// Degraded mode when database is unavailable
	if m.deps == nil || !m.deps.DBAvailable {
		return tea.NewView(core.DegradeBanner("Database unavailable. Rules require an active database connection."))
	}

	if m.loading {
		return tea.NewView("Loading rules...")
	}

	// Recalculate table heights to fit within terminal
	if m.height > 0 {
		helpH := 1
		errH := 0
		if m.err != nil {
			errH = 2
		}
		remaining := m.height - helpH - errH
		panelH := remaining / 2
		if panelH < 4 {
			panelH = 4
		}
		m.globalTable.SetHeight(panelH - 3) // subtract card overhead
		m.projectTable.SetHeight(panelH - 3)
	}

	var sections []string

	if m.err != nil {
		sections = append(sections, core.StatusErr.Render(fmt.Sprintf("Error: %v", m.err)))
		sections = append(sections, "")
	}

	globalView := m.globalTable.View()
	globalCard := core.Card("Global Defaults", globalView, m.width-4, m.focusPanel == 0)
	sections = append(sections, globalCard)

	projectView := m.projectTable.View()
	projectCard := core.Card("Per-Project Overrides", projectView, m.width-4, m.focusPanel == 1)
	sections = append(sections, projectCard)

	hint := core.LabelStyle.Render("  tab:Switch Panel  r:Reload  j/k:Navigate")
	sections = append(sections, hint)

	joined := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if m.height > 0 {
		return tea.NewView(lipgloss.NewStyle().MaxHeight(m.height).Render(joined))
	}
	return tea.NewView(joined)
}

// buildRuleRows converts a slice of rules into table rows.
func buildRuleRows(rules []domain.Rule) []table.Row {
	rows := make([]table.Row, 0, len(rules))
	for _, r := range rules {
		enabled := core.StatusIcon(r.Enabled)
		maxSize := core.FormatBytes(r.MaxSizeBytes)
		rows = append(rows, table.Row{
			r.Pattern,
			maxSize,
			string(r.Action),
			fmt.Sprintf("%d", r.Priority),
			enabled,
		})
	}
	return rows
}
