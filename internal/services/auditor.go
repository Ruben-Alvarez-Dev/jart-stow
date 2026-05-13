// Package services — AuditService verifies exclusion consistency across backup systems
// and provides detailed project inspection for the CLI and TUI audit screens.
package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// ProjectInspection holds the full audit data for a single project.
type ProjectInclusion struct {
	Project    domain.Project   `json:"project"`
	Exclusions []domain.Exclusion `json:"exclusions"`
	TotalSize  int64            `json:"total_size"`
	ArtifactFolders int         `json:"artifact_folders"`
	HasIssues   bool            `json:"has_issues"`
	Issues      []string        `json:"issues,omitempty"`
}

// ExclusionVerdict records whether a database exclusion is actually applied on disk.
type ExclusionVerdict struct {
	Exclusion    domain.Exclusion `json:"exclusion"`
	AppliedOnDisk bool            `json:"applied_on_disk"`
	MissingFrom   []string        `json:"missing_from,omitempty"`
}

// AuditSummary is the top-level result of a full audit run.
type AuditSummary struct {
	TotalProjects    int                `json:"total_projects"`
	ActiveProjects   int                `json:"active_projects"`
	TotalExclusions  int                `json:"total_exclusions"`
	VerifiedExclusions int              `json:"verified_exclusions"`
	MissingExclusions int               `json:"missing_exclusions"`
	TotalSizeBytes   int64              `json:"total_size_bytes"`
	Verdicts         []ExclusionVerdict `json:"verdicts,omitempty"`
	Inspections      []ProjectInclusion `json:"inspections,omitempty"`
	Duration         time.Duration      `json:"duration"`
}

// AuditService verifies exclusion consistency and performs project inspection.
type AuditService struct {
	projectRepo    ports.ProjectRepository
	exclusionRepo  ports.ExclusionRepository
	backupProviders []ports.BackupProvider
}

// NewAuditService creates an AuditService with constructor injection.
func NewAuditService(
	projectRepo ports.ProjectRepository,
	exclusionRepo ports.ExclusionRepository,
	backupProviders ...ports.BackupProvider,
) *AuditService {
	return &AuditService{
		projectRepo:    projectRepo,
		exclusionRepo:  exclusionRepo,
		backupProviders: backupProviders,
	}
}

// InspectProject returns a full inspection of a single project by path.
func (s *AuditService) InspectProject(ctx context.Context, projectPath string) (*ProjectInclusion, error) {
	project, err := s.projectRepo.FindByPath(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("finding project %s: %w", projectPath, err)
	}

	exclusions, err := s.exclusionRepo.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("loading exclusions for project %d: %w", project.ID, err)
	}

	var totalSize int64
	var issues []string
	exclusionPaths := make(map[string]bool)

	for _, e := range exclusions {
		if e.IsActive() {
			totalSize += e.SizeBytes
			exclusionPaths[e.FolderPath] = true
			// Check if the excluded path still exists on disk
			if _, err := os.Stat(e.FolderPath); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("excluded path no longer exists: %s", e.FolderPath))
			}
		}
	}

	return &ProjectInclusion{
		Project:    *project,
		Exclusions: exclusions,
		TotalSize:  totalSize,
		ArtifactFolders: len(exclusionPaths),
		HasIssues:  len(issues) > 0,
		Issues:     issues,
	}, nil
}

// VerifyExclusions checks every active exclusion in the database against
// the actual backup system state (tmutil/CCC). Returns verdicts for each.
func (s *AuditService) VerifyExclusions(ctx context.Context) (*AuditSummary, error) {
	start := time.Now()

	exclusions, err := s.exclusionRepo.FindActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading active exclusions: %w", err)
	}

	totalProjects := 0
	activeProjects := 0
	if projects, err := s.projectRepo.FindAll(ctx); err == nil {
		totalProjects = len(projects)
		for _, p := range projects {
			if p.Status == domain.ProjectStatusActive {
				activeProjects++
			}
		}
	}

	var totalSize int64
	var verdicts []ExclusionVerdict
	verifiedCount := 0
	missingCount := 0

	for _, ex := range exclusions {
		var missingFrom []string

		for _, bp := range s.backupProviders {
			available, err := bp.IsAvailable(ctx)
			if err != nil || !available {
				continue
			}
			excluded, err := bp.IsExcluded(ctx, ex.FolderPath)
			if err != nil || !excluded {
				missingFrom = append(missingFrom, bp.Name())
			}
		}

		totalSize += ex.SizeBytes
		if len(missingFrom) > 0 {
			missingCount++
		} else {
			verifiedCount++
		}

		verdicts = append(verdicts, ExclusionVerdict{
			Exclusion:    ex,
			AppliedOnDisk: len(missingFrom) == 0,
			MissingFrom:  missingFrom,
		})
	}

	return &AuditSummary{
		TotalProjects:      totalProjects,
		ActiveProjects:     activeProjects,
		TotalExclusions:    len(exclusions),
		VerifiedExclusions: verifiedCount,
		MissingExclusions:  missingCount,
		TotalSizeBytes:     totalSize,
		Verdicts:           verdicts,
		Duration:           time.Since(start),
	}, nil
}

// FormatInspection returns a human-readable string of a project inspection.
func FormatInspection(pi *ProjectInclusion) string {
	var s string
	s += fmt.Sprintf("Project: %s\n", pi.Project.Name)
	s += fmt.Sprintf("  Path: %s\n", pi.Project.Path)
	s += fmt.Sprintf("  Status: %s\n", pi.Project.Status)
	s += fmt.Sprintf("  Root: %s\n", pi.Project.RootPath)
	if pi.Project.LastScanned != nil {
		s += fmt.Sprintf("  Last Scanned: %s\n", pi.Project.LastScanned.Format(time.RFC3339))
	}
	s += fmt.Sprintf("  Artifact Folders: %d\n", pi.ArtifactFolders)
	s += fmt.Sprintf("  Total Size: %s\n", formatBytesInt64(pi.TotalSize))
	s += fmt.Sprintf("  Exclusions: %d\n", len(pi.Exclusions))

	if pi.HasIssues {
		s += "\n  Issues:\n"
		for _, issue := range pi.Issues {
			s += fmt.Sprintf("    - %s\n", issue)
		}
	}

	s += "\n  Exclusions:\n"
	for _, e := range pi.Exclusions {
		status := "active"
		if !e.IsActive() {
			status = "removed"
		}
		ePath := e.FolderPath
		if len(ePath) > 60 {
			ePath = "..." + ePath[len(ePath)-57:]
		}
		s += fmt.Sprintf("    %-60s %8s %s\n", ePath, formatBytesInt64(e.SizeBytes), status)
	}

	return s
}

// FormatAuditSummary returns a human-readable string of the audit summary.
func FormatAuditSummary(summary *AuditSummary) string {
	var s string
	s += fmt.Sprintf("Audit completed in %v\n", summary.Duration.Round(time.Millisecond))
	s += fmt.Sprintf("  Projects: %d (%d active)\n", summary.TotalProjects, summary.ActiveProjects)
	s += fmt.Sprintf("  Exclusions: %d total\n", summary.TotalExclusions)
	s += fmt.Sprintf("  Verified: %d\n", summary.VerifiedExclusions)
	s += fmt.Sprintf("  Missing: %d\n", summary.MissingExclusions)
	s += fmt.Sprintf("  Total Size: %s\n", formatBytesInt64(summary.TotalSizeBytes))

	if summary.MissingExclusions > 0 {
		s += "\n  Missing Exclusions:\n"
		for _, v := range summary.Verdicts {
			if !v.AppliedOnDisk {
				ePath := v.Exclusion.FolderPath
				if len(ePath) > 50 {
					ePath = "..." + ePath[len(ePath)-47:]
				}
				s += fmt.Sprintf("    - %s (missing from: %v)\n", ePath, v.MissingFrom)
			}
		}
	}

	return s
}

// formatBytesInt64 formats bytes into a human-readable string.
func formatBytesInt64(bytes int64) string {
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


