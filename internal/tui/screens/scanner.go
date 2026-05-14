package screens

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

const (
	focusVolumes  = 0
	focusPatterns = 1
	focusResults  = 2
	cardPad       = 6 // border(4) + padding(2) inside a CardStyle box
)

var defaultPatterns = []string{
	"node_modules",
	".venv",
	"venv",
	"__pycache__",
	".pytest_cache",
	"target",
	"vendor",
	"build",
	"dist",
	".next",
	".nuxt",
	".cache",
	".turbo",
	".eslintcache",
	"coverage",
}

type ScannerModel struct {
	deps        *core.TUIDependencies
	width       int
	height      int
	volumes     []domain.Volume
	selectedVol int
	patterns    []string
	allPatterns []string
	cursor      int
	results     []services.QuickScanResult
	checked     map[int]bool
	scanning    bool
	spinner     spinner.Model
	logEntries  []string
	focusPanel  int
	scrollOff   int // first visible result index
}

func NewScannerModel(deps *core.TUIDependencies) *ScannerModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	volumes := services.GetVolumes()

	pats := make([]string, len(defaultPatterns))
	copy(pats, defaultPatterns)
	all := make([]string, len(defaultPatterns))
	copy(all, defaultPatterns)

	return &ScannerModel{
		deps:        deps,
		volumes:     volumes,
		patterns:    pats,
		allPatterns: all,
		checked:     make(map[int]bool),
		spinner:     s,
		focusPanel:  focusVolumes,
	}
}

func (m *ScannerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *ScannerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case core.ResizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case core.ScanResultMsg:
		m.scanning = false
		m.checked = make(map[int]bool)
		m.scrollOff = 0
		m.cursor = 0
		if msg.Err != nil {
			m.logEntries = append(m.logEntries, fmt.Sprintf("Scan error: %v", msg.Err))
		} else {
			m.results = msg.Results
			m.logEntries = append(m.logEntries, fmt.Sprintf("Found %d artifacts.", len(msg.Results)))
		}
		return m, nil

	case core.ExcludeResultMsg:
		if len(msg.Failures) == 0 {
			m.logEntries = append(m.logEntries, "All paths excluded.")
		} else {
			for path, err := range msg.Failures {
				m.logEntries = append(m.logEntries, fmt.Sprintf("Failed: %s: %v", path, err))
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()
		code := msg.Key().Code

		switch {
		case key == "j" || key == "down":
			m.moveCursor(1)
		case key == "k" || key == "up":
			m.moveCursor(-1)
		case key == "tab":
			m.focusPanel = (m.focusPanel + 1) % 3
			m.cursor = 0
			m.scrollOff = 0
		case key == "space" || code == tea.KeySpace:
			m.toggleSelection()
		case key == "enter" || key == "s":
			return m, m.startScan()
		case key == "e":
			return m, m.excludeSelected()
		case key == "a":
			m.toggleAll()
		}
		return m, nil
	}

	return m, nil
}

func (m *ScannerModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Initializing scanner...")
	}

	// Calculate panel heights to fit within m.height
	logH := 5
	topH := m.height - logH
	if topH < 8 {
		topH = 8
		logH = m.height - topH
	}

	leftW := m.width / 3
	if leftW > 40 {
		leftW = 40
	}
	rightW := m.width - leftW - 2

	left := m.renderLeft(leftW, topH)
	right := m.renderRight(rightW, topH)
	log := m.renderLog(logH)

	var top string
	if core.ResponsiveLayout(m.width) == core.LayoutCompact {
		top = left + "\n" + right
	} else {
		top = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, top, log))
}

func (m *ScannerModel) moveCursor(delta int) {
	n := 0
	switch m.focusPanel {
	case focusVolumes:
		n = len(m.volumes)
	case focusPatterns:
		n = len(m.allPatterns)
	case focusResults:
		n = len(m.results)
	}
	if n == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = n - 1
	}
	if m.cursor >= n {
		m.cursor = 0
	}
	m.clampScroll()
}

func (m *ScannerModel) clampScroll() {
	if m.focusPanel != focusResults {
		return
	}
	visible := m.resultVisibleCount()
	if visible <= 0 {
		visible = 1
	}
	// Keep cursor within visible window
	if m.cursor < m.scrollOff {
		m.scrollOff = m.cursor
	}
	if m.cursor >= m.scrollOff+visible {
		m.scrollOff = m.cursor - visible + 1
	}
	// Don't scroll past the end
	maxOff := len(m.results) - visible
	if maxOff < 0 {
		maxOff = 0
	}
	if m.scrollOff > maxOff {
		m.scrollOff = maxOff
	}
}

func (m *ScannerModel) resultVisibleCount() int {
	if m.height == 0 {
		return 10
	}
	// topH = m.height - logH(5), right panel inner = topH - cardPad - 3(title+ctrl+blank)
	inner := m.height - 5 - cardPad - 3
	if inner < 3 {
		inner = 3
	}
	return inner
}

func (m *ScannerModel) toggleSelection() {
	switch m.focusPanel {
	case focusVolumes:
		if m.cursor < len(m.volumes) {
			m.selectedVol = m.cursor
		}
	case focusPatterns:
		if m.cursor < len(m.allPatterns) {
			pat := m.allPatterns[m.cursor]
			for i, p := range m.patterns {
				if p == pat {
					m.patterns = append(m.patterns[:i], m.patterns[i+1:]...)
					return
				}
			}
			m.patterns = append(m.patterns, pat)
		}
	case focusResults:
		if m.cursor < len(m.results) {
			if m.checked[m.cursor] {
				delete(m.checked, m.cursor)
			} else {
				m.checked[m.cursor] = true
			}
		}
	}
}

func (m *ScannerModel) toggleAll() {
	allChecked := true
	for i, r := range m.results {
		if !r.AlreadyDone && !m.checked[i] {
			allChecked = false
			break
		}
	}
	for i, r := range m.results {
		if !r.AlreadyDone {
			if allChecked {
				delete(m.checked, i)
			} else {
				m.checked[i] = true
			}
		}
	}
}

func (m *ScannerModel) startScan() tea.Cmd {
	if m.scanning || len(m.volumes) == 0 || m.deps.QuickExclude == nil {
		return nil
	}
	m.scanning = true
	vol := m.volumes[m.selectedVol]
	return func() tea.Msg {
		results, err := m.deps.QuickExclude.Scan(context.Background(), vol.Path)
		return core.ScanResultMsg{Results: results, Err: err}
	}
}

func (m *ScannerModel) excludeSelected() tea.Cmd {
	if len(m.checked) == 0 || m.deps.QuickExclude == nil {
		return nil
	}
	var paths []string
	for i, r := range m.results {
		if m.checked[i] && !r.AlreadyDone {
			paths = append(paths, r.Path)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return func() tea.Msg {
		return core.ExcludeResultMsg{Failures: m.deps.QuickExclude.ExcludePaths(context.Background(), paths)}
	}
}

func (m *ScannerModel) patternOn(pat string) bool {
	for _, p := range m.patterns {
		if p == pat {
			return true
		}
	}
	return false
}

// fit truncates or right-pads s to exactly n visible runes.
func fit(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}

// --- Rendering ---

func (m *ScannerModel) renderLeft(w, maxH int) string {
	inner := w - cardPad
	if inner < 10 {
		inner = 10
	}

	var lines []string
	lines = append(lines, fit("Volumes", inner))

	for i, vol := range m.volumes {
		cur := "  "
		if m.focusPanel == focusVolumes && m.cursor == i {
			cur = "> "
		}
		mark := "○ "
		if i == m.selectedVol {
			mark = "● "
		}
		name := fit(core.TruncatePath(vol.Name, inner-4), inner-4)
		line := cur + mark + name
		if m.focusPanel == focusVolumes && m.cursor == i {
			line = core.SelectedRowStyle.Render(fit(line, inner))
		}
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, fit(fmt.Sprintf("Patterns (%d/%d)", len(m.patterns), len(m.allPatterns)), inner))

	for i, pat := range m.allPatterns {
		cur := "  "
		if m.focusPanel == focusPatterns && m.cursor == i {
			cur = "> "
		}
		mark := "○ "
		if m.patternOn(pat) {
			mark = "● "
		}
		line := cur + mark + fit(pat, inner-4)
		if m.focusPanel == focusPatterns && m.cursor == i {
			line = core.SelectedRowStyle.Render(fit(line, inner))
		} else if m.patternOn(pat) {
			line = core.ValueStyle.Render(fit(line, inner))
		} else {
			line = core.LabelStyle.Render(fit(line, inner))
		}
		lines = append(lines, line)
	}

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *ScannerModel) renderRight(w, maxH int) string {
	inner := w - cardPad
	if inner < 20 {
		inner = 20
	}

	// Title
	title := "Scan Results"
	if m.scanning {
		title = fmt.Sprintf("%s %s Scanning...", title, m.spinner.View())
	} else if len(m.results) > 0 {
		title = fmt.Sprintf("%s (%d found, %d sel)", title, len(m.results), len(m.checked))
	}

	var lines []string
	lines = append(lines, fit(title, inner))

	if len(m.results) == 0 && !m.scanning {
		lines = append(lines, "")
		lines = append(lines, "  No results.")
		lines = append(lines, "  Press Enter or s to scan.")
	} else if m.scanning {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s Scanning...", m.spinner.View()))
	} else {
		patW := 16
		sizeW := 9
		pathW := inner - 3 - patW - sizeW
		if pathW < 8 {
			pathW = 8
		}

		// Calculate visible window
		m.clampScroll()
		visible := m.resultVisibleCount()
		end := m.scrollOff + visible
		if end > len(m.results) {
			end = len(m.results)
		}

		for i := m.scrollOff; i < end; i++ {
			r := m.results[i]
			mark := "[ ]"
			if m.checked[i] {
				mark = "[x]"
			}
			pattern := fit(core.TruncatePath(r.PatternName, patW), patW)
			path := fit(core.TruncatePath(r.Path, pathW), pathW)
			size := fit(core.FormatBytes(r.SizeBytes), sizeW)
			line := mark + " " + pattern + " " + path + size

			if m.focusPanel == focusResults && m.cursor == i {
				line = core.SelectedRowStyle.Render(fit(line, inner))
			} else if m.checked[i] {
				line = core.ValueStyle.Render(fit(line, inner))
			} else {
				line = fit(line, inner)
			}
			lines = append(lines, line)
		}

		// Scroll indicator
		if m.scrollOff > 0 || end < len(m.results) {
			info := fmt.Sprintf("  ↑ %d-%d / %d ↓", m.scrollOff+1, end, len(m.results))
			lines = append(lines, core.LabelStyle.Render(fit(info, inner)))
		}
	}

	// Controls — always last line
	lines = append(lines, "")
	lines = append(lines, core.LabelStyle.Render(fit("tab:panels  space:toggle  a:all  s:scan  e:exclude", inner)))

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *ScannerModel) renderLog(h int) string {
	inner := m.width - cardPad
	if inner < 10 {
		inner = 10
	}

	var lines []string
	lines = append(lines, fit("Log", inner))

	if len(m.logEntries) == 0 {
		lines = append(lines, core.LabelStyle.Render("  No activity yet."))
	} else {
		start := 0
		// Reserve 1 line for header, rest for entries
		maxEntries := h - cardPad - 1
		if maxEntries < 1 {
			maxEntries = 1
		}
		if len(m.logEntries) > maxEntries {
			start = len(m.logEntries) - maxEntries
		}
		for _, entry := range m.logEntries[start:] {
			lines = append(lines, fit(entry, inner))
		}
	}

	return core.CardStyle.Width(m.width).Height(h).Render(strings.Join(lines, "\n"))
}
