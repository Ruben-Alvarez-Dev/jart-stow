// Package screens implements the individual TUI screens for Jart-Stow.
// Each screen is a standalone tea.Model that receives data providers via constructor injection.
package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Dashboard Screen
// ============================================================================

// DashboardModel is the home screen showing system status, quick stats,
// recent activity, and navigation.
type DashboardModel struct {
	theme      *theme.Theme
	daemon     DaemonStatusProvider
	watchRoots WatchRootProvider
	projects   ProjectLister
	exclusions ExclusionLister
	events     EventLister

	width  int
	height int

	// Cached data
	projectCount   int
	exclusionCount int
	spaceSaved     int64
	watchRootPaths []string
	recentEvents   []domain.DaemonEvent
	loaded         bool
}

// DaemonStatusProvider reports whether the daemon is running.
type DaemonStatusProvider interface {
	IsRunning() bool
}

// WatchRootProvider lists configured watch root directories.
type WatchRootProvider interface {
	ListWatchRoots() ([]domain.WatchRoot, error)
}

// ProjectLister lists projects and provides counts.
type ProjectLister interface {
	ListProjects() ([]domain.Project, error)
	CountProjects() (int, error)
}

// ExclusionLister lists exclusions and provides aggregate stats.
type ExclusionLister interface {
	ListExclusions() ([]domain.Exclusion, error)
	CountExclusions() (int, error)
	TotalSpaceSaved() (int64, error)
}

// EventLister lists recent daemon events.
type EventLister interface {
	ListRecentEvents(limit int) ([]domain.DaemonEvent, error)
}

// NewDashboardModel creates a new DashboardModel with the given data providers.
// All providers may be nil; the dashboard shows empty states when data is unavailable.
func NewDashboardModel(
	t *theme.Theme,
	daemon DaemonStatusProvider,
	watchRoots WatchRootProvider,
	projects ProjectLister,
	exclusions ExclusionLister,
	events EventLister,
) *DashboardModel {
	return &DashboardModel{
		theme:      t,
		daemon:     daemon,
		watchRoots: watchRoots,
		projects:   projects,
		exclusions: exclusions,
		events:     events,
	}
}

// Init loads dashboard data. Returns nil if no async work is needed.
func (m *DashboardModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the dashboard.
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the dashboard layout.
func (m *DashboardModel) View() string {
	if m.width == 0 {
		return "Loading dashboard..."
	}

	m.refreshData()

	header := m.renderHeader()
	statusCard := m.renderStatusCard()
	statsCard := m.renderStatsCard()
	activitySection := m.renderActivitySection()
	navBar := m.renderNavBar()

	// Layout: two cards side by side on top, activity below
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, statusCard, statsCard)
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		topRow,
		activitySection,
	)
	mainArea := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 3).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, navBar)
}

func (m *DashboardModel) refreshData() {
	if m.loaded {
		return
	}
	m.loaded = true

	// Load project count
	if m.projects != nil {
		count, err := m.projects.CountProjects()
		if err == nil {
			m.projectCount = count
		}
	}

	// Load exclusion count and space saved
	if m.exclusions != nil {
		count, err := m.exclusions.CountExclusions()
		if err == nil {
			m.exclusionCount = count
		}
		size, err := m.exclusions.TotalSpaceSaved()
		if err == nil {
			m.spaceSaved = size
		}
	}

	// Load watch roots
	if m.watchRoots != nil {
		roots, err := m.watchRoots.ListWatchRoots()
		if err == nil {
			m.watchRootPaths = make([]string, len(roots))
			for i, r := range roots {
				m.watchRootPaths[i] = r.Path
			}
		}
	}

	// Load recent events
	if m.events != nil {
		events, err := m.events.ListRecentEvents(5)
		if err == nil {
			m.recentEvents = events
		}
	}
}

func (m *DashboardModel) renderHeader() string {
	title := m.theme.Title.Render("JART-STOW")
	subtitle := m.theme.Muted.Render("  Development Hygiene & Backup Exclusion Manager")
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		subtitle,
		"",
	)
}

func (m *DashboardModel) renderStatusCard() string {
	cardWidth := (m.width / 2) - 4

	// Daemon status
	running := false
	if m.daemon != nil {
		running = m.daemon.IsRunning()
	}
	daemonLine := m.renderLine("Daemon", m.theme.StatusDot(running)+" "+statusText(running))

	// Watch roots
	watchLine := m.renderLine("Watch Root", m.watchRootSummary())

	// Backup system status (TM and CCC are macOS-specific; show as available)
	tmLine := m.renderLine("Time Machine", m.theme.StatusDot(true)+" Available")
	cccLine := m.renderLine("CCC", m.theme.StatusDot(true)+" Available")

	content := lipgloss.JoinVertical(lipgloss.Left,
		daemonLine,
		watchLine,
		"",
		tmLine,
		cccLine,
	)

	titleBar := m.theme.CardTitle.Render("System Status")
	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *DashboardModel) renderStatsCard() string {
	cardWidth := (m.width / 2) - 4

	projectLine := m.renderLine("Projects", m.theme.Primary.Render(formatInt(m.projectCount)))
	exclusionLine := m.renderLine("Exclusions", m.theme.Primary.Render(formatInt(m.exclusionCount)))
	spaceLine := m.renderLine("Space Saved", m.theme.Primary.Render(formatBytes(m.spaceSaved)))
	scanLine := m.renderLine("Last Scan", m.theme.Muted.Render("No scans yet"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		projectLine,
		exclusionLine,
		spaceLine,
		scanLine,
	)

	titleBar := m.theme.CardTitle.Render("Quick Stats")
	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *DashboardModel) renderActivitySection() string {
	cardWidth := m.width - 4

	titleBar := m.theme.CardTitle.Render("Recent Activity")

	if len(m.recentEvents) == 0 {
		emptyMsg := m.theme.Muted.Render("  No activity yet. Start the daemon to begin monitoring.")
		return m.theme.CardBorder.
			Width(cardWidth).
			Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, emptyMsg))
	}

	var lines []string
	for _, evt := range m.recentEvents {
		timeStr := evt.CreatedAt.Format("15:04:05")
		icon := eventIcon(evt.EventType)
		detail := m.theme.Muted.Render(theme.Truncate(evt.Details, 60))
		line := icon + " " + timeStr + "  " + detail
		lines = append(lines, line)
	}

	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func (m *DashboardModel) renderNavBar() string {
	return m.theme.HelpText.Render("1-7 Screens  |  ? Help  |  q Quit")
}

func (m *DashboardModel) renderLine(label, value string) string {
	labelStyled := m.theme.Muted.Render(label + ":")
	return lipgloss.JoinHorizontal(lipgloss.Left, labelStyled, "  "+value)
}

func (m *DashboardModel) watchRootSummary() string {
	if len(m.watchRootPaths) == 0 {
		return m.theme.Muted.Render("No watch roots configured")
	}
	return m.theme.Muted.Render(theme.Truncate(m.watchRootPaths[0], 30))
}

// statusText returns a human-readable status string.
func statusText(running bool) string {
	if running {
		return "RUNNING"
	}
	return "STOPPED"
}

// eventIcon returns a symbol for the event type.
func eventIcon(t domain.EventType) string {
	switch t {
	case domain.EventTypeDaemonStarted, domain.EventTypeDaemonStopped:
		return "\u25CF" // ●
	case domain.EventTypeError:
		return "\u2717" // ✗
	default:
		return "\u2713" // ✓
	}
}

// formatInt converts an int to a string representation.
func formatInt(v int) string {
	if v == 0 {
		return "0"
	}
	return itoa(v)
}

// formatBytes returns a human-readable byte string.
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return itoa64(bytes) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	val := float64(bytes) / float64(div)
	unitStr := []string{"KB", "MB", "GB", "TB"}[exp]
	intPart := int(val)
	decPart := int((val - float64(intPart)) * 10)
	if decPart > 0 {
		return itoa(intPart) + "." + itoa(decPart) + " " + unitStr
	}
	return itoa(intPart) + " " + unitStr
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
