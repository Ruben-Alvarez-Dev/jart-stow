package screens

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/apfs"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/docker"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/filesystem"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

const (
	focusVolumes  = 0
	focusPatterns = 1
	focusResults  = 2
	cardPad       = 6
	maxTreeDepth  = 3
)

type scanCat struct {
	id       string
	label    string
	group    string
	isSystem bool
}

type catGroup struct {
	name    string
	icon    string
	cats    []scanCat
	expanded bool
}

var groups = []catGroup{
	{name: "Node.js", icon: "⬢", cats: []scanCat{
		{"node_modules", "node_modules", "Node.js", false},
		{".next", ".next", "Node.js", false},
		{".nuxt", ".nuxt", "Node.js", false},
		{".turbo", ".turbo", "Node.js", false},
		{".eslintcache", ".eslintcache", "Node.js", false},
	}},
	{name: "Python", icon: "🐍", cats: []scanCat{
		{".venv", ".venv", "Python", false},
		{"venv", "venv", "Python", false},
		{"__pycache__", "__pycache__", "Python", false},
		{".pytest_cache", ".pytest_cache", "Python", false},
	}},
	{name: "Rust/Java", icon: "⚙", cats: []scanCat{
		{"target", "target", "Rust/Java", false},
		{".gradle", ".gradle", "Rust/Java", false},
		{".m2", ".m2", "Rust/Java", false},
	}},
	{name: "Go", icon: " Go", cats: []scanCat{
		{"vendor", "vendor", "Go", false},
	}},
	{name: "Generic", icon: "📦", cats: []scanCat{
		{"build", "build", "Generic", false},
		{"dist", "dist", "Generic", false},
		{"coverage", "coverage", "Generic", false},
		{".cache", ".cache", "Generic", false},
	}},
	{name: "Hidden", icon: "👁", cats: []scanCat{
		{".config", ".config", "Hidden", false},
		{".local", ".local", "Hidden", false},
		{".npm", ".npm", "Hidden", false},
		{".cargo", ".cargo", "Hidden", false},
		{".rustup", ".rustup", "Hidden", false},
		{".docker", ".docker", "Hidden", false},
	}},
	{name: "Docker", icon: "🐳", cats: []scanCat{
		{"docker_images", "Docker Images", "Docker", true},
		{"docker_containers", "Docker Containers", "Docker", true},
		{"docker_volumes", "Docker Volumes", "Docker", true},
		{"docker_build_cache", "Docker Build Cache", "Docker", true},
	}},
	{name: "APFS", icon: "💾", cats: []scanCat{
		{"apfs_snapshots", "APFS Snapshots", "APFS", true},
	}},
	{name: "System", icon: "🔧", cats: []scanCat{
		{"tmp_files", "/tmp Files", "System", true},
		{"downloads", "Downloads (>100MB)", "System", true},
		{"system_caches", "System Caches", "System", true},
		{"user_caches", "User Caches", "System", true},
	}},
	{name: "Dev Tools", icon: "🛠", cats: []scanCat{
		{"xcode_derived_data", "Xcode DerivedData", "Dev Tools", true},
		{"brew_cache", "Brew Cache", "Dev Tools", true},
	}},
}

// flatCat is a flattened category for indexed access.
type flatCat struct {
	id       string
	label    string
	group    string
	isSystem bool
	groupIdx int
}

func buildFlatCats(expanded map[int]bool) []flatCat {
	var flat []flatCat
	for gi, g := range groups {
		if !expanded[gi] {
			flat = append(flat, flatCat{id: fmt.Sprintf("__group_%d", gi), label: g.name, groupIdx: gi})
			continue
		}
		flat = append(flat, flatCat{id: fmt.Sprintf("__group_%d", gi), label: g.name, groupIdx: gi})
		for _, c := range g.cats {
			flat = append(flat, flatCat{
				id: c.id, label: c.label, group: c.group,
				isSystem: c.isSystem, groupIdx: gi,
			})
		}
	}
	return flat
}

type treeNode struct {
	path  string
	name  string
	depth int
	isDir bool
}

type ScannerModel struct {
	deps         *core.TUIDependencies
	width        int
	height       int
	volumes      []domain.Volume
	selectedVols map[int]bool
	activeCats   map[string]bool
	cursor       int
	results      []services.QuickScanResult
	checked      map[int]bool
	scanning     bool
	spinner      spinner.Model
	logEntries   []string
	focusPanel   int
	scrollOff    int
	leftScrollOff int // scroll offset for left panel (categories/volumes)

	treeNodes    []treeNode
	treeCursor   int
	expandedDirs map[string]bool
	expandedGroups map[int]bool
	showHidden   bool
	keyConsumed  bool

	// Real-time progress fields
	currentProgress services.ScanProgress
	liveFeed        []services.QuickScanResult
	progressChan    chan services.ScanProgress
}

func NewScannerModel(deps *core.TUIDependencies) *ScannerModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	volumes := services.GetVolumes()

	active := make(map[string]bool)
	for _, g := range groups {
		for _, c := range g.cats {
			if !c.isSystem {
				active[c.id] = true
			}
		}
	}

	expandedGroups := make(map[int]bool)
	for i := range groups {
		expandedGroups[i] = true
	}

	m := &ScannerModel{
		deps:           deps,
		volumes:        volumes,
		selectedVols:   make(map[int]bool),
		activeCats:     active,
		checked:        make(map[int]bool),
		spinner:        s,
		focusPanel:     focusVolumes,
		expandedDirs:   make(map[string]bool),
		expandedGroups: expandedGroups,
		showHidden:     true,
	}
	m.rebuildTree()
	return m
}

func (m *ScannerModel) Init() tea.Cmd { return m.spinner.Tick }

func (m *ScannerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	m.keyConsumed = false

	switch msg := message.(type) {
	case core.ResizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case core.ScanProgressMsg:
		m.currentProgress = msg.Progress
		if msg.Progress.FoundArtifact != nil {
			art := msg.Progress.FoundArtifact
			res := services.QuickScanResult{
				Path: art.Path, PatternName: art.PatternName, SizeBytes: art.SizeBytes,
			}
			m.results = append(m.results, res)
			// Add to live feed (keep last 5)
			m.liveFeed = append([]services.QuickScanResult{res}, m.liveFeed...)
			if len(m.liveFeed) > 5 {
				m.liveFeed = m.liveFeed[:5]
			}
		}
		if msg.Progress.ScanningDone {
			m.scanning = false
			m.logEntries = append(m.logEntries, fmt.Sprintf("Scan completed: Found %d items, %s total.", len(m.results), core.FormatBytes(m.currentProgress.TotalSize)))
			return m, nil
		}
		// Wait for next progress
		return m, m.waitForProgress()

	case core.ScanResultMsg:
		// We handle progress now, so results are already in m.results
		return m, nil

	case core.ExcludeResultMsg:
		excluded := 0
		for i := range m.results {
			if m.checked[i] {
				path := m.results[i].Path
				if _, failed := msg.Failures[path]; !failed {
					m.results[i].AlreadyDone = true
					delete(m.checked, i)
					excluded++
				}
			}
		}
		if len(msg.Failures) == 0 {
			m.logEntries = append(m.logEntries, fmt.Sprintf("Excluded %d paths.", excluded))
		} else {
			m.logEntries = append(m.logEntries, fmt.Sprintf("Excluded %d. %d failed.", excluded, len(msg.Failures)))
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
		case key == "left":
			if m.focusPanel == focusVolumes {
				m.collapseDir()
				m.keyConsumed = true
			}
		case key == "right":
			if m.focusPanel == focusVolumes {
				m.expandDir()
				m.keyConsumed = true
			}
		case key == "enter":
			m.toggleSelection()
		case key == "space" || code == tea.KeySpace:
			m.toggleSelection()
		case key == "s":
			return m, m.startScan()
		case key == "e":
			if len(m.checked) == 0 {
				m.logEntries = append(m.logEntries, "Nothing selected.")
				return m, nil
			}
			m.logEntries = append(m.logEntries, fmt.Sprintf("Excluding %d...", len(m.checked)))
			return m, m.excludeSelected()
		case key == "a":
			m.toggleAll()
		case key == "h":
			m.showHidden = !m.showHidden
			m.rebuildTree()
		}
		return m, nil
	}

	return m, nil
}

func (m *ScannerModel) IsAtBottom() bool {
	if len(m.results) == 0 {
		return true
	}
	return m.cursor >= len(m.results)-1
}

func (m *ScannerModel) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		if m.progressChan == nil {
			return nil
		}
		p, ok := <-m.progressChan
		if !ok {
			return nil
		}
		return core.ScanProgressMsg{Progress: p}
	}
}

func (m *ScannerModel) ConsumedKey() bool { return m.keyConsumed }

func (m *ScannerModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Initializing...")
	}

	footerH := 6
	topH := m.height - footerH
	if topH < 8 {
		topH = 8
		footerH = m.height - topH
	}

	leftW := m.width / 3
	if leftW > 40 {
		leftW = 40
	}
	rightW := m.width - leftW - 2

	left := m.renderLeft(leftW, topH)
	right := m.renderRight(rightW, topH)
	footer := m.renderFooter(footerH)

	var top string
	if core.ResponsiveLayout(m.width) == core.LayoutCompact {
		top = left + "\n" + right
	} else {
		top = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, top, footer))
}

// --- Cursor ---

func (m *ScannerModel) moveCursor(delta int) {
	switch m.focusPanel {
	case focusVolumes:
		n := len(m.treeNodes)
		if n == 0 {
			return
		}
		m.treeCursor += delta
		if m.treeCursor < 0 {
			m.treeCursor = n - 1
		}
		if m.treeCursor >= n {
			m.treeCursor = 0
		}
		m.clampLeftScroll(m.treeCursor)
	case focusPatterns:
		flat := buildFlatCats(m.expandedGroups)
		n := len(flat)
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
		m.clampLeftScroll(m.cursor)
	case focusResults:
		n := len(m.results)
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
}

// --- Tree ---

func (m *ScannerModel) rebuildTree() {
	var nodes []treeNode
	for _, vol := range m.volumes {
		nodes = append(nodes, treeNode{path: vol.Path, name: vol.Name, depth: 0, isDir: true})
		if m.expandedDirs[vol.Path] {
			nodes = append(nodes, m.readChildren(vol.Path, 1)...)
		}
	}
	m.treeNodes = nodes
	if m.treeCursor >= len(m.treeNodes) {
		m.treeCursor = len(m.treeNodes) - 1
	}
	if m.treeCursor < 0 {
		m.treeCursor = 0
	}
}

func (m *ScannerModel) readChildren(dirPath string, depth int) []treeNode {
	if depth > maxTreeDepth {
		return nil
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil
	}
	var nodes []treeNode
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !m.showHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fullPath := filepath.Join(dirPath, e.Name())
		nodes = append(nodes, treeNode{path: fullPath, name: e.Name(), depth: depth, isDir: true})
		if m.expandedDirs[fullPath] {
			nodes = append(nodes, m.readChildren(fullPath, depth+1)...)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].name < nodes[j].name })
	return nodes
}

func (m *ScannerModel) expandDir() {
	if m.treeCursor >= len(m.treeNodes) {
		return
	}
	node := m.treeNodes[m.treeCursor]
	if m.expandedDirs[node.path] {
		return
	}
	m.expandedDirs[node.path] = true
	m.rebuildTree()
}

func (m *ScannerModel) collapseDir() {
	if m.treeCursor >= len(m.treeNodes) {
		return
	}
	node := m.treeNodes[m.treeCursor]
	if m.expandedDirs[node.path] {
		delete(m.expandedDirs, node.path)
		m.rebuildTree()
		return
	}
	parent := filepath.Dir(node.path)
	for i, n := range m.treeNodes {
		if n.path == parent {
			m.treeCursor = i
			return
		}
	}
}

// --- Selection ---

func (m *ScannerModel) toggleSelection() {
	switch m.focusPanel {
	case focusVolumes:
		if m.treeCursor < len(m.treeNodes) {
			if m.selectedVols[m.treeCursor] {
				delete(m.selectedVols, m.treeCursor)
			} else {
				m.selectedVols[m.treeCursor] = true
			}
		}
	case focusPatterns:
		flat := buildFlatCats(m.expandedGroups)
		if m.cursor >= len(flat) {
			return
		}
		fc := flat[m.cursor]
		if strings.HasPrefix(fc.id, "__group_") {
			m.expandedGroups[fc.groupIdx] = !m.expandedGroups[fc.groupIdx]
			return
		}
		if m.activeCats[fc.id] {
			delete(m.activeCats, fc.id)
		} else {
			m.activeCats[fc.id] = true
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
	switch m.focusPanel {
	case focusVolumes:
		allSel := len(m.selectedVols) == len(m.treeNodes) && len(m.treeNodes) > 0
		m.selectedVols = make(map[int]bool)
		if !allSel {
			for i := range m.treeNodes {
				m.selectedVols[i] = true
			}
		}
	case focusPatterns:
		allOn := true
		for _, g := range groups {
			for _, c := range g.cats {
				if !m.activeCats[c.id] {
					allOn = false
				}
			}
		}
		m.activeCats = make(map[string]bool)
		if !allOn {
			for _, g := range groups {
				for _, c := range g.cats {
					m.activeCats[c.id] = true
				}
			}
		}
	case focusResults:
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
}

// --- Scan ---

func (m *ScannerModel) startScan() tea.Cmd {
	if m.scanning {
		return nil
	}
	m.scanning = true
	return func() tea.Msg {
		ctx := context.Background()
		var allResults []services.QuickScanResult
		var scanErr error

		var devLabels []string
		var sysCats []scanCat
		for _, g := range groups {
			for _, c := range g.cats {
				if !m.activeCats[c.id] {
					continue
				}
				if c.isSystem {
					sysCats = append(sysCats, c)
				} else {
					devLabels = append(devLabels, c.label)
				}
			}
		}

		if len(devLabels) > 0 {
			var paths []string
			for i, sel := range m.selectedVols {
				if sel && i < len(m.treeNodes) {
					paths = append(paths, m.treeNodes[i].path)
				}
			}
			if len(paths) == 0 {
				for _, vol := range m.volumes {
					paths = append(paths, vol.Path)
				}
			}
			
			m.progressChan = make(chan services.ScanProgress, 100)
			svc := services.NewScanService(4, devLabels)

			// Start scanning in a goroutine
			go func() {
				for _, p := range paths {
					_, _ = svc.FindArtifacts(ctx, p, m.progressChan)
				}
				close(m.progressChan)
			}()

			return m, m.waitForProgress()
		}

		for _, c := range sysCats {
			for _, item := range m.scanSystem(ctx, c) {
				allResults = append(allResults, services.QuickScanResult{
					Path: item.Path, PatternName: c.id, SizeBytes: item.SizeBytes,
				})
			}
		}

		return core.ScanResultMsg{Results: allResults, Err: scanErr}
	}
}

func (m *ScannerModel) scanSystem(ctx context.Context, cat scanCat) []domain.JunkItem {
	dummy := domain.JunkCategory{Name: cat.id, Scanner: domain.ScannerName(strings.ToLower(cat.group))}
	switch cat.id {
	case "docker_images", "docker_containers", "docker_volumes", "docker_build_cache":
		s := docker.NewScanner()
		if !s.IsAvailable(ctx) {
			return nil
		}
		items, _ := s.Scan(ctx, dummy)
		return items
	case "apfs_snapshots":
		s := apfs.NewScanner()
		if !s.IsAvailable(ctx) {
			return nil
		}
		items, _ := s.Scan(ctx, dummy)
		return items
	case "tmp_files":
		items, _ := filesystem.NewTempScanner().Scan(ctx, dummy)
		return items
	case "system_caches":
		dummy.Name = "system_caches"
		items, _ := filesystem.NewCacheScanner().Scan(ctx, dummy)
		return items
	case "user_caches":
		dummy.Name = "user_caches"
		items, _ := filesystem.NewCacheScanner().Scan(ctx, dummy)
		return items
	case "xcode_derived_data":
		items, _ := filesystem.NewXcodeScanner().Scan(ctx, dummy)
		return items
	case "brew_cache":
		items, _ := filesystem.NewBrewScanner().Scan(ctx, dummy)
		return items
	case "downloads":
		return m.scanDownloads()
	}
	return nil
}

func (m *ScannerModel) scanDownloads() []domain.JunkItem {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dl := filepath.Join(home, "Downloads")
	entries, err := os.ReadDir(dl)
	if err != nil {
		return nil
	}
	var items []domain.JunkItem
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.IsDir() || info.Size() < 100<<20 {
			continue
		}
		items = append(items, domain.JunkItem{
			Path: filepath.Join(dl, e.Name()), Description: e.Name(), SizeBytes: info.Size(),
		})
	}
	return items
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

func (m *ScannerModel) clampScroll() {
	if m.focusPanel != focusResults {
		return
	}
	visible := m.resultVisibleCount()
	if visible <= 0 {
		visible = 1
	}
	if m.cursor < m.scrollOff {
		m.scrollOff = m.cursor
	}
	if m.cursor >= m.scrollOff+visible {
		m.scrollOff = m.cursor - visible + 1
	}
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
	inner := m.height - 6 - cardPad - 3
	if inner < 3 {
		inner = 3
	}
	return inner
}

func (m *ScannerModel) leftVisibleCount() int {
	if m.height == 0 {
		return 20
	}
	// topH = m.height - footerH(6), inner = topH - cardPad
	return m.height - 6 - cardPad
}

func (m *ScannerModel) clampLeftScroll(cursorIdx int) {
	visible := m.leftVisibleCount()
	if visible <= 0 {
		visible = 1
	}
	if cursorIdx < m.leftScrollOff {
		m.leftScrollOff = cursorIdx
	}
	if cursorIdx >= m.leftScrollOff+visible {
		m.leftScrollOff = cursorIdx - visible + 1
	}
	if m.leftScrollOff < 0 {
		m.leftScrollOff = 0
	}
}

// --- Stats helpers ---

func (m *ScannerModel) maxResultSize() int64 {
	var max int64
	for _, r := range m.results {
		if r.SizeBytes > max {
			max = r.SizeBytes
		}
	}
	return max
}

func (m *ScannerModel) totalResultSize() int64 {
	var t int64
	for _, r := range m.results {
		t += r.SizeBytes
	}
	return t
}

func sizeColor(size int64) lipgloss.Style {
	switch {
	case size >= 1<<30:
		return core.StatusErr
	case size >= 100<<20:
		return core.StatusWarn
	default:
		return core.StatusOK
	}
}

func sizeBar(size, maxVal int64, barW int) string {
	if maxVal == 0 || barW <= 0 {
		return ""
	}
	filled := int(float64(size) / float64(maxVal) * float64(barW))
	if filled > barW {
		filled = barW
	}
	if filled < 1 && size > 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
}

func catForID(id string) scanCat {
	for _, g := range groups {
		for _, c := range g.cats {
			if c.id == id || c.label == id {
				return c
			}
		}
	}
	return scanCat{id: id, label: id, group: "Other"}
}

// --- Rendering ---

// fit is an alias kept for other screens that import it.
func fit(s string, n int) string { return truncPlain(s, n) }

func truncPlain(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}

var groupHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(core.ColorPrimary).
	Underline(true)

func (m *ScannerModel) renderLeft(w, maxH int) string {
	inner := w - cardPad
	if inner < 10 {
		inner = 10
	}
	clampW := lipgloss.NewStyle().MaxWidth(inner)

	var lines []string
	hStatus := "visible"
	if !m.showHidden {
		hStatus = "hidden"
	}
	lines = append(lines, truncPlain(fmt.Sprintf("Volumes (dotfiles: %s, h)", hStatus), inner))

	for i, node := range m.treeNodes {
		prefix := "  "
		if m.focusPanel == focusVolumes && m.treeCursor == i {
			prefix = "> "
		}
		mark := "○ "
		if m.selectedVols[i] {
			mark = "● "
		}
		indent := strings.Repeat("  ", node.depth)
		arrow := "▶ "
		if m.expandedDirs[node.path] {
			arrow = "▼ "
		}
		nameW := inner - len(prefix) - len(indent) - len(arrow) - len(mark)
		if nameW < 4 {
			nameW = 4
		}
		plain := prefix + indent + arrow + mark + truncPlain(core.TruncatePath(node.name, nameW), nameW)
		if m.focusPanel == focusVolumes && m.treeCursor == i {
			plain = core.SelectedRowStyle.Render(plain)
		} else if m.selectedVols[i] {
			plain = core.ValueStyle.Render(plain)
		}
		lines = append(lines, clampW.Render(plain))
	}

	if len(m.results) > 0 {
		lines = append(lines, "")
		lines = append(lines, truncPlain(fmt.Sprintf("Found: %d items, %s", len(m.results), core.FormatBytes(m.totalResultSize())), inner))
	}

	activeCount := len(m.activeCats)
	totalCount := 0
	for _, g := range groups {
		totalCount += len(g.cats)
	}
	lines = append(lines, "")
	lines = append(lines, truncPlain(fmt.Sprintf("Categories (%d/%d)", activeCount, totalCount), inner))

	// Accordion-style grouped categories
	flatIdx := 0
	for gi, g := range groups {
		isGroupLine := flatIdx == m.cursor && m.focusPanel == focusPatterns
		groupCur := "  "
		if isGroupLine {
			groupCur = "> "
		}
		exp := "▶"
		if m.expandedGroups[gi] {
			exp = "▼"
		}

		activeInGroup := 0
		for _, c := range g.cats {
			if m.activeCats[c.id] {
				activeInGroup++
			}
		}
		groupLabel := fmt.Sprintf("%s %s %s %s (%d/%d)", groupCur, exp, g.icon, g.name, activeInGroup, len(g.cats))
		lines = append(lines, clampW.Render(groupHeaderStyle.Render(groupLabel)))
		flatIdx++

		if m.expandedGroups[gi] {
			for _, c := range g.cats {
				catCur := "    "
				isCatLine := flatIdx == m.cursor && m.focusPanel == focusPatterns
				if isCatLine {
					catCur = "  > "
				}
				mark := "○"
				if m.activeCats[c.id] {
					mark = "●"
				}

				matchCount := 0
				var matchSize int64
				for _, r := range m.results {
					if r.PatternName == c.id || r.PatternName == c.label {
						matchCount++
						matchSize += r.SizeBytes
					}
				}
				countStr := ""
				if matchCount > 0 {
					countStr = sizeColor(matchSize).Render(fmt.Sprintf(" %d %s", matchCount, core.FormatBytes(matchSize)))
				}

				plain := catCur + mark + " " + c.label
				var catLine string
				if isCatLine {
					catLine = core.SelectedRowStyle.Render(plain) + countStr
				} else if m.activeCats[c.id] {
					catLine = core.ValueStyle.Render(plain) + countStr
				} else {
					catLine = core.LabelStyle.Render(plain) + countStr
				}
				lines = append(lines, clampW.Render(catLine))
				flatIdx++
			}
		}
	}

	// Scroll: only show visible lines
	maxLines := maxH - cardPad
	if maxLines < 3 {
		maxLines = 3
	}
	totalLines := len(lines)
	if totalLines > maxLines {
		// Use leftScrollOff
		off := m.leftScrollOff
		if off < 0 {
			off = 0
		}
		end := off + maxLines
		if end > totalLines {
			end = totalLines
			off = end - maxLines
			if off < 0 {
				off = 0
			}
		}
		visible := lines[off:end]
		if off > 0 || end < totalLines {
			indicator := fmt.Sprintf("  ↑ %d-%d / %d ↓", off+1, end, totalLines)
			if len(visible) > 0 {
				visible[len(visible)-1] = clampW.Render(core.LabelStyle.Render(indicator))
			}
		}
		return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(visible, "\n"))
	}

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *ScannerModel) renderRight(w, maxH int) string {
	inner := w - cardPad
	if inner < 20 {
		inner = 20
	}
	clampW := lipgloss.NewStyle().MaxWidth(inner)

	totalSize := m.totalResultSize()
	var selSize int64
	for i, r := range m.results {
		if m.checked[i] {
			selSize += r.SizeBytes
		}
	}

	title := "Scan Results"
	if m.scanning {
		return m.renderProgress(w, maxH)
	} else if len(m.results) > 0 {
		title = fmt.Sprintf("%d found | %s", len(m.results), core.FormatBytes(totalSize))
		if len(m.checked) > 0 {
			title += fmt.Sprintf(" | sel %s", core.FormatBytes(selSize))
		}
	}

	var lines []string
	lines = append(lines, clampW.Render(title))

	if len(m.results) == 0 && !m.scanning {
		lines = append(lines, "", "  No results.", "  Select categories & press s.")
	} else if m.scanning {
		lines = append(lines, "", fmt.Sprintf("  %s Scanning...", m.spinner.View()))
	} else {
		maxSize := m.maxResultSize()
		barW := 8
		if inner > 80 {
			barW = 12
		}
		// Column widths (plain text only)
		sizeW := 8
		patW := 14
		// Total: mark(3) sp pat sp path sp bar sp size = 3+1+14+1+pathW+1+barW+1+8
		pathW := inner - 3 - 1 - patW - 1 - 1 - barW - 1 - sizeW
		if pathW < 10 {
			pathW = 10
		}

		m.clampScroll()
		visible := m.resultVisibleCount()
		end := m.scrollOff + visible
		if end > len(m.results) {
			end = len(m.results)
		}

		for i := m.scrollOff; i < end; i++ {
			r := m.results[i]
			mark := "[ ]"
			if r.AlreadyDone {
				mark = "[✓]"
			} else if m.checked[i] {
				mark = "[x]"
			}

			// Plain text columns, truncated
			plainPat := truncPlain(core.TruncatePath(r.PatternName, patW), patW)
			plainPath := truncPlain(core.TruncatePath(r.Path, pathW), pathW)
			plainSize := truncPlain(core.FormatBytes(r.SizeBytes), sizeW)
			plainBar := sizeBar(r.SizeBytes, maxSize, barW)

			// Style only bar and size
			styledBar := sizeColor(r.SizeBytes).Render(plainBar)
			styledSize := sizeColor(r.SizeBytes).Render(plainSize)

			line := mark + " " + plainPat + " " + plainPath + " " + styledBar + " " + styledSize

			if m.focusPanel == focusResults && m.cursor == i {
				line = core.SelectedRowStyle.Render(clampW.Render(line))
			} else if m.checked[i] {
				line = core.ValueStyle.Render(clampW.Render(line))
			} else if r.AlreadyDone {
				line = core.LabelStyle.Render(clampW.Render(line))
			} else {
				line = clampW.Render(line)
			}
			lines = append(lines, line)
		}

		if m.scrollOff > 0 || end < len(m.results) {
			lines = append(lines, clampW.Render(core.LabelStyle.Render(fmt.Sprintf("  ↑ %d-%d / %d ↓", m.scrollOff+1, end, len(m.results)))))
		}
	}

	lines = append(lines, "")
	lines = append(lines, clampW.Render(core.LabelStyle.Render("enter:select a:all s:scan e:exclude ←→:tree h:hidden")))

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *ScannerModel) renderProgress(w, maxH int) string {
	inner := w - cardPad
	clampW := lipgloss.NewStyle().MaxWidth(inner)

	var lines []string
	
	// Title & Speed
	title := core.SelectedRowStyle.Render("  🚀 EXTREME SCAN IN PROGRESS")
	lines = append(lines, "", title, "")

	// Big Progress Bar (Simulated or based on items found)
	barW := inner - 10
	if barW < 20 { barW = 20 }
	// We use a pulsing animation style bar
	barFill := (m.currentProgress.ItemsFound % barW) + 1
	bar := core.StatusOK.Render(strings.Repeat("█", barFill) + strings.Repeat("░", barW-barFill))
	lines = append(lines, fmt.Sprintf("  Progress: [%s]", bar))
	lines = append(lines, "")

	// Stats Grid
	lines = append(lines, 
		fmt.Sprintf("  %s %-15d  %s %s", 
			core.LabelStyle.Render("Artifacts:"), m.currentProgress.ItemsFound,
			core.LabelStyle.Render("Total Size:"), core.StatusOK.Render(core.FormatBytes(m.currentProgress.TotalSize)),
		),
	)
	lines = append(lines, "")

	// Current Activity
	currentPath := core.TruncatePath(m.currentProgress.CurrentPath, inner-15)
	lines = append(lines, fmt.Sprintf("  %s %s", core.LabelStyle.Render("Scanning:"), core.ValueStyle.Render(currentPath)))
	lines = append(lines, "")

	// Live Feed Header
	lines = append(lines, core.LabelStyle.Render("  Live Feed (Last Hallucinations):"))
	for _, res := range m.liveFeed {
		path := core.TruncatePath(res.Path, inner-20)
		size := sizeColor(res.SizeBytes).Render(core.FormatBytes(res.SizeBytes))
		lines = append(lines, fmt.Sprintf("    %-10s %-30s %s", res.PatternName, path, size))
	}

	// Fill remaining space
	for len(lines) < maxH-3 {
		lines = append(lines, "")
	}

	return core.CardFocusedStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *ScannerModel) renderFooter(h int) string {
	inner := m.width - cardPad
	if inner < 10 {
		inner = 10
	}
	clampW := lipgloss.NewStyle().MaxWidth(inner)

	var selCount int
	var selSize int64
	for i, r := range m.results {
		if m.checked[i] {
			selCount++
			selSize += r.SizeBytes
		}
	}

	volCount := len(m.selectedVols)
	var line1 string
	if selCount > 0 || volCount > 0 {
		parts := []string{}
		if volCount > 0 {
			parts = append(parts, fmt.Sprintf("%d vols", volCount))
		}
		if selCount > 0 {
			parts = append(parts, fmt.Sprintf("%d items", selCount))
			parts = append(parts, core.FormatBytes(selSize))
		}
		barW := 20
		if inner < 60 {
			barW = 10
		}
		bar := sizeColor(selSize).Render(sizeBar(selSize, m.totalResultSize(), barW))
		pct := ""
		if m.totalResultSize() > 0 {
			pct = fmt.Sprintf("%.0f%%", float64(selSize)/float64(m.totalResultSize())*100)
		}
		line1 = core.StatusOK.Render("Selected: " + strings.Join(parts, " | ") + " " + bar + " " + pct)
	} else if len(m.results) > 0 {
		line1 = core.LabelStyle.Render(fmt.Sprintf("%d items | %s — enter to select", len(m.results), core.FormatBytes(m.totalResultSize())))
	} else {
		line1 = core.LabelStyle.Render("Select categories and press s to scan")
	}

	var line2 string
	if len(m.results) > 0 {
		// Group breakdown
		gmap := map[string]*struct{ count, sel int; total, selB int64 }{}
		for i, r := range m.results {
			c := catForID(r.PatternName)
			g := c.group
			st := gmap[g]
			if st == nil {
				st = &struct{ count, sel int; total, selB int64 }{}
				gmap[g] = st
			}
			st.count++
			st.total += r.SizeBytes
			if m.checked[i] {
				st.sel++
				st.selB += r.SizeBytes
			}
		}
		var gkeys []string
		for k := range gmap {
			gkeys = append(gkeys, k)
		}
		sort.Slice(gkeys, func(i, j int) bool { return gmap[gkeys[i]].total > gmap[gkeys[j]].total })
		var parts []string
		for _, k := range gkeys {
			st := gmap[k]
			s := fmt.Sprintf("%s:%d(%s)", k, st.count, core.FormatBytes(st.total))
			if st.sel > 0 {
				s += fmt.Sprintf(" sel:%d(%s)", st.sel, core.FormatBytes(st.selB))
			}
			parts = append(parts, sizeColor(st.total).Render(s))
		}
		line2 = strings.Join(parts, " ")
	} else if len(m.logEntries) > 0 {
		line2 = m.logEntries[len(m.logEntries)-1]
	}

	// Line 3: Top items
	var line3 string
	if len(m.results) > 0 {
		type itemInfo struct {
			name string
			size int64
		}
		var top []itemInfo
		for _, r := range m.results {
			if !r.AlreadyDone {
				top = append(top, itemInfo{r.PatternName, r.SizeBytes})
			}
		}
		sort.Slice(top, func(i, j int) bool { return top[i].size > top[j].size })
		if len(top) > 5 {
			top = top[:5]
		}
		var topParts []string
		for _, it := range top {
			topParts = append(topParts, sizeColor(it.size).Render(fmt.Sprintf("%s(%s)", it.name, core.FormatBytes(it.size))))
		}
		if len(topParts) > 0 {
			line3 = "Top: " + strings.Join(topParts, " ")
		}
	}

	// Line 4: Last log
	var line4 string
	if len(m.logEntries) > 0 {
		line4 = core.LabelStyle.Render(m.logEntries[len(m.logEntries)-1])
	}

	// Line 5: Previous log
	var line5 string
	if len(m.logEntries) > 1 {
		line5 = core.LabelStyle.Render(m.logEntries[len(m.logEntries)-2])
	}

	allLines := []string{clampW.Render(line1), clampW.Render(line2), clampW.Render(line3), clampW.Render(line4), clampW.Render(line5)}
	maxFooterLines := h - cardPad
	if maxFooterLines < len(allLines) {
		allLines = allLines[:maxFooterLines]
	}
	return core.CardStyle.Width(m.width).Height(h).Render(strings.Join(allLines, "\n"))
}
