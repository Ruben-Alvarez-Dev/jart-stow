package core

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
)

// TUIDependencies bundles all service dependencies for the TUI screens.
// Nil fields cause graceful degradation — screens show a banner when a required dep is missing.
type TUIDependencies struct {
	QuickExclude   *services.QuickExcludeService
	ScanService    *services.ScanService
	ExcludeService *services.ExcludeService
	Auditor        *services.AuditService
	Reporter       *services.ReportService
	JunkService    *services.JunkScanService

	ProjectRepo   ports.ProjectRepository
	ExclusionRepo ports.ExclusionRepository
	RuleRepo      ports.RuleRepository
	EventRepo     ports.EventRepository
	JunkCatRepo   ports.JunkCategoryRepository
	JunkItemRepo  ports.JunkItemRepository
	WatchRootRepo ports.WatchRootRepository

	DBAvailable bool
}

// Colors matching the TUI SPEC palette.
var (
	ColorPrimary   = lipgloss.Color("6")
	ColorSuccess   = lipgloss.Color("2")
	ColorWarning   = lipgloss.Color("3")
	ColorDanger    = lipgloss.Color("1")
	ColorMuted     = lipgloss.Color("8")
	ColorHighlight = lipgloss.Color("5")
)

// LayoutMode represents responsive layout modes.
type LayoutMode int

const (
	LayoutWide    LayoutMode = iota // >= 120 cols
	LayoutNormal                    // 80-119 cols
	LayoutCompact                   // < 80 cols
)

// ResponsiveLayout returns the layout mode for the given terminal width.
func ResponsiveLayout(width int) LayoutMode {
	switch {
	case width >= 120:
		return LayoutWide
	case width >= 80:
		return LayoutNormal
	default:
		return LayoutCompact
	}
}

// Styles
var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1)

	NavBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			Padding(0, 1)

	CardStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	CardFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	LabelStyle     = lipgloss.NewStyle().Foreground(ColorMuted)
	ValueStyle     = lipgloss.NewStyle().Bold(true)
	StatusOK       = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StatusWarn     = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	StatusErr      = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
	SelectedRowStyle = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)

	DegradeBannerStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				BorderForeground(ColorWarning).
				BorderStyle(lipgloss.RoundedBorder()).
				Padding(0, 1)
)

// FormatBytes converts bytes to a human-readable string.
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	var i int
	for i = 0; i < len(units)-1 && size >= 1024; i++ {
		size /= 1024
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}

// TruncatePath shortens a path to fit within maxLen, keeping the tail.
func TruncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if maxLen < 4 {
		return path[:maxLen]
	}
	return "..." + path[len(path)-maxLen+3:]
}

// StatusIcon returns a colored status indicator.
func StatusIcon(ok bool) string {
	if ok {
		return StatusOK.Render("✓")
	}
	return StatusWarn.Render("⚠")
}

// RunningIcon returns a colored dot for running processes.
func RunningIcon() string {
	return StatusOK.Render("●")
}

// StoppedIcon returns a hollow dot for stopped processes.
func StoppedIcon() string {
	return LabelStyle.Render("○")
}

// Card renders a bordered panel with a title and body content.
func Card(title, body string, width int, focused bool) string {
	style := CardStyle
	if focused {
		style = CardFocusedStyle
	}
	style = style.Width(width)
	header := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(title)
	return style.Render(header + "\n" + body)
}

// DegradeBanner renders a warning banner for missing database.
func DegradeBanner(message string) string {
	return DegradeBannerStyle.Render("⚠ " + message)
}

// SparkLine renders a mini chart using Unicode block characters.
func SparkLine(values []int64, maxVal int64, width int) string {
	if len(values) == 0 || width == 0 {
		return ""
	}
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var sampled []int64
	if len(values) > width {
		step := float64(len(values)) / float64(width)
		for i := 0; i < width; i++ {
			idx := int(float64(i) * step)
			sampled = append(sampled, values[idx])
		}
	} else {
		sampled = values
	}
	var sb strings.Builder
	for _, v := range sampled {
		if maxVal == 0 {
			sb.WriteRune(blocks[0])
			continue
		}
		idx := int(float64(v) / float64(maxVal) * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}

// BarChart renders a horizontal bar chart.
func BarChart(label string, value, maxVal int64, width int) string {
	if maxVal == 0 || width == 0 {
		return ""
	}
	barLen := int(float64(value) / float64(maxVal) * float64(width))
	if barLen > width {
		barLen = width
	}
	bar := strings.Repeat("█", barLen)
	pct := fmt.Sprintf("%.0f%%", float64(value)/float64(maxVal)*100)
	return fmt.Sprintf("  %-20s %s %s", label, bar, pct)
}

// Message types --------------------------------------------------------

// ResizeMsg propagates terminal size changes to screens.
type ResizeMsg struct {
	Width  int
	Height int
}

// Dashboard
type DashboardLoadMsg struct {
	Summary *services.ReportSummary
	Events  []domain.DaemonEvent
	Err     error
}

// Scanner
type ScanStartMsg struct{ Root string }

type ScanResultMsg struct {
	Results []services.QuickScanResult
	Err     error
}

type ExcludeResultMsg struct{ Failures map[string]error }

// Exclusions
type ExclusionsLoadMsg struct {
	Exclusions []domain.Exclusion
	Err        error
}

type ExclusionRemovedMsg struct {
	ID  int64
	Err error
}

// Rules
type RulesLoadMsg struct {
	Rules []domain.Rule
	Err   error
}

type RuleSavedMsg struct {
	Rule *domain.Rule
	Err  error
}

type RuleDeletedMsg struct {
	ID  int64
	Err error
}

// Audit
type AuditLoadMsg struct {
	Projects []domain.Project
	Err      error
}

type AuditInspectMsg struct {
	Inspection *services.ProjectInclusion
	Err        error
}

type AuditVerifyMsg struct {
	Summary *services.AuditSummary
	Err     error
}

// Hygiene
type HygieneCategoriesLoadMsg struct {
	Categories []domain.JunkCategory
	Err        error
}

type HygieneItemsLoadMsg struct {
	Items []domain.JunkItem
	Err   error
}

type HygieneScanMsg struct {
	CategoryID int64
	Items      []domain.JunkItem
	Err        error
}

type HygieneVerifyMsg struct {
	IDs    []int64
	Status domain.VerificationStatus
	Err    error
}

// Report
type ReportLoadMsg struct {
	Summary *services.ReportSummary
	History []services.HistoryEntry
	Err     error
}
