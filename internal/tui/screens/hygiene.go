package screens

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

// HygieneModel is the Bubble Tea model for the Hygiene screen.
type HygieneModel struct {
	deps         *core.TUIDependencies
	width        int
	height       int
	categories   []domain.JunkCategory
	items        []domain.JunkItem
	catCursor    int
	itemCursor   int
	catScrollOff int
	itemScrollOff int
	checkedItems map[int64]bool
	focusPanel   int // 0=categories, 1=items
	scanning     bool
	spinner      spinner.Model
	loading      bool
}

// NewHygieneModel creates a new HygieneModel with the given dependencies.
func NewHygieneModel(deps *core.TUIDependencies) *HygieneModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return &HygieneModel{
		deps:         deps,
		checkedItems: make(map[int64]bool),
		spinner:      s,
	}
}

// Init loads the initial category list.
func (m *HygieneModel) Init() tea.Cmd {
	if m.deps == nil || !m.deps.DBAvailable || m.deps.JunkCatRepo == nil {
		return nil
	}
	m.loading = true
	return func() tea.Msg {
		cats, err := m.deps.JunkCatRepo.FindAll(context.Background())
		return core.HygieneCategoriesLoadMsg{Categories: cats, Err: err}
	}
}

// Update handles messages for the Hygiene screen.
func (m *HygieneModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case core.ResizeMsg:
		m.width = message.Width
		m.height = message.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd

	case core.HygieneCategoriesLoadMsg:
		m.loading = false
		if message.Err != nil {
			return m, nil
		}
		m.categories = message.Categories
		m.catScrollOff = 0
		if m.catCursor >= len(m.categories) {
			m.catCursor = len(m.categories) - 1
		}
		if m.catCursor < 0 {
			m.catCursor = 0
		}
		if len(m.categories) > 0 && m.deps.JunkItemRepo != nil {
			catID := m.categories[0].ID
			return m, func() tea.Msg {
				items, err := m.deps.JunkItemRepo.FindByCategory(context.Background(), catID)
				return core.HygieneItemsLoadMsg{Items: items, Err: err}
			}
		}
		return m, nil

	case core.HygieneItemsLoadMsg:
		if message.Err != nil {
			return m, nil
		}
		m.items = message.Items
		m.itemCursor = 0
		m.itemScrollOff = 0
		m.checkedItems = make(map[int64]bool)
		return m, nil

	case core.HygieneScanMsg:
		m.scanning = false
		if message.Err == nil {
			if len(m.categories) > 0 && m.categories[m.catCursor].ID == message.CategoryID {
				m.items = message.Items
				m.itemCursor = 0
				m.itemScrollOff = 0
				m.checkedItems = make(map[int64]bool)
			}
		}
		return m, nil

	case core.HygieneVerifyMsg:
		if len(m.categories) > 0 && m.deps.JunkItemRepo != nil {
			catID := m.categories[m.catCursor].ID
			return m, func() tea.Msg {
				items, err := m.deps.JunkItemRepo.FindByCategory(context.Background(), catID)
				return core.HygieneItemsLoadMsg{Items: items, Err: err}
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		switch message.String() {
		case "j", "down":
			if m.focusPanel == 0 {
				if m.catCursor < len(m.categories)-1 {
					m.catCursor++
				}
				m.clampCatScroll()
			} else {
				if m.itemCursor < len(m.items)-1 {
					m.itemCursor++
				}
				m.clampItemScroll()
			}
		case "k", "up":
			if m.focusPanel == 0 {
				if m.catCursor > 0 {
					m.catCursor--
				}
				m.clampCatScroll()
			} else {
				if m.itemCursor > 0 {
					m.itemCursor--
				}
				m.clampItemScroll()
			}
		case "tab":
			if m.focusPanel == 0 {
				m.focusPanel = 1
			} else {
				m.focusPanel = 0
			}
		case " ":
			if m.focusPanel == 1 && len(m.items) > 0 {
				id := m.items[m.itemCursor].ID
				if m.checkedItems[id] {
					delete(m.checkedItems, id)
				} else {
					m.checkedItems[id] = true
				}
			}
		case "enter":
			if len(m.categories) == 0 || m.deps.JunkService == nil {
				return m, nil
			}
			selected := m.categories[m.catCursor]
			m.scanning = true
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				items, err := m.deps.JunkService.ScanCategory(context.Background(), selected)
				return core.HygieneScanMsg{CategoryID: selected.ID, Items: items, Err: err}
			})
		case "a":
			if len(m.checkedItems) == 0 || m.deps.JunkItemRepo == nil {
				return m, nil
			}
			ids := m.checkedIDs()
			return m, func() tea.Msg {
				_, err := m.deps.JunkItemRepo.BatchSetVerification(context.Background(), ids, domain.VerificationApproved)
				return core.HygieneVerifyMsg{IDs: ids, Status: domain.VerificationApproved, Err: err}
			}
		case "s":
			if len(m.checkedItems) == 0 || m.deps.JunkItemRepo == nil {
				return m, nil
			}
			ids := m.checkedIDs()
			return m, func() tea.Msg {
				_, err := m.deps.JunkItemRepo.BatchSetVerification(context.Background(), ids, domain.VerificationSkipped)
				return core.HygieneVerifyMsg{IDs: ids, Status: domain.VerificationSkipped, Err: err}
			}
		}
	}

	return m, nil
}

func (m *HygieneModel) catVisibleCount() int {
	if m.height == 0 {
		return 10
	}
	bottomH := 3 // detail + help + scanning
	contentH := m.height - bottomH
	inner := contentH - cardPad - 1 // title
	if inner < 3 {
		inner = 3
	}
	return inner
}

func (m *HygieneModel) itemVisibleCount() int {
	return m.catVisibleCount() // same panel height
}

func (m *HygieneModel) clampCatScroll() {
	visible := m.catVisibleCount()
	if visible <= 0 {
		visible = 1
	}
	if m.catCursor < m.catScrollOff {
		m.catScrollOff = m.catCursor
	}
	if m.catCursor >= m.catScrollOff+visible {
		m.catScrollOff = m.catCursor - visible + 1
	}
	maxOff := len(m.categories) - visible
	if maxOff < 0 {
		maxOff = 0
	}
	if m.catScrollOff > maxOff {
		m.catScrollOff = maxOff
	}
}

func (m *HygieneModel) clampItemScroll() {
	visible := m.itemVisibleCount()
	if visible <= 0 {
		visible = 1
	}
	if m.itemCursor < m.itemScrollOff {
		m.itemScrollOff = m.itemCursor
	}
	if m.itemCursor >= m.itemScrollOff+visible {
		m.itemScrollOff = m.itemCursor - visible + 1
	}
	maxOff := len(m.items) - visible
	if maxOff < 0 {
		maxOff = 0
	}
	if m.itemScrollOff > maxOff {
		m.itemScrollOff = maxOff
	}
}

// View renders the Hygiene screen.
func (m *HygieneModel) View() tea.View {
	if m.deps == nil || !m.deps.DBAvailable {
		return tea.NewView(core.DegradeBanner("Database unavailable -- hygiene data cannot be loaded."))
	}

	if m.loading {
		return tea.NewView(fmt.Sprintf("\n  %s Loading categories...", m.spinner.View()))
	}

	if m.width == 0 || m.height == 0 {
		return tea.NewView("Initializing hygiene...")
	}

	layout := core.ResponsiveLayout(m.width)

	bottomH := 3 // detail bar + help + possible scanning line
	contentH := m.height - bottomH
	if contentH < 8 {
		contentH = 8
	}

	leftW := m.width / 3
	if layout == core.LayoutCompact {
		leftW = m.width
	}
	rightW := m.width - leftW - 3
	if layout == core.LayoutCompact {
		rightW = m.width
	}

	left := m.renderCategories(leftW, contentH)
	right := m.renderItems(rightW, contentH)

	var content string
	if layout == core.LayoutCompact {
		content = left + "\n" + right
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	// Detail bar
	var detailLines []string
	if len(m.items) > 0 && m.itemCursor < len(m.items) {
		selected := m.items[m.itemCursor]
		detailLines = append(detailLines,
			core.LabelStyle.Render("  Path: ")+core.ValueStyle.Render(selected.Path),
			core.LabelStyle.Render("  Size: ")+core.ValueStyle.Render(core.FormatBytes(selected.SizeBytes)),
			core.LabelStyle.Render("  Status: ")+verificationLabel(selected.VerifiedByUser),
		)
	} else {
		detailLines = append(detailLines, core.LabelStyle.Render("  Select an item to view details"))
	}
	detailBar := lipgloss.NewStyle().Width(m.width).Render(strings.Join(detailLines, "  │  "))

	checked := len(m.checkedItems)
	helpLine := core.LabelStyle.Render(fmt.Sprintf("  j/k:navigate  tab:switch  space:toggle  enter:scan  a:approve(%d)  s:skip(%d)", checked, checked))

	parts := []string{content, detailBar, helpLine}
	if m.scanning {
		parts = append([]string{fmt.Sprintf("  %s Scanning...", m.spinner.View())}, parts...)
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m *HygieneModel) renderCategories(w, maxH int) string {
	inner := w - cardPad
	if inner < 10 {
		inner = 10
	}

	var lines []string
	lines = append(lines, fit("Categories", inner))

	if len(m.categories) == 0 {
		lines = append(lines, core.LabelStyle.Render("  No categories found."))
	} else {
		m.clampCatScroll()
		visible := m.catVisibleCount()
		end := m.catScrollOff + visible
		if end > len(m.categories) {
			end = len(m.categories)
		}

		for i := m.catScrollOff; i < end; i++ {
			c := m.categories[i]
			enabled := core.StatusIcon(c.Enabled)
			label := fmt.Sprintf("  %s %s", enabled, fit(c.Name, inner-4))
			if i == m.catCursor {
				label = core.SelectedRowStyle.Render(fit(label, inner))
			}
			lines = append(lines, label)
		}

		if m.catScrollOff > 0 || end < len(m.categories) {
			info := fmt.Sprintf("  ↑ %d-%d / %d ↓", m.catScrollOff+1, end, len(m.categories))
			lines = append(lines, core.LabelStyle.Render(fit(info, inner)))
		}
	}

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *HygieneModel) renderItems(w, maxH int) string {
	inner := w - cardPad
	if inner < 20 {
		inner = 20
	}

	catName := ""
	if len(m.categories) > 0 {
		catName = m.categories[m.catCursor].Name
	}
	title := "Items"
	if catName != "" {
		title = fmt.Sprintf("Items ─ %s", catName)
	}

	var lines []string
	lines = append(lines, fit(title, inner))

	if len(m.items) == 0 {
		lines = append(lines, core.LabelStyle.Render(fmt.Sprintf("  No items for %s. Press Enter to scan.", catName)))
	} else {
		m.clampItemScroll()
		visible := m.itemVisibleCount()
		end := m.itemScrollOff + visible
		if end > len(m.items) {
			end = len(m.items)
		}

		for i := m.itemScrollOff; i < end; i++ {
			item := m.items[i]
			checkbox := "[ ]"
			if m.checkedItems[item.ID] {
				checkbox = core.StatusOK.Render("[x]")
			}
			status := verificationIcon(item.VerifiedByUser)
			path := fit(core.TruncatePath(item.Path, inner-14), inner-14)
			size := core.FormatBytes(item.SizeBytes)
			line := fmt.Sprintf("  %s %s %s %s", checkbox, status, path, size)
			if i == m.itemCursor {
				line = core.SelectedRowStyle.Render(fit(line, inner))
			}
			lines = append(lines, line)
		}

		if m.itemScrollOff > 0 || end < len(m.items) {
			info := fmt.Sprintf("  ↑ %d-%d / %d ↓", m.itemScrollOff+1, end, len(m.items))
			lines = append(lines, core.LabelStyle.Render(fit(info, inner)))
		}
	}

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *HygieneModel) checkedIDs() []int64 {
	ids := make([]int64, 0, len(m.checkedItems))
	for id := range m.checkedItems {
		ids = append(ids, id)
	}
	return ids
}

func verificationIcon(status domain.VerificationStatus) string {
	switch status {
	case domain.VerificationApproved:
		return core.StatusOK.Render("✓")
	case domain.VerificationSkipped:
		return core.LabelStyle.Render("⊘")
	default:
		return core.StatusWarn.Render("○")
	}
}

func verificationLabel(status domain.VerificationStatus) string {
	switch status {
	case domain.VerificationApproved:
		return core.StatusOK.Render("Approved")
	case domain.VerificationSkipped:
		return core.LabelStyle.Render("Skipped")
	default:
		return core.StatusWarn.Render("Pending")
	}
}
