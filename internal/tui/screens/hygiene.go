package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/components"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Hygiene Screen
// ============================================================================

// JunkLister provides read-only access to junk categories and items for the
// hygiene screen.
type JunkLister interface {
	ListCategories() ([]domain.JunkCategory, error)
	ListPendingItemsByCategory(categoryID int64) ([]domain.JunkItem, error)
	CountPendingItems() (int, error)
}

// catScanCompleteMsg carries the result of an async category scan.
type catScanCompleteMsg struct {
	categoryID int64
	items      []domain.JunkItem
	err        error
}

// HygieneModel displays junk categories in a sidebar and items for the selected
// category in an interactive table. Users can scan categories, review
// discovered items, and approve or skip them.
type HygieneModel struct {
	theme      *theme.Theme
	junk       JunkLister
	junkRunner ScreenJunkScanRunner

	width  int
	height int

	focus       int // 0 = categories sidebar, 1 = items panel
	cursor      int
	selectedCat int
	categories  []domain.JunkCategory
	items       []domain.JunkItem
	loaded      bool

	// Scanning state
	scanning     bool
	scannerCatID int64 // which category is being scanned
	spinner      spinner.Model
	scanErr      string

	// Items table
	itemsTable table.Model

	navRequest string
}

// NewHygieneModel creates a new HygieneModel with junk listing and scan
// capabilities. Providers may be nil; the screen shows appropriate empty
// states when data is unavailable.
func NewHygieneModel(t *theme.Theme, junk JunkLister, junkRunner ScreenJunkScanRunner) *HygieneModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	return &HygieneModel{
		theme:      t,
		junk:       junk,
		junkRunner: junkRunner,
		spinner:    sp,
	}
}

// Init initializes the hygiene screen. Returns nil; data is loaded on first
// render.
func (m *HygieneModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the hygiene screen, including keyboard input,
// window resizes, spinner ticks, and async scan completion.
func (m *HygieneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case catScanCompleteMsg:
		m.scanning = false
		if msg.err != nil {
			m.scanErr = msg.err.Error()
		} else {
			m.scanErr = ""
			// Find the category and update its items
			for i := range m.categories {
				if m.categories[i].ID == msg.categoryID {
					m.selectedCat = i
					break
				}
			}
			m.items = msg.items
			m.rebuildItemsTable()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 2
			m.cursor = 0

		case "up", "k":
			m.handleUp()

		case "down", "j":
			m.handleDown()

		case "enter":
			if m.focus == 0 && m.cursor < len(m.categories) {
				return m, m.startCategoryScan(m.categories[m.cursor])
			}

		case "a":
			return m, m.handleApproveSingle()

		case "s":
			return m, m.handleSkipSingle()

		case "A":
			return m, m.handleBatchApprove()

		case "S":
			return m, m.handleBatchSkip()

		case "esc", "backspace":
			m.navRequest = "back"

		case "q":
			m.navRequest = "quit"
		}
	}

	return m, nil
}

// handleUp moves the cursor up in the active panel.
func (m *HygieneModel) handleUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// handleDown moves the cursor down in the active panel.
func (m *HygieneModel) handleDown() {
	if m.focus == 0 {
		if m.cursor < len(m.categories)-1 {
			m.cursor++
		}
	} else {
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	}
}

// startCategoryScan triggers an async scan of a junk category, showing a
// spinner while the scan runs.
func (m *HygieneModel) startCategoryScan(cat domain.JunkCategory) tea.Cmd {
	if m.junkRunner == nil {
		m.scanErr = "Scan runner not connected"
		return nil
	}

	m.scanning = true
	m.scanErr = ""
	m.scannerCatID = cat.ID

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			items, err := m.junkRunner.ScanCategory(cat)
			return catScanCompleteMsg{categoryID: cat.ID, items: items, err: err}
		},
	)
}

// handleApproveSingle approves the currently selected item.
func (m *HygieneModel) handleApproveSingle() tea.Cmd {
	if m.focus != 1 || m.junkRunner == nil {
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	item := m.items[m.cursor]
	if err := m.junkRunner.ApproveItem(item.ID); err != nil {
		return nil
	}
	m.refreshItems()
	return nil
}

// handleSkipSingle skips the currently selected item.
func (m *HygieneModel) handleSkipSingle() tea.Cmd {
	if m.focus != 1 || m.junkRunner == nil {
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	item := m.items[m.cursor]
	if err := m.junkRunner.SkipItem(item.ID); err != nil {
		return nil
	}
	m.refreshItems()
	return nil
}

// handleBatchApprove approves all visible items in the current category.
func (m *HygieneModel) handleBatchApprove() tea.Cmd {
	if m.focus != 1 || m.junkRunner == nil || len(m.items) == 0 {
		return nil
	}
	ids := make([]int64, len(m.items))
	for i, item := range m.items {
		ids[i] = item.ID
	}
	if err := m.junkRunner.BatchApprove(ids); err != nil {
		return nil
	}
	m.refreshItems()
	return nil
}

// handleBatchSkip skips all visible items in the current category.
func (m *HygieneModel) handleBatchSkip() tea.Cmd {
	if m.focus != 1 || m.junkRunner == nil || len(m.items) == 0 {
		return nil
	}
	ids := make([]int64, len(m.items))
	for i, item := range m.items {
		ids[i] = item.ID
	}
	if err := m.junkRunner.BatchSkip(ids); err != nil {
		return nil
	}
	m.refreshItems()
	return nil
}

// refreshItems reloads items for the currently selected category.
func (m *HygieneModel) refreshItems() {
	if m.junk == nil || m.selectedCat >= len(m.categories) {
		return
	}
	cat := m.categories[m.selectedCat]
	items, err := m.junk.ListPendingItemsByCategory(cat.ID)
	if err == nil {
		m.items = items
	} else {
		m.items = nil
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
	m.rebuildItemsTable()
}

// rebuildItemsTable creates a new bubbles table from the current items.
func (m *HygieneModel) rebuildItemsTable() {
	if len(m.items) == 0 {
		m.itemsTable = table.Model{}
		return
	}

	panelWidth := m.width - (m.width / 3) - 10
	if panelWidth < 30 {
		panelWidth = 30
	}

	pathWidth := panelWidth - 45
	if pathWidth < 12 {
		pathWidth = 12
	}

	columns := []table.Column{
		{Title: "Path", Width: pathWidth},
		{Title: "Description", Width: 20},
		{Title: "Size", Width: 10},
		{Title: "Status", Width: 10},
	}

	var rows []table.Row
	for _, item := range m.items {
		status := verificationStatusTextShort(item.VerifiedByUser)
		if item.IsCleaned() {
			status = "cleaned"
		}
		rows = append(rows, table.Row{
			item.Path,
			theme.Truncate(item.Description, 18),
			formatBytes(item.SizeBytes),
			status,
		})
	}

	maxHeight := m.height - 12
	if maxHeight < 4 {
		maxHeight = 4
	}
	rowCount := len(rows)
	if rowCount < 1 {
		rowCount = 1
	}
	if rowCount > maxHeight {
		rowCount = maxHeight
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(rowCount),
		table.WithFocused(false),
	)

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
	m.itemsTable = tbl
}

// ============================================================================
// Rendering
// ============================================================================

// View renders the hygiene screen: header, categories sidebar, items panel,
// detail panel, and navigation bar.
func (m *HygieneModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	m.loadData()

	// Pending count in subtitle
	subtitle := ""
	if m.junk != nil {
		if count, err := m.junk.CountPendingItems(); err == nil && count > 0 {
			subtitle = itoa(count) + " pending"
		}
	}
	s := components.NewScreen(m.theme, m.width, m.height, "HYGIENE", subtitle)

	panels := lipgloss.JoinVertical(lipgloss.Left, m.renderPanels(), m.renderDetailPanel())
	return s.Render(panels, "", "← Esc:Back  ·  q:Quit")
}

func (m *HygieneModel) loadData() {
	if m.loaded {
		return
	}
	m.loaded = true

	if m.junk != nil {
		cats, err := m.junk.ListCategories()
		if err == nil {
			m.categories = cats
			if len(cats) > 0 {
				// Load items for first category
				items, err := m.junk.ListPendingItemsByCategory(cats[0].ID)
				if err == nil {
					m.items = items
					m.rebuildItemsTable()
				}
			}
		}
	}
}

func (m *HygieneModel) renderHeader() string {
	title := m.theme.ScreenTitle.Render("HYGIENE")
	var pendingCount int
	if m.junk != nil {
		count, err := m.junk.CountPendingItems()
		if err == nil {
			pendingCount = count
		}
	}
	subtitle := m.theme.Muted.Render("Pending: " + itoa(pendingCount) + " items")
	return lipgloss.JoinHorizontal(lipgloss.Left, title, "  "+subtitle)
}

func (m *HygieneModel) renderPanels() string {
	leftWidth := m.width / 3
	if leftWidth < 20 {
		leftWidth = 20
	}
	rightWidth := m.width - leftWidth - 4
	if rightWidth < 30 {
		rightWidth = 30
	}

	leftPanel := m.renderCategoriesSidebar(leftWidth)
	rightPanel := m.renderItemsPanel(rightWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m *HygieneModel) renderCategoriesSidebar(width int) string {
	titleBar := m.theme.CardTitle.Render("Categories")

	var lines []string

	if m.scanning {
		spinLine := m.spinner.View() + " Scanning..."
		for _, cat := range m.categories {
			if cat.ID == m.scannerCatID {
				scanName := cat.Name
				if len(scanName) > 25 {
					scanName = scanName[:22] + "..."
				}
				spinLine = m.spinner.View() + " " + scanName
				break
			}
		}
		lines = append(lines, m.theme.Muted.Render("  "+spinLine))
		lines = append(lines, "")
	}

	if m.scanErr != "" {
		lines = append(lines, m.theme.ErrorText.Render("  "+m.scanErr))
		lines = append(lines, "")
	}

	if len(m.categories) > 0 {
		for i, cat := range m.categories {
			enabledIcon := "[ ]"
			if cat.Enabled {
				enabledIcon = m.theme.Success.Render("[*]")
			}

			scannerIcon := scannerEmoji(cat.Scanner)
			entryLine := enabledIcon + " " + scannerIcon + " " + theme.Truncate(cat.Name, width-18)

			if m.focus == 0 && i == m.cursor {
				lines = append(lines, m.theme.SelectedRow.Render(entryLine))
			} else {
				lines = append(lines, "  "+entryLine)
			}
		}
	} else {
		if m.junk == nil {
			lines = append(lines, m.theme.Muted.Render("  Junk data not available."))
		} else {
			lines = append(lines, m.theme.Muted.Render("  No categories."))
		}
	}

	lines = append(lines, "")
	lines = append(lines, m.theme.Muted.Render("  Enter = scan category"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	border := m.theme.CardBorder
	if m.focus == 0 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *HygieneModel) renderItemsPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Items to Review")

	var content string

	if m.scanning {
		content = m.spinner.View() + " Scanning category..."
	} else if len(m.items) == 0 {
		if m.junk == nil || len(m.categories) == 0 {
			content = m.theme.Muted.Render("  No junk items found. Scan categories to discover junk.")
		} else if m.selectedCat < len(m.categories) {
			catName := m.categories[m.selectedCat].Name
			content = m.theme.Muted.Render("  No items for " + catName + ".\n  Press Enter on a category to scan.")
		} else {
			content = m.theme.Muted.Render("  Select a category on the left to view items.")
		}
	} else {
		// Summary
		var totalSize int64
		for _, item := range m.items {
			totalSize += item.SizeBytes
		}
		summary := m.theme.Primary.Render(
			itoa(len(m.items)) + " items  |  " + formatBytes(totalSize),
		)

		// Show cursor indicator in table by setting cursor and rendering
		// We use the table but override its cursor with ours when focused
		if m.focus == 1 && m.cursor >= 0 && m.cursor < len(m.items) {
			m.itemsTable.SetCursor(m.cursor)
		} else {
			m.itemsTable.SetCursor(-1) // hide cursor when not focused
		}

		tableContent := m.itemsTable.View()

		// Action hints
		actions := m.theme.Muted.Render("a: Approve  |  s: Skip  |  A: Approve All  |  S: Skip All")

		content = lipgloss.JoinVertical(lipgloss.Left, summary, "", tableContent, "", actions)
	}

	border := m.theme.CardBorder
	if m.focus == 1 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *HygieneModel) renderDetailPanel() string {
	cardWidth := m.width - 4
	titleBar := m.theme.CardTitle.Render("Item Detail")

	var detail string
	if m.focus == 1 && m.cursor < len(m.items) && len(m.items) > 0 {
		item := m.items[m.cursor]
		var catName string
		if m.selectedCat < len(m.categories) {
			catName = m.categories[m.selectedCat].Name
		}

		lines := []string{
			m.theme.Muted.Render("Category:") + " " + catName,
			m.theme.Muted.Render("Path:") + " " + theme.Truncate(item.Path, cardWidth-10),
			m.theme.Muted.Render("Description:") + " " + theme.Truncate(item.Description, cardWidth-18),
			m.theme.Muted.Render("Size:") + " " + formatBytes(item.SizeBytes),
			m.theme.Muted.Render("Status:") + " " + verificationStatusText(item.VerifiedByUser),
		}
		detail = lipgloss.JoinVertical(lipgloss.Left, lines...)
	} else {
		detail = m.theme.Muted.Render("  Use up/down to navigate. a/S to approve/skip items.")
	}

	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, detail))
}

func (m *HygieneModel) renderNavBar() string {
	return m.theme.NavBar.Width(m.width).Padding(0, 1).Render(
		"Tab: Switch  |  Up/Down: Navigate  |  Enter: Scan  |  a/A: Approve  |  s/S: Skip  |  Esc: Back  |  q: Quit",
	)
}

// ============================================================================
// Navigation interface
// ============================================================================

// NavRequest returns the pending navigation request, or "" if none.
func (m *HygieneModel) NavRequest() string { return m.navRequest }

// NavTarget returns the target model for navigation (nil for back/quit).
func (m *HygieneModel) NavTarget() tea.Model { return nil }

// ClearNav clears the navigation request.
func (m *HygieneModel) ClearNav() { m.navRequest = "" }

// ============================================================================
// Helpers
// ============================================================================

// scannerEmoji returns a short label for a scanner type.
func scannerEmoji(scanner domain.ScannerName) string {
	switch scanner {
	case domain.ScannerDocker:
		return "[Dkr]"
	case domain.ScannerAPFS:
		return "[APFS]"
	case domain.ScannerCache:
		return "[Cch]"
	case domain.ScannerFilesystem:
		return "[FS]"
	case domain.ScannerLogs:
		return "[Log]"
	case domain.ScannerXcode:
		return "[Xcd]"
	case domain.ScannerBrew:
		return "[Brw]"
	default:
		return "[?]"
	}
}

// verificationStatusText returns a readable status string.
func verificationStatusText(s domain.VerificationStatus) string {
	switch s {
	case domain.VerificationPending:
		return "Pending review"
	case domain.VerificationApproved:
		return "Approved"
	case domain.VerificationSkipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// verificationStatusTextShort returns a compact verification status label.
func verificationStatusTextShort(s domain.VerificationStatus) string {
	switch s {
	case domain.VerificationPending:
		return "pending"
	case domain.VerificationApproved:
		return "approved"
	case domain.VerificationSkipped:
		return "skipped"
	default:
		return "?"
	}
}
