// Package tui is the Bubble Tea terminal user interface for Jart-Stow.
// It manages screen navigation and delegates rendering to individual screen models.
package tui

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/screens"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// Screen constants define the 7 available screens in the TUI.
const (
	ScreenDashboard  = 0
	ScreenScanner    = 1
	ScreenExclusions = 2
	ScreenRules      = 3
	ScreenAudit      = 4
	ScreenHygiene    = 5
	ScreenReport     = 6
)

// MainModel is the root Bubble Tea model for the TUI.
// It manages screen navigation and delegates rendering to individual screens.
type MainModel struct {
	theme  *theme.Theme
	screen int
	pages  [7]tea.Model

	width  int
	height int
	ready  bool
}

// NewMainModel creates a fully initialized MainModel with all 7 screens wired up.
// All service providers may be nil; screens gracefully handle missing data.
// The provider interfaces are defined in the screens package and shared across screens.
func NewMainModel(
	daemon screens.DaemonStatusProvider,
	watchRoots screens.WatchRootProvider,
	projects screens.ProjectLister,
	exclusions screens.ExclusionLister,
	events screens.EventLister,
	rules screens.RuleLister,
	junk screens.JunkLister,
) *MainModel {
	t := theme.NewTheme()

	m := &MainModel{
		theme:  t,
		screen: ScreenDashboard,
	}

	m.pages[ScreenDashboard] = screens.NewDashboardModel(t, daemon, watchRoots, projects, exclusions, events)
	m.pages[ScreenScanner] = screens.NewScannerModel(t, watchRoots)
	m.pages[ScreenExclusions] = screens.NewExclusionsModel(t, exclusions)
	m.pages[ScreenRules] = screens.NewRulesModel(t, rules)
	m.pages[ScreenAudit] = screens.NewAuditModel(t, projects, exclusions)
	m.pages[ScreenHygiene] = screens.NewHygieneModel(t, junk)
	m.pages[ScreenReport] = screens.NewReportModel(t, exclusions, events)

	return m
}

// Init initializes the main model and sets the window title.
func (m *MainModel) Init() tea.Cmd {
	return tea.Batch(
		m.screenInit(),
		tea.SetWindowTitle("Jart-Stow"),
	)
}

// screenInit calls Init on the current active screen.
func (m *MainModel) screenInit() tea.Cmd {
	if m.pages[m.screen] == nil {
		return nil
	}
	return m.pages[m.screen].Init()
}

// Update processes messages and delegates to the current screen.
func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Forward resize to all screens
		for _, page := range m.pages {
			if page != nil {
				if _, cmd := page.Update(msg); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "1":
			m.screen = ScreenDashboard
			cmds = append(cmds, m.screenInit())
		case "2":
			m.screen = ScreenScanner
			cmds = append(cmds, m.screenInit())
		case "3":
			m.screen = ScreenExclusions
			cmds = append(cmds, m.screenInit())
		case "4":
			m.screen = ScreenRules
			cmds = append(cmds, m.screenInit())
		case "5":
			m.screen = ScreenAudit
			cmds = append(cmds, m.screenInit())
		case "6":
			m.screen = ScreenHygiene
			cmds = append(cmds, m.screenInit())
		case "7":
			m.screen = ScreenReport
			cmds = append(cmds, m.screenInit())

		default:
			// Forward key to the current screen
			if m.pages[m.screen] != nil {
				_, cmd := m.pages[m.screen].Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	default:
		// Forward all other messages to the current screen
		if m.pages[m.screen] != nil {
			_, cmd := m.pages[m.screen].Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the current active screen.
func (m *MainModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.pages[m.screen] == nil {
		return "Screen not available"
	}

	return m.pages[m.screen].View()
}
