// Package screens implements the Bubble Tea v2 screen models for the TUI.
package screens

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"
)

// AuditModel is the Bubble Tea model for the Audit screen.
type AuditModel struct {
	deps        *core.TUIDependencies
	width       int
	height      int
	projects    []domain.Project
	cursor      int
	scrollOff   int
	inspections map[int64]*services.ProjectInclusion
	verifying   bool
	spinner     spinner.Model
	loading     bool
	summary     *services.AuditSummary
}

// NewAuditModel creates a new AuditModel with the given dependencies.
func NewAuditModel(deps *core.TUIDependencies) *AuditModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return &AuditModel{
		deps:        deps,
		inspections: make(map[int64]*services.ProjectInclusion),
		spinner:     s,
	}
}

// Init loads the initial project list.
func (m *AuditModel) Init() tea.Cmd {
	if m.deps == nil || !m.deps.DBAvailable || m.deps.ProjectRepo == nil {
		return nil
	}
	m.loading = true
	return func() tea.Msg {
		projects, err := m.deps.ProjectRepo.FindAll(context.Background())
		return core.AuditLoadMsg{Projects: projects, Err: err}
	}
}

// Update handles messages for the Audit screen.
func (m *AuditModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case core.ResizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case core.AuditLoadMsg:
		m.loading = false
		if msg.Err != nil {
			return m, nil
		}
		m.projects = msg.Projects
		m.scrollOff = 0
		if m.cursor >= len(m.projects) {
			m.cursor = len(m.projects) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil

	case core.AuditInspectMsg:
		if msg.Err != nil || msg.Inspection == nil {
			return m, nil
		}
		m.inspections[msg.Inspection.Project.ID] = msg.Inspection
		return m, nil

	case core.AuditVerifyMsg:
		m.verifying = false
		if msg.Err == nil && msg.Summary != nil {
			m.summary = msg.Summary
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.projects)-1 {
				m.cursor++
			}
			m.clampScroll()
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			m.clampScroll()
		case "enter":
			if len(m.projects) == 0 {
				return m, nil
			}
			selected := m.projects[m.cursor]
			return m, func() tea.Msg {
				insp, err := m.deps.Auditor.InspectProject(context.Background(), selected.Path)
				return core.AuditInspectMsg{Inspection: insp, Err: err}
			}
		case "a":
			if m.deps.Auditor == nil || m.verifying {
				return m, nil
			}
			m.verifying = true
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				summary, err := m.deps.Auditor.VerifyExclusions(context.Background())
				return core.AuditVerifyMsg{Summary: summary, Err: err}
			})
		case "r":
			return m, m.Init()
		}
	}

	return m, nil
}

func (m *AuditModel) clampScroll() {
	visible := m.projectVisibleCount()
	if visible <= 0 {
		visible = 1
	}
	if m.cursor < m.scrollOff {
		m.scrollOff = m.cursor
	}
	if m.cursor >= m.scrollOff+visible {
		m.scrollOff = m.cursor - visible + 1
	}
	maxOff := len(m.projects) - visible
	if maxOff < 0 {
		maxOff = 0
	}
	if m.scrollOff > maxOff {
		m.scrollOff = maxOff
	}
}

func (m *AuditModel) projectVisibleCount() int {
	if m.height == 0 {
		return 10
	}
	bottomH := 2 // summary bar + help line
	contentH := m.height - bottomH
	inner := contentH - cardPad - 2 // card overhead + title + separator
	if inner < 3 {
		inner = 3
	}
	return inner
}

// View renders the Audit screen.
func (m *AuditModel) View() tea.View {
	if m.deps == nil || !m.deps.DBAvailable {
		return tea.NewView(core.DegradeBanner("Database unavailable -- audit data cannot be loaded."))
	}

	if m.loading {
		return tea.NewView(fmt.Sprintf("\n  %s Loading projects...", m.spinner.View()))
	}

	if m.width == 0 || m.height == 0 {
		return tea.NewView("Initializing audit...")
	}

	layout := core.ResponsiveLayout(m.width)

	bottomH := 2
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

	left := m.renderLeft(leftW, contentH)
	right := m.renderRight(rightW, contentH)

	var content string
	if layout == core.LayoutCompact {
		content = left + "\n" + right
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	// Summary bar
	var summaryParts []string
	if m.summary != nil {
		summaryParts = append(summaryParts,
			fmt.Sprintf("Projects: %d/%d", m.summary.ActiveProjects, m.summary.TotalProjects),
			fmt.Sprintf("Verified: %d", m.summary.VerifiedExclusions),
			fmt.Sprintf("Missing: %d", m.summary.MissingExclusions),
			fmt.Sprintf("Size: %s", core.FormatBytes(m.summary.TotalSizeBytes)),
		)
	} else {
		summaryParts = append(summaryParts, "Press 'a' to audit all exclusions")
	}
	summaryBar := lipgloss.NewStyle().Width(m.width).Render(strings.Join(summaryParts, "  │  "))

	if m.verifying {
		summaryBar += fmt.Sprintf("\n  %s Verifying exclusions...", m.spinner.View())
	}

	helpLine := core.LabelStyle.Render("  j/k:navigate  enter:inspect  a:audit all  r:reload")

	full := lipgloss.JoinVertical(lipgloss.Left, content, summaryBar, helpLine)
	return tea.NewView(full)
}

func (m *AuditModel) renderLeft(w, maxH int) string {
	inner := w - cardPad
	if inner < 10 {
		inner = 10
	}

	var lines []string
	lines = append(lines, fit("Projects", inner))

	if len(m.projects) == 0 {
		lines = append(lines, core.LabelStyle.Render("  No projects found."))
	} else {
		m.clampScroll()
		visible := m.projectVisibleCount()
		end := m.scrollOff + visible
		if end > len(m.projects) {
			end = len(m.projects)
		}

		for i := m.scrollOff; i < end; i++ {
			p := m.projects[i]
			label := fmt.Sprintf("  %s %s", core.StatusIcon(m.projectOK(p.ID)), fit(core.TruncatePath(p.Name, inner-6), inner-6))
			if i == m.cursor {
				label = core.SelectedRowStyle.Render(fit(label, inner))
			}
			lines = append(lines, label)
		}

		if m.scrollOff > 0 || end < len(m.projects) {
			info := fmt.Sprintf("  ↑ %d-%d / %d ↓", m.scrollOff+1, end, len(m.projects))
			lines = append(lines, core.LabelStyle.Render(fit(info, inner)))
		}
	}

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

func (m *AuditModel) renderRight(w, maxH int) string {
	inner := w - cardPad
	if inner < 20 {
		inner = 20
	}

	var lines []string
	lines = append(lines, fit("Project Detail", inner))

	if len(m.projects) > 0 && m.cursor < len(m.projects) {
		selected := m.projects[m.cursor]
		insp, hasInsp := m.inspections[selected.ID]

		lines = append(lines, core.LabelStyle.Render("  Name: ")+core.ValueStyle.Render(selected.Name))
		lines = append(lines, core.LabelStyle.Render("  Path: ")+core.ValueStyle.Render(core.TruncatePath(selected.Path, inner-10)))
		lines = append(lines, core.LabelStyle.Render("  Status: ")+core.ValueStyle.Render(string(selected.Status)))

		if hasInsp {
			lines = append(lines, "")
			lines = append(lines, core.LabelStyle.Render("  Artifact Folders: ")+core.ValueStyle.Render(fmt.Sprintf("%d", insp.ArtifactFolders)))
			lines = append(lines, core.LabelStyle.Render("  Total Size: ")+core.ValueStyle.Render(core.FormatBytes(insp.TotalSize)))
			lines = append(lines, core.LabelStyle.Render("  Exclusions: ")+core.ValueStyle.Render(fmt.Sprintf("%d", len(insp.Exclusions))))

			if insp.HasIssues {
				lines = append(lines, "")
				lines = append(lines, core.StatusErr.Render("  Issues:"))
				for _, issue := range insp.Issues {
					lines = append(lines, core.StatusWarn.Render("    - "+issue))
				}
			}

			if len(insp.Exclusions) > 0 {
				lines = append(lines, "")
				lines = append(lines, core.LabelStyle.Render("  Exclusion Details:"))
				for _, ex := range insp.Exclusions {
					status := "active"
					if !ex.IsActive() {
						status = "removed"
					}
					line := fmt.Sprintf("    %s  %s  %s",
						core.TruncatePath(ex.FolderPath, inner-20),
						core.FormatBytes(ex.SizeBytes),
						status,
					)
					lines = append(lines, line)
				}
			}
		} else {
			lines = append(lines, "")
			lines = append(lines, core.LabelStyle.Render("  Press Enter to inspect"))
		}
	} else {
		lines = append(lines, core.LabelStyle.Render("  No project selected"))
	}

	return core.CardStyle.Width(w).Height(maxH).Render(strings.Join(lines, "\n"))
}

// projectOK returns true if the project has been inspected and has no issues.
func (m *AuditModel) projectOK(id int64) bool {
	insp, ok := m.inspections[id]
	if !ok {
		return true
	}
	return !insp.HasIssues
}
