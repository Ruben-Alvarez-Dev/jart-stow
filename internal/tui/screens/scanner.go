package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/components"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type scanCompleteMsg struct {
	results []ScanResult
	err     error
}

type excludeDoneMsg struct {
	success int
	failed  int
	err     error
}

type ScannerModel struct {
	theme        *theme.Theme
	watchRoots   WatchRootProvider
	scanEngine   ScreenScanEngine
	quickExclude ScreenQuickExclude

	width, height int
	focus         int // 0 = browser, 1 = results
	navRequest    string

	entries       []Volume
	breadcrumb    []string
	currentPath   string
	browserCursor int
	entriesLoaded bool
	browserErr    string

	scanning     bool
	spinner      spinner.Model
	results      []ScanResult
	resultsTable table.Model
	scanPath     string
	scanErr      string

	selected    map[int]bool
	allSelected bool

	excluding      bool
	excludeSuccess int
	excludeFailed  int
	excludeErr     string

	// Available content height inside results card (set during View, used by rebuildResultsTable)
	resultsContentH int
}

func NewScannerModel(t *theme.Theme, watchRoots WatchRootProvider, scanEngine ScreenScanEngine, quickExclude ScreenQuickExclude) *ScannerModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	return &ScannerModel{
		theme:        t,
		watchRoots:   watchRoots,
		scanEngine:   scanEngine,
		quickExclude: quickExclude,
		spinner:      sp,
		selected:     make(map[int]bool),
	}
}

func (m *ScannerModel) Init() tea.Cmd { return nil }

func (m *ScannerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if len(m.results) > 0 {
			m.rebuildResultsTable()
		}
	case spinner.TickMsg:
		if m.scanning || m.excluding {
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
		m.selected = make(map[int]bool)
		m.allSelected = false
		m.rebuildResultsTable()
	case excludeDoneMsg:
		m.excluding = false
		if msg.err != nil {
			m.excludeErr = msg.err.Error()
		} else {
			m.excludeSuccess = msg.success
			m.excludeFailed = msg.failed
			m.excludeErr = ""
		}
		if m.scanPath != "" {
			m.scanning = true
			return m, tea.Batch(m.spinner.Tick, m.runScan(m.scanPath))
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 2
		case "esc", "backspace":
			m.navRequest = "back"
		case "q":
			m.navRequest = "quit"
		case "left", "h":
			return m, m.handleBack()
		case "right", "l":
			return m, m.handleBrowseInto()
		case "up", "k":
			return m, m.handleUp()
		case "down", "j":
			return m, m.handleDown()
		case "enter":
			return m, m.handleEnter()
		case " ":
			if m.focus == 1 && len(m.results) > 0 {
				idx := m.resultsTable.Cursor()
				if idx >= 0 && idx < len(m.results) {
					m.selected[idx] = !m.selected[idx]
					m.allSelected = false
					m.rebuildResultsTable()
				}
			}
		case "a":
			if len(m.results) > 0 {
				m.allSelected = !m.allSelected
				for i := range m.results {
					m.selected[i] = m.allSelected
				}
				m.rebuildResultsTable()
			}
		default:
			if m.focus == 1 && m.resultsTable.Width() > 0 {
				var cmd tea.Cmd
				m.resultsTable, cmd = m.resultsTable.Update(msg)
				return m, cmd
			}
		}
	default:
		if m.focus == 1 && m.resultsTable.Width() > 0 {
			var cmd tea.Cmd
			m.resultsTable, cmd = m.resultsTable.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *ScannerModel) handleEnter() tea.Cmd {
	if m.focus == 0 {
		if !m.entriesLoaded {
			m.loadVolumes()
			return nil
		}
		if m.browserCursor < 0 || m.browserCursor >= len(m.entries) {
			return nil
		}
		entry := m.entries[m.browserCursor]
		if !entry.IsDir || m.scanEngine == nil {
			return nil
		}
		m.scanning = true
		m.scanErr = ""
		m.scanPath = entry.Path
		m.results = nil
		m.resultsTable = table.Model{}
		m.excludeSuccess = 0
		m.excludeFailed = 0
		m.excludeErr = ""
		m.selected = make(map[int]bool)
		m.allSelected = false
		return tea.Batch(m.spinner.Tick, m.runScan(entry.Path))
	}
	if m.focus == 1 && len(m.results) > 0 {
		paths := m.getSelectedPaths()
		if len(paths) == 0 {
			return nil
		}
		m.excluding = true
		m.excludeErr = ""
		return tea.Batch(m.spinner.Tick, m.runExclude(paths))
	}
	return nil
}

func (m *ScannerModel) handleBack() tea.Cmd {
	if !m.entriesLoaded || len(m.breadcrumb) == 0 {
		return nil
	}
	m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
	if len(m.breadcrumb) == 0 {
		m.currentPath = ""
		m.loadVolumes()
	} else {
		if idx := lastSlashIndex(m.currentPath); idx > 0 {
			parentPath := m.currentPath[:idx]
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
	}
	return nil
}

func (m *ScannerModel) handleBrowseInto() tea.Cmd {
	if m.focus != 0 || !m.entriesLoaded {
		return nil
	}
	if m.browserCursor < 0 || m.browserCursor >= len(m.entries) {
		return nil
	}
	entry := m.entries[m.browserCursor]
	if !entry.IsDir || m.scanEngine == nil {
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

func (m *ScannerModel) runScan(path string) tea.Cmd {
	return func() tea.Msg {
		results, err := m.scanEngine.ScanPath(path)
		return scanCompleteMsg{results: results, err: err}
	}
}

func (m *ScannerModel) getSelectedPaths() []string {
	hasSelection := false
	for _, sel := range m.selected {
		if sel {
			hasSelection = true
			break
		}
	}
	if !hasSelection {
		paths := make([]string, len(m.results))
		for i, r := range m.results {
			paths[i] = r.Path
		}
		return paths
	}
	var paths []string
	for i, r := range m.results {
		if m.selected[i] {
			paths = append(paths, r.Path)
		}
	}
	return paths
}

func (m *ScannerModel) runExclude(paths []string) tea.Cmd {
	return func() tea.Msg {
		failures := m.quickExclude.ExcludePaths(paths)
		success := len(paths) - len(failures)
		return excludeDoneMsg{success: success, failed: len(failures), err: nil}
	}
}

func (m *ScannerModel) rebuildResultsTable() {
	if len(m.results) == 0 {
		m.resultsTable = table.Model{}
		return
	}
	// Use stored results content height if set, otherwise calculate default
	contentH := m.resultsContentH
	if contentH < 2 {
		s := components.NewScreen(m.theme, m.width, m.height, "", "")
		contentH = s.CardContentHeight(5)
	}
	maxDataRows := contentH - 4 // count(1) + blank(1) + table_header(1) + table_sep(1)
	if maxDataRows < 1 {
		maxDataRows = 1
	}
	if maxDataRows > len(m.results) {
		maxDataRows = len(m.results)
	}

	panelWidth := (m.width*2)/3 - 6
	if panelWidth < 30 {
		panelWidth = 30
	}
	selW, patW, szW := 3, 14, 10
	pathW := panelWidth - selW - patW - szW - 3
	if pathW < 8 {
		pathW = 8
	}

	columns := []table.Column{
		{Title: "Sel", Width: selW},
		{Title: "Path", Width: pathW},
		{Title: "Pattern", Width: patW},
		{Title: "Size", Width: szW},
	}
	var rows []table.Row
	for i, r := range m.results {
		chk := " "
		if m.selected[i] {
			chk = "✓"
		}
		rows = append(rows, table.Row{chk, r.Path, r.PatternName, formatBytes(r.SizeBytes)})
	}

	tbl := table.New(table.WithColumns(columns), table.WithRows(rows), table.WithHeight(maxDataRows), table.WithFocused(true))
	s2 := table.DefaultStyles()
	s2.Header = s2.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8")).BorderBottom(true).Bold(true).Foreground(lipgloss.Color("6"))
	s2.Selected = s2.Selected.Foreground(lipgloss.Color("6")).Background(lipgloss.Color("235")).Bold(false)
	s2.Cell = s2.Cell.Foreground(lipgloss.Color("15"))
	tbl.SetStyles(s2)
	m.resultsTable = tbl
}

// ============================================================================
// View — uses components.Screen for viewport-safe layout
// ============================================================================

func (m *ScannerModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	s := components.NewScreen(m.theme, m.width, m.height,
		"SCANNER", "Browse, scan, and apply backup exclusions")

	panels := m.renderPanels() // no action bar needed — space in nav
	navText := m.buildNavText()

	return s.Render(panels, "", navText)
}

func (m *ScannerModel) renderPanels() string {
	fullW := m.width - 2
	if fullW < 20 {
		fullW = 20
	}

	// No action bar — use full ContentHeight for panels
	s := components.NewScreen(m.theme, m.width, m.height, "", "")
	panelsMaxH := s.ContentHeight() // all space between header and nav

	if len(m.results) > 0 {
		// Results only: browser hidden, results get full panel height
		m.resultsContentH = panelsMaxH - 5 // card chrome
		if m.resultsContentH < 4 {
			m.resultsContentH = 4
		}
		m.rebuildResultsTable()
		return m.buildResultsPanel(fullW, m.resultsContentH)
	}

	// No results: side by side
	leftW := m.width / 3
	if leftW < 20 {
		leftW = 20
	}
	rightW := m.width - leftW - 2
	if rightW < 20 {
		rightW = 20
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.buildBrowserPanel(leftW, panelsMaxH),
		" ",
		m.buildResultsPanel(rightW, panelsMaxH),
	)
}

func (m *ScannerModel) buildBrowserPanel(width int, contentMaxH int) string {
	card := components.NewCard(m.theme, "Volume Browser", width)

	var lines []string
	if len(m.breadcrumb) == 0 {
		lines = append(lines, m.theme.Muted.Render("/ (volumes)"))
	} else {
		parts := append([]string{"/"}, m.breadcrumb...)
		lines = append(lines, m.theme.Primary.Render(joinPath(parts...)))
	}
	lines = append(lines, "")

	if m.browserErr != "" {
		lines = append(lines, m.theme.ErrorText.Render(m.browserErr))
	} else if !m.entriesLoaded {
		lines = append(lines, m.theme.Muted.Render("Press Enter to load volumes."))
	} else if len(m.entries) == 0 {
		lines = append(lines, m.theme.Muted.Render("(empty)"))
	} else {
		// Use contentMaxH for the available content height inside the card
		// (or calculate from Screen if contentMaxH is not explicitly limited)
		avail := contentMaxH - 2 // breadcrumb(1) + blank(1)
		if avail < 1 {
			avail = 1
		}
		// Also limit by card content height from screen
		s := components.NewScreen(m.theme, m.width, m.height, "", "")
		cardMax := s.CardContentHeight(0) // assume no actions for browser height
		if avail > cardMax {
			avail = cardMax
		}
		maxEntries := avail
		showCount := len(m.entries)
		if showCount > maxEntries {
			showCount = maxEntries
		}
		for i := 0; i < showCount; i++ {
			entry := m.entries[i]
			name := theme.Truncate(entry.Name, width-8)
			if i == m.browserCursor && m.focus == 0 {
				lines = append(lines, m.theme.SelectedRow.Render("▸ [D] "+name))
			} else {
				lines = append(lines, "  [D] "+m.theme.Muted.Render(name))
			}
		}
		if len(m.entries) > maxEntries {
			lines = append(lines, m.theme.Muted.Render("  ... and "+itoa(len(m.entries)-maxEntries)+" more"))
		}
	}

	return card.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *ScannerModel) buildResultsPanel(width int, _ int) string {
	card := components.NewCard(m.theme, "Scan Results", width)

	var content string
	if m.scanning {
		content = m.spinner.View() + " Scanning..."
	} else if m.excluding {
		content = m.spinner.View() + " Applying exclusions..."
	} else if m.scanErr != "" {
		content = m.theme.ErrorText.Render("Error: " + m.scanErr)
	} else if m.excludeErr != "" {
		content = m.theme.ErrorText.Render("Error: " + m.excludeErr)
	} else if m.excludeSuccess > 0 || m.excludeFailed > 0 {
		content = m.theme.Primary.Render("✅ " + itoa(m.excludeSuccess) + " exc, ✗ " + itoa(m.excludeFailed) + " fail")
		if len(m.results) > 0 {
			content += "\n\n" + m.resultsTable.View()
		}
	} else if len(m.results) == 0 {
		if m.scanPath != "" {
			content = m.theme.Muted.Render("No artifacts found.")
		} else {
			content = m.theme.Muted.Render("Browse to a dir and press Enter.")
		}
	} else {
		selCount := 0
		for _, v := range m.selected {
			if v {
				selCount++
			}
		}
		label := itoa(len(m.results)) + " artifacts"
		if selCount > 0 && selCount < len(m.results) {
			label += " (" + itoa(selCount) + " sel)"
		} else if selCount == len(m.results) && len(m.results) > 0 {
			label += " (all)"
		}
		content = m.theme.Primary.Render(label) + "\n\n" + m.resultsTable.View()
	}

	return card.Render(content)
}

func (m *ScannerModel) buildActionText() string {
	if len(m.results) == 0 {
		return ""
	}
	selCount := 0
	for _, v := range m.selected {
		if v {
			selCount++
		}
	}
	text := "[Space] Toggle"
	if selCount == 0 {
		text += " | [a] All"
	} else {
		text += " | [a] None"
	}
	if selCount > 0 {
		text += " | [Enter] Exc " + itoa(selCount)
	} else {
		text += " | [Enter] Exc all " + itoa(len(m.results))
	}
	return components.ActionBar(m.theme, m.width, text)
}

func (m *ScannerModel) buildNavText() string {
	if len(m.results) > 0 {
		return "Tab ↹ | ↑↓ Nav | ␣ Sel | a All | ↵ Apply | Esc Menu | q Quit"
	}
	return "Tab ↹ | ↑↓ Nav | → Brws | ← Back | ↵ Scan | Esc Menu | q Quit"
}

func (m *ScannerModel) NavRequest() string   { return m.navRequest }
func (m *ScannerModel) NavTarget() tea.Model { return nil }
func (m *ScannerModel) ClearNav()            { m.navRequest = "" }

func lastSlashIndex(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

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
