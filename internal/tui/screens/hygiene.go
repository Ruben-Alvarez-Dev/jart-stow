package screens

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Hygiene Screen
// ============================================================================

// JunkLister provides access to junk categories and items for the hygiene screen.
type JunkLister interface {
	ListCategories() ([]domain.JunkCategory, error)
	ListPendingItemsByCategory(categoryID int64) ([]domain.JunkItem, error)
	CountPendingItems() (int, error)
}

// HygieneModel displays junk categories and items for user review and cleanup.
type HygieneModel struct {
	theme *theme.Theme
	junk  JunkLister

	width  int
	height int

	focus       int // 0 = categories sidebar, 1 = items panel
	cursor      int
	selectedCat int
	categories  []domain.JunkCategory
	items       []domain.JunkItem
	loaded      bool

	navRequest string
}

// NewHygieneModel creates a new HygieneModel.
func NewHygieneModel(t *theme.Theme, junk JunkLister) *HygieneModel {
	return &HygieneModel{
		theme: t,
		junk:  junk,
	}
}

// Init initializes the hygiene screen.
func (m *HygieneModel) Init() tea.Cmd {
	return nil
}

// Update handles key events for the hygiene screen.
func (m *HygieneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 2
			m.cursor = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			limit := len(m.categories)
			if m.focus == 1 {
				limit = len(m.items)
			}
			if m.cursor < limit-1 {
				m.cursor++
			}
		case "enter":
			// Select category to view its items
			if m.focus == 0 && m.cursor < len(m.categories) {
				m.selectedCat = m.cursor
				m.loadItems()
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

// View renders the hygiene screen layout.
func (m *HygieneModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	m.loadData()

	header := m.renderHeader()
	panels := m.renderPanels()
	detailPanel := m.renderDetailPanel()
	navBar := m.renderNavBar()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		panels,
		detailPanel,
	)

	mainArea := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 3).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, navBar)
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
			m.loadItems()
		}
	}
}

func (m *HygieneModel) loadItems() {
	if m.junk == nil || m.selectedCat >= len(m.categories) {
		return
	}
	cat := m.categories[m.selectedCat]
	items, err := m.junk.ListPendingItemsByCategory(cat.ID)
	if err == nil {
		m.items = items
	}
}

func (m *HygieneModel) renderHeader() string {
	title := m.theme.Header.Render("HYGIENE")
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
	rightWidth := m.width - leftWidth - 6

	leftPanel := m.renderCategoriesSidebar(leftWidth)
	rightPanel := m.renderItemsPanel(rightWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m *HygieneModel) renderCategoriesSidebar(width int) string {
	titleBar := m.theme.CardTitle.Render("Categories")

	var lines []string
	if len(m.categories) > 0 {
		for i, cat := range m.categories {
			var count int
			if m.junk != nil {
				c, _ := m.junk.CountPendingItems()
				_ = c // Use category-specific count in future
			}
			rowStyle := lipgloss.NewStyle()
			if m.focus == 0 && i == m.cursor {
				rowStyle = m.theme.Selected
			}
			scannerIcon := scannerEmoji(cat.Scanner)
			line := rowStyle.Render(scannerIcon + " " + theme.Truncate(cat.Name, width-12) + " " + itoa(count))
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, m.theme.Muted.Render("  No categories."))
	}

	lines = append(lines, "")
	lines = append(lines, m.theme.Muted.Render("[Scan All] [Scan Selected]"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	border := m.theme.CardBorder
	if m.focus == 0 {
		border = border.BorderForeground(theme.ColorPrimary)
	}
	return border.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, content))
}

func (m *HygieneModel) renderItemsPanel(width int) string {
	titleBar := m.theme.CardTitle.Render("Items to Review")

	var lines []string
	if len(m.items) > 0 {
		for i, item := range m.items {
			rowStyle := lipgloss.NewStyle()
			if m.focus == 1 && i == m.cursor {
				rowStyle = m.theme.Selected
			}
			check := "[ ]"
			if item.IsApproved() {
				check = m.theme.Success.Render("[x]")
			} else if item.IsCleaned() {
				check = m.theme.Muted.Render("[x]")
			}
			line := rowStyle.Render(check + " " + theme.Truncate(item.Description, width-10) + " " + formatBytes(item.SizeBytes))
			lines = append(lines, line)
		}

		// Summary
		lines = append(lines, m.theme.Muted.Render(stringOfChar('-', width-4)))
		var totalSize int64
		for _, item := range m.items {
			totalSize += item.SizeBytes
		}
		summary := m.theme.Primary.Render("Total: " + itoa(len(m.items)) + " items  " + formatBytes(totalSize))
		lines = append(lines, summary)
	} else {
		if m.junk == nil || len(m.categories) == 0 {
			lines = append(lines, m.theme.Muted.Render("  No junk items found. Scan categories to discover junk."))
		} else {
			lines = append(lines, m.theme.Muted.Render("  Select a category on the left to view items."))
		}
	}

	lines = append(lines, "")
	lines = append(lines, m.theme.Muted.Render("[Approve Selected] [Skip Selected] [Approve All]"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

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
	if m.focus == 1 && m.cursor < len(m.items) {
		item := m.items[m.cursor]
		var catName string
		if m.selectedCat < len(m.categories) {
			catName = m.categories[m.selectedCat].Name
		}
		lines := []string{
			m.theme.Muted.Render("Category:") + " " + catName,
			m.theme.Muted.Render("Path:") + " " + theme.Truncate(item.Path, cardWidth-10),
			m.theme.Muted.Render("Size:") + " " + formatBytes(item.SizeBytes),
			m.theme.Muted.Render("Status:") + " " + verificationStatusText(item.VerifiedByUser),
		}
		detail = lipgloss.JoinVertical(lipgloss.Left, lines...)
	} else {
		detail = m.theme.Muted.Render("  Use up/down to navigate items. Press Space to toggle selection.")
	}

	return m.theme.CardBorder.
		Width(cardWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleBar, detail))
}

func (m *HygieneModel) renderNavBar() string {
	return m.theme.HelpText.Render("← Esc:Back  q:Quit  |  up/down Navigate  |  Space: Toggle  |  A: Approve  |  S: Skip  |  C: Clean All")
}

// NavRequest returns the pending navigation request, or "" if none.
func (m *HygieneModel) NavRequest() string { return m.navRequest }

// NavTarget returns the target model for navigation (nil for back/quit).
func (m *HygieneModel) NavTarget() tea.Model { return nil }

// ClearNav clears the navigation request.
func (m *HygieneModel) ClearNav() { m.navRequest = "" }

// scannerEmoji returns an icon for a scanner type.
func scannerEmoji(scanner domain.ScannerName) string {
	switch scanner {
	case domain.ScannerDocker:
		return "[Docker]"
	case domain.ScannerAPFS:
		return "[APFS]"
	case domain.ScannerCache:
		return "[Cache]"
	case domain.ScannerFilesystem:
		return "[FS]"
	case domain.ScannerLogs:
		return "[Logs]"
	case domain.ScannerXcode:
		return "[Xcode]"
	case domain.ScannerBrew:
		return "[Brew]"
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
