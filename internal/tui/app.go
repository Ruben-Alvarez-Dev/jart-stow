package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/screens"
)

// ScreenID enumerates the 7 TUI screens.
type ScreenID int

const (
	ScreenDashboard  ScreenID = iota
	ScreenScanner
	ScreenExclusions
	ScreenRules
	ScreenAudit
	ScreenHygiene
	ScreenReport
)

var screenNames = map[ScreenID]string{
	ScreenDashboard:  "Dashboard",
	ScreenScanner:    "Scanner",
	ScreenExclusions: "Exclusions",
	ScreenRules:      "Rules",
	ScreenAudit:      "Audit",
	ScreenHygiene:    "Hygiene",
	ScreenReport:     "Report",
}

// MainModel is the root tea.Model handling navigation and screen routing.
type MainModel struct {
	deps   *core.TUIDependencies
	screen ScreenID
	width  int
	height int
	ready  bool

	dashboard  *screens.DashboardModel
	scanner    *screens.ScannerModel
	exclusions *screens.ExclusionsModel
	rules      *screens.RulesModel
	audit      *screens.AuditModel
	hygiene    *screens.HygieneModel
	report     *screens.ReportModel
}

// NewMainModel creates the root TUI model with all screens initialized.
func NewMainModel(deps *core.TUIDependencies) *MainModel {
	return &MainModel{
		deps:       deps,
		screen:     ScreenDashboard,
		dashboard:  screens.NewDashboardModel(deps),
		scanner:    screens.NewScannerModel(deps),
		exclusions: screens.NewExclusionsModel(deps),
		rules:      screens.NewRulesModel(deps),
		audit:      screens.NewAuditModel(deps),
		hygiene:    screens.NewHygieneModel(deps),
		report:     screens.NewReportModel(deps),
	}
}

// Init starts the program with initial data loading.
func (m *MainModel) Init() tea.Cmd {
	return tea.Batch(m.dashboard.Init())
}

// Update handles global keys and delegates to the active screen.
func (m *MainModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		cmds := m.propagateResize(msg.Width, msg.Height)
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, func() tea.Msg { return tea.Quit() }
		case "1":
			return m.switchScreen(ScreenDashboard)
		case "2":
			return m.switchScreen(ScreenScanner)
		case "3":
			return m.switchScreen(ScreenExclusions)
		case "4":
			return m.switchScreen(ScreenRules)
		case "5":
			return m.switchScreen(ScreenAudit)
		case "6":
			return m.switchScreen(ScreenHygiene)
		case "7":
			return m.switchScreen(ScreenReport)
		}
	}

	model, cmd := m.activeScreen().Update(message)
	m.setActiveScreen(model)
	return m, cmd
}

// View renders the global chrome (header + screen + navbar).
func (m *MainModel) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	header := m.renderHeader()
	screenView := m.activeScreen().View()
	content := screenView.Content
	navbar := m.renderNavBar()

	full := lipgloss.JoinVertical(lipgloss.Left, header, content, navbar)
	view := tea.NewView(full)
	view.AltScreen = true
	return view
}

func (m *MainModel) activeScreen() tea.Model {
	switch m.screen {
	case ScreenDashboard:
		return m.dashboard
	case ScreenScanner:
		return m.scanner
	case ScreenExclusions:
		return m.exclusions
	case ScreenRules:
		return m.rules
	case ScreenAudit:
		return m.audit
	case ScreenHygiene:
		return m.hygiene
	case ScreenReport:
		return m.report
	default:
		return m.dashboard
	}
}

func (m *MainModel) setActiveScreen(model tea.Model) {
	switch m.screen {
	case ScreenDashboard:
		m.dashboard = model.(*screens.DashboardModel)
	case ScreenScanner:
		m.scanner = model.(*screens.ScannerModel)
	case ScreenExclusions:
		m.exclusions = model.(*screens.ExclusionsModel)
	case ScreenRules:
		m.rules = model.(*screens.RulesModel)
	case ScreenAudit:
		m.audit = model.(*screens.AuditModel)
	case ScreenHygiene:
		m.hygiene = model.(*screens.HygieneModel)
	case ScreenReport:
		m.report = model.(*screens.ReportModel)
	}
}

func (m *MainModel) switchScreen(id ScreenID) (tea.Model, tea.Cmd) {
	m.screen = id
	return m, m.activeScreen().Init()
}

func (m *MainModel) propagateResize(w, h int) []tea.Cmd {
	r := core.ResizeMsg{Width: w, Height: h - 2}
	var cmds []tea.Cmd

	d, cmd := m.dashboard.Update(r)
	m.dashboard = d.(*screens.DashboardModel)
	cmds = append(cmds, cmd)

	s, cmd := m.scanner.Update(r)
	m.scanner = s.(*screens.ScannerModel)
	cmds = append(cmds, cmd)

	e, cmd := m.exclusions.Update(r)
	m.exclusions = e.(*screens.ExclusionsModel)
	cmds = append(cmds, cmd)

	ru, cmd := m.rules.Update(r)
	m.rules = ru.(*screens.RulesModel)
	cmds = append(cmds, cmd)

	a, cmd := m.audit.Update(r)
	m.audit = a.(*screens.AuditModel)
	cmds = append(cmds, cmd)

	hy, cmd := m.hygiene.Update(r)
	m.hygiene = hy.(*screens.HygieneModel)
	cmds = append(cmds, cmd)

	rp, cmd := m.report.Update(r)
	m.report = rp.(*screens.ReportModel)
	cmds = append(cmds, cmd)

	return cmds
}

func (m *MainModel) renderHeader() string {
	name := screenNames[m.screen]
	return core.HeaderStyle.Width(m.width).Render(
		fmt.Sprintf("JART-STOW  │  %s", name),
	)
}

func (m *MainModel) renderNavBar() string {
	var items []string
	for i := ScreenID(0); i <= ScreenReport; i++ {
		label := fmt.Sprintf("%d:%s", int(i)+1, screenNames[i])
		if i == m.screen {
			items = append(items, core.SelectedRowStyle.Render(label))
		} else {
			items = append(items, label)
		}
	}

	nav := strings.Join(items, " │ ")
	hint := core.LabelStyle.Render("  q:Quit")
	return core.NavBarStyle.Width(m.width).Render(nav + hint)
}

// Run launches the TUI program. Called from main.go.
func Run(deps *core.TUIDependencies) error {
	model := NewMainModel(deps)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
