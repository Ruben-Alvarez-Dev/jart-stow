package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Scanner Screen
// ============================================================================

// scanCompleteMsg carries the result of an async scan operation.
type scanCompleteMsg struct {
	results []ScanResult
	err     error
}

// ScannerModel displays a volume/directory browser on the left and scan results
// on the right. It supports browsing filesystem directories, triggering scans,
// and viewing discovered artifacts in an interactive table.
type ScannerModel struct {
	theme      *theme.Theme
	watchRoots WatchRootProvider
	scanEngine ScreenScanEngine

	width  int
	height int

	focus int // 0 = browser (left), 1 = results (right)

	navRequest string

	// ── Browser state ──────────────────────────────────────────────────
	entries       []Volume // current-level directory entries
	breadcrumb    []string // path segments for breadcrumb display
	currentPath   string   // full path of the directory being viewed
	browserCursor int
	entriesLoaded bool
	browserErr    string

	// ── Scan state ─────────────────────────────────────────────────────
	scanning     bool
	spinner      spinner.Model
	results      []ScanResult
	resultsTable table.Model
	scanPath     string
	scanErr      string
}

// NewScannerModel creates a new ScannerModel with volume browsing and scan
// capabilities. All providers may be nil; the screen shows appropriate empty
// states when data is unavailable.
func NewScannerModel(t *theme.Theme, watchRoots WatchRootProvider, scanEngine ScreenScanEngine) *ScannerModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	return &ScannerModel{
		theme:      t,
		watchRoots: watchRoots,
		scanEngine: scanEngine,
		spinner:    sp,
	}
}

// Init initializes the scanner screen. Returns nil; the screen is synchronous
// until the user triggers an action.
func (m *ScannerModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the scanner screen, including keyboard input,
// window resizes, spinner ticks, and async scan completion.
func (m *ScannerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		if m.scanning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case scanCompleteMsg:
		m.scanning = false
		if msg.err != nil {
			m.scanErr = msg.err.Error()
			m.results = nil
		} else {
			m.scanErr = ""
			m.results = msg.results
		}
		m.rebuildResultsTable()

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 2

		case "esc", "backspace":
			return m, m.handleBack()

		case "q":
			m.navRequest = "quit"

		case "up", "k":
			return m, m.handleUp()

		case "down", "j":
			return m, m.handleDown()

		case "enter":
			return m, m.handleEnter()

		case "right", "l":
			return m, m.handleBrowseInto()

		default:
			// When results panel is focused, delegate to table
			if m.focus == 1 && m.resultsTable.Width() > 0 {
				var cmd tea.Cmd
				m.resultsTable, cmd = m.resultsTable.Update(msg)
				return m, cmd
			}
		}

	default:
		// Forward unknown messages to the table when focused
		if m.focus == 1 && m.resultsTable.Width() > 0 {
			var cmd tea.Cmd
			m.resultsTable, cmd = m.resultsTable.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// handleEnter processes the Enter key based on context:
//   - Browser not loaded: loads volumes from the scan engine.
//   - Browser loaded and entry selected: triggers an async scan of the path.
func (m *ScannerModel) handleEnter() tea.Cmd {
	if m.focus != 0 {
		return nil
	}

	if !m.entriesLoaded {
		m.loadVolumes()
		return nil
	}

	if m.browserCursor < 0 || m.browserCursor >= len(m.entries) {
		return nil
	}

	entry := m.entries[m.browserCursor]
	if !entry.IsDir {
		return nil
	}

	if m.scanEngine == nil {
		m.scanErr = "Scan engine not connected"
		return nil
	}

	m.scanning = true
	m.scanErr = ""
	m.scanPath = entry.Path
	m.results = nil
	m.resultsTable = table.Model{}

	return tea.Batch(
		m.spinner.Tick,
		m.runScan(entry.Path),
	)
}

// handleBrowseInto browses into the selected directory by calling
// BrowseDirectory and updating the breadcrumb.
func (m *ScannerModel) handleBrowseInto() tea.Cmd {
	if m.focus != 0 || !m.entriesLoaded {
		return nil
	}
	if m.browserCursor < 0 || m.browserCursor >= len(m.entries) {
		return nil
	}

	entry := m.entries[m.browserCursor]
	if !entry.IsDir {
		return nil
	}

	if m.scanEngine == nil {
		return nil
	}

	children, err := m.scanEngine.BrowseDirectory(entry.Path)
	if err != nil {
		m.browserErr = err.Error()
		return nil
	}

	m.breadcrumb = append(m.breadcrumb, entry.Name)
	m.currentPath = entry.Path
	m.entries = children
	m.browserCursor = 0
	m.browserErr = ""
	return nil
}

// handleBack navigates one level up in the directory browser.
// If already at the root level, requests navigation back to the main menu.
func (m *ScannerModel) handleBack() tea.Cmd {
	if !m.entriesLoaded || len(m.breadcrumb) == 0 {
		m.navRequest = "back"
		return nil
	}

	// Pop breadcrumb
	m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]

	// Determine parent path
	var parentPath string
	if len(m.breadcrumb) == 0 {
		// Back to volumes root
		m.currentPath = ""
		m.loadVolumes()
	} else {
		// Back to previous directory — we need to reconstruct the path
		// Use the first volume's path as a heuristic: rebuild parent from currentPath
		if idx := lastSlashIndex(m.currentPath); idx > 0 {
			parentPath = m.currentPath[:idx]
		} else {
			parentPath = "/"
		}
		m.currentPath = parentPath
		if m.scanEngine != nil {
			children, err := m.scanEngine.BrowseDirectory(parentPath)
			if err != nil {
				m.browserErr = err.Error()
			} else {
				m.entries = children
				m.browserCursor = 0
				m.browserErr = ""
			}
		}
	}

	return nil
}

// handleUp moves the cursor up in the active panel.
func (m *ScannerModel) handleUp() tea.Cmd {
	if m.focus == 0 {
		if m.browserCursor > 0 {
			m.browserCursor--
		}
	} else {
		m.resultsTable, _ = m.resultsTable.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	return nil
}

// handleDown moves the cursor down in the active panel.
func (m *ScannerModel) handleDown() tea.Cmd {
	if m.focus == 0 {
		if m.browserCursor < len(m.entries)-1 {
			m.browserCursor++
		}
	} else {
		m.resultsTable, _ = m.resultsTable.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	return nil
}

// loadVolumes loads the root volume list from the scan engine.
func (m *ScannerModel) loadVolumes() {
	if m.scanEngine == nil {
		m.browserErr = "Scan engine not connected"
		return
	}
	volumes, err := m.scanEngine.ListVolumes()
	if err != nil {
		m.browserErr = err.Error()
		return
	}
	m.entries = volumes
	m.entriesLoaded = true
	m.browserCursor = 0
	m.browserErr = ""
	m.breadcrumb = nil
	m.currentPath = ""
}

// runScan returns a tea.Cmd that executes the scan asynchronously.
func (m *ScannerModel) runScan(path string) tea.Cmd {
	return func() tea.Msg {
		results, err := m.scanEngine.ScanPath(path)
		return scanCompleteMsg{results: results, err: err}
	}
}

// rebuildResultsTable creates a new bubbles table from the current results.
func (m *ScannerModel) rebuildResultsTable() {
	if len(m.results) == 0 {
		m.resultsTable = table.Model{}
		return
	}

	panelWidth := m.width - (m.width / 3) - 10
	if panelWidth < 40 {
		panelWidth = 40
	}

	pathWidth := panelWidth - 30
	if pathWidth < 10 {
		pathWidth = 10
	}

	columns := []table.Column{
		{Title: "Path", Width: pathWidth},
		{Title: "Pattern", Width: 16},
		{Title: "Size", Width: 10},
	}

	var rows []table.Row
	for _, r := range m.results {
		rows = append(rows, table.Row{
			r.Path,
			r.PatternName,
			formatBytes(r.SizeBytes),
		})
	}

	maxHeight := m.height - 10
	if maxHeight < 4 {
		maxHeight = 4
	}
	rowCount := len(rows)
	if rowCount > maxHeight {
		rowCount = maxHeight
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(rowCount),
		table.WithFocused(true),
	)

	// Style the table using theme colors
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
	m.resultsTable = tbl
}

// ============================================================================
// Rendering
// ============================================================================

// View renders the scanner screen layout: header, browser panel, results
// panel, and navigation bar.
func (m *ScannerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	panels := m.renderPanels()
	navBar := m.renderNavBar()

	contentHeight := m.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, panels)
	mainArea := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, navBar)
}

func (m *ScannerModel) renderHeader() string {
	title := m.theme.ScreenTitle.Render("SCANNER")
	subtitle := m.theme.Muted.Render("Browse volumes and scan for development artifacts")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "")
}

func (m *ScannerModel) renderPanels() string {
	leftWidth := m.width / 3
	if leftWidth < 25 {
		leftWidth = 25
	}
	rightWidth := m.width - leftWidth - 4
	if rightWidth < 25 {
		rightWidth = 25
	}

	leftPanel := m.renderBrowserPanel(leftWidth)
	rightPanel := m.renderResultsPanel(rightWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m *ScannerModel) renderBrowserPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Volume Browser")

	// Breadcrumb
	var breadcrumbLine string
	if len(m.breadcrumb) == 0 {
		breadcrumbLine = m.theme.Muted.Render("/ (volumes)")
	} else {
		parts := append([]string{"/"}, m.breadcrumb...)
		breadcrumbLine = lipgloss.JoinHorizontal(lipgloss.Left,
			m.theme.Primary.Render(joinPath(parts...)),
		)
	}
	breadcrumbLine = m.theme.Muted.Render("  ") + breadcrumbLine

	var lines []string
	lines = append(lines, breadcrumbLine, "")

	// Error
	if m.browserErr != "" {
		lines = append(lines, m.theme.ErrorText.Render("  "+m.browserErr))
	} else if !m.entriesLoaded {
		lines = append(lines,
			m.theme.Muted.Render("  Press Enter to load volumes."),
			"",
			m.theme.Muted.Render("  Enter  = load / scan"),
			m.theme.Muted.Render("  Right  = browse into"),
			m.theme.Muted.Render("  Esc    = go back"),
		)
	} else if len(m.entries) == 0 {
		lines = append(lines, m.theme.Muted.Render("  (empty directory)"))
	} else {
		// Directory entries
		for i, entry := range m.entries {
			prefix := "  "
			if i == m.browserCursor && m.focus == 0 {
				prefix = m.theme.SelectedRow.Render("> ")
			} else {
				prefix = "  "
			}
			icon := "[D]"
			if !entry.IsDir {
				icon = "[F]"
			}
			name := theme.Truncate(entry.Name, width-10)
			line := prefix + icon + " " + name

			if i == m.browserCursor && m.focus == 0 {
				line = m.theme.SelectedRow.Render(prefix + icon + " " + name)
			} else {
				line = prefix + icon + " " + m.theme.Muted.Render(name)
			}
			lines = append(lines, line)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	border := m.theme.CardBorder
	if m.focus == 0 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *ScannerModel) renderResultsPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Scan Results")

	var content string

	if m.scanning {
		spin := m.spinner.View() + " Scanning " + theme.Truncate(m.scanPath, width-20) + "..."
		content = spin
	} else if m.scanErr != "" {
		content = m.theme.ErrorText.Render("  Error: " + m.scanErr)
	} else if len(m.results) == 0 {
		if m.scanPath != "" {
			content = m.theme.Muted.Render("  Scan of " + theme.Truncate(m.scanPath, width-20) + "\n  produced no results.")
		} else {
			content = m.theme.Muted.Render("  No scan results yet.\n  Browse to a directory and press Enter to scan.")
		}
	} else {
		// Show summary + table
		summary := m.theme.Primary.Render("  " + itoa(len(m.results)) + " artifacts found in " + theme.Truncate(m.scanPath, width-30))
		tableContent := m.resultsTable.View()
		content = lipgloss.JoinVertical(lipgloss.Left, summary, "", tableContent)
	}

	border := m.theme.CardBorder
	if m.focus == 1 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *ScannerModel) renderNavBar() string {
	return m.theme.NavBar.Width(m.width).Padding(0, 1).Render(
		"Tab: Switch  |  Up/Down: Navigate  |  Enter: Load/Scan  |  Right: Browse  |  Esc: Back  |  q: Quit",
	)
}

// ============================================================================
// Navigation interface
// ============================================================================

// NavRequest returns the pending navigation request, or "" if none.
func (m *ScannerModel) NavRequest() string { return m.navRequest }

// NavTarget returns the target model for navigation (nil for back/quit).
func (m *ScannerModel) NavTarget() tea.Model { return nil }

// ClearNav clears the navigation request.
func (m *ScannerModel) ClearNav() { m.navRequest = "" }

// ============================================================================
// Helpers
// ============================================================================

// stringOfChar creates a string of n repeated characters.
func stringOfChar(c byte, n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = c
	}
	return string(buf)
}

// lastSlashIndex returns the index of the last '/' in s, or -1 if not found.
func lastSlashIndex(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// joinPath joins path segments with '/'.
func joinPath(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	result := segments[0]
	for i := 1; i < len(segments); i++ {
		if result[len(result)-1] != '/' {
			result += "/"
		}
		result += segments[i]
	}
	return result
}
