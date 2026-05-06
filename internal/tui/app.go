// Package tui is the Bubble Tea terminal user interface for Jart-Stow.
// It uses a hierarchical navigation model: a main menu as the central hub,
// with sub-screens pushed/popped on a navigation stack.
package tui

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/screens"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// navigator is implemented by models that handle their own navigation.
type navigator interface {
	NavRequest() string
	NavTarget() tea.Model
	ClearNav()
}

// MainModel is the root Bubble Tea model with a navigation stack.
type MainModel struct {
	theme *theme.Theme
	stack []tea.Model

	width  int
	height int
	ready  bool
}

// NewMainModel creates the TUI with the main menu as entry point.
// Action interfaces may be nil for read-only mode.
func NewMainModel(
	daemon screens.DaemonStatusProvider,
	watchRoots screens.WatchRootProvider,
	projects screens.ProjectLister,
	exclusions screens.ExclusionLister,
	events screens.EventLister,
	rules screens.RuleLister,
	junk screens.JunkLister,
	scanEngine screens.ScreenScanEngine,
	junkRunner screens.ScreenJunkScanRunner,
	exclusionMgr screens.ScreenExclusionManager,
) *MainModel {
	t := theme.NewTheme()

	buildDashboard := func() tea.Model {
		return screens.NewDashboardModel(t, daemon, watchRoots, projects, exclusions, events)
	}
	buildScanner := func() tea.Model {
		return screens.NewScannerModel(t, watchRoots, scanEngine)
	}
	buildExclusions := func() tea.Model {
		return screens.NewExclusionsModel(t, exclusions, exclusionMgr)
	}
	buildRules := func() tea.Model {
		return screens.NewRulesModel(t, rules)
	}
	buildAudit := func() tea.Model {
		return screens.NewAuditModel(t, projects, exclusions)
	}
	buildHygiene := func() tea.Model {
		return screens.NewHygieneModel(t, junk, junkRunner)
	}
	buildReport := func() tea.Model {
		return screens.NewReportModel(t, exclusions, events)
	}

	menu := NewMainMenu(t, buildDashboard, buildScanner, buildExclusions,
		buildRules, buildAudit, buildHygiene, buildReport)

	return &MainModel{
		theme: t,
		stack: []tea.Model{menu},
	}
}

// Init initializes the TUI.
func (m *MainModel) Init() tea.Cmd {
	return tea.Batch(
		m.stack[len(m.stack)-1].Init(),
		tea.SetWindowTitle("Jart-Stow"),
	)
}

// sizeMsg returns a WindowSizeMsg with current stored dimensions.
func (m *MainModel) sizeMsg() tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: m.width, Height: m.height}
}

// Update handles window resize and delegates input to the top of the stack.
func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}

	top := m.stack[len(m.stack)-1]
	model, cmd := top.Update(msg)
	m.stack[len(m.stack)-1] = model

	nav, ok := model.(navigator)
	if !ok {
		return m, cmd
	}

	req := nav.NavRequest()
	if req == "" {
		return m, cmd
	}

	// Save target before clearing (ClearNav sets navTarget to nil).
	target := nav.NavTarget()

	// Clear the request immediately to prevent re-triggering on next Update.
	nav.ClearNav()
	m.stack[len(m.stack)-1] = nav.(tea.Model)

	switch req {
	case "back":
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		popped := m.stack[len(m.stack)-1]
		popped, _ = popped.Update(m.sizeMsg())
		m.stack[len(m.stack)-1] = popped
		return m, popped.Init()

	case "quit":
		return m, tea.Quit

	default:
		// "forward" — push the target screen onto the stack.
		if target != nil {
			target, _ = target.Update(m.sizeMsg())
			m.stack = append(m.stack, target)
			return m, target.Init()
		}
	}

	return m, cmd
}

// View renders the top of the navigation stack.
func (m *MainModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return m.stack[len(m.stack)-1].View()
}
