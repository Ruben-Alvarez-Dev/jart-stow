// Package tui — Action interface implementations for TUI screens.
// These bridge the TUI screens to real service implementations.
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/screens"
)

// ============================================================================
// ScanEngineImpl
// ============================================================================

// ScanEngineImpl implements screens.ScreenScanEngine using the real ScanService.
type ScanEngineImpl struct {
	scanService *services.ScanService
}

// NewScanEngineImpl creates a ScanEngineImpl wired to a ScanService.
func NewScanEngineImpl(svc *services.ScanService) *ScanEngineImpl {
	return &ScanEngineImpl{scanService: svc}
}

// ScanPath scans a directory and returns found artifacts.
func (e *ScanEngineImpl) ScanPath(path string) ([]screens.ScanResult, error) {
	artifacts, err := e.scanService.FindArtifacts(ctx(), path)
	if err != nil {
		return nil, err
	}
	results := make([]screens.ScanResult, len(artifacts))
	for i, a := range artifacts {
		results[i] = screens.ScanResult{
			Path:        a.Path,
			PatternName: a.PatternName,
			SizeBytes:   a.SizeBytes,
		}
	}
	return results, nil
}

// ListVolumes returns available mount points for browsing.
func (e *ScanEngineImpl) ListVolumes() ([]screens.Volume, error) {
	var volumes []screens.Volume

	out, err := exec.Command("df", "-h", "-t", "apfs").CombinedOutput()
	if err != nil {
		volumes = append(volumes, screens.Volume{Path: "/", Name: "Root (/)", IsDir: true})
		entries, _ := os.ReadDir("/Volumes")
		for _, entry := range entries {
			if entry.IsDir() {
				volumes = append(volumes, screens.Volume{
					Path:  filepathJoin("/Volumes", entry.Name()),
					Name:  entry.Name(),
					IsDir: true,
				})
			}
		}
	} else {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			mountPoint := fields[len(fields)-1]
			if mountPoint == "/" || strings.HasPrefix(mountPoint, "/Volumes/") {
				name := mountPoint
				if mountPoint == "/" {
					name = "Macintosh HD"
				} else {
					name = filepathBase(mountPoint)
				}
				volumes = append(volumes, screens.Volume{
					Path:  mountPoint,
					Name:  name,
					IsDir: true,
				})
			}
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		volumes = append(volumes, screens.Volume{
			Path:  home,
			Name:  "Home",
			IsDir: true,
		})
	}

	return volumes, nil
}

// BrowseDirectory lists immediate subdirectories of a path.
func (e *ScanEngineImpl) BrowseDirectory(path string) ([]screens.Volume, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", path, err)
	}
	var dirs []screens.Volume
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, screens.Volume{
				Path:  filepathJoin(path, entry.Name()),
				Name:  entry.Name() + "/",
				IsDir: true,
			})
		}
	}
	return dirs, nil
}

// ============================================================================
// JunkScanRunnerImpl
// ============================================================================

// JunkScanRunnerImpl implements screens.ScreenJunkScanRunner using JunkScanService and repos.
type JunkScanRunnerImpl struct {
	junkService *services.JunkScanService
	itemRepo    ports.JunkItemRepository
}

// NewJunkScanRunnerImpl creates a JunkScanRunnerImpl.
func NewJunkScanRunnerImpl(junkService *services.JunkScanService, itemRepo ports.JunkItemRepository) *JunkScanRunnerImpl {
	return &JunkScanRunnerImpl{
		junkService: junkService,
		itemRepo:    itemRepo,
	}
}

// ScanCategory runs a scan for one junk category and saves items to DB.
func (r *JunkScanRunnerImpl) ScanCategory(category domain.JunkCategory) ([]domain.JunkItem, error) {
	items, err := r.junkService.ScanCategory(ctx(), category)
	if err != nil {
		return nil, err
	}
	var saved []domain.JunkItem
	for _, item := range items {
		savedItem, err := r.itemRepo.Save(ctx(), &item)
		if err != nil {
			continue
		}
		if savedItem != nil {
			saved = append(saved, *savedItem)
		}
	}
	if len(saved) == 0 && err == nil {
		return items, nil
	}
	return saved, nil
}

// ApproveItem marks a single item as approved.
func (r *JunkScanRunnerImpl) ApproveItem(itemID int64) error {
	return r.itemRepo.SetVerification(ctx(), itemID, domain.VerificationApproved)
}

// SkipItem marks a single item as skipped.
func (r *JunkScanRunnerImpl) SkipItem(itemID int64) error {
	return r.itemRepo.SetVerification(ctx(), itemID, domain.VerificationSkipped)
}

// BatchApprove approves multiple items.
func (r *JunkScanRunnerImpl) BatchApprove(ids []int64) error {
	_, err := r.itemRepo.BatchSetVerification(ctx(), ids, domain.VerificationApproved)
	return err
}

// BatchSkip skips multiple items.
func (r *JunkScanRunnerImpl) BatchSkip(ids []int64) error {
	_, err := r.itemRepo.BatchSetVerification(ctx(), ids, domain.VerificationSkipped)
	return err
}

// ListScanners returns registered scanner names.
func (r *JunkScanRunnerImpl) ListScanners() []string {
	return r.junkService.RegisteredScanners()
}

// ============================================================================
// ExclusionManagerImpl
// ============================================================================

// ExclusionManagerImpl implements screens.ScreenExclusionManager.
type ExclusionManagerImpl struct {
	excludeService *services.ExcludeService
	exclusionRepo  ports.ExclusionRepository
}

// NewExclusionManagerImpl creates an ExclusionManagerImpl.
func NewExclusionManagerImpl(es *services.ExcludeService, er ports.ExclusionRepository) *ExclusionManagerImpl {
	return &ExclusionManagerImpl{
		excludeService: es,
		exclusionRepo:  er,
	}
}

// ExcludeProject scans and excludes a project path.
func (m *ExclusionManagerImpl) ExcludeProject(projectPath string) (int, int64, error) {
	return m.excludeService.ExcludeProject(ctx(), projectPath)
}

// RemoveExclusion removes an exclusion record from the database.
func (m *ExclusionManagerImpl) RemoveExclusion(exclusionID int64) error {
	return m.exclusionRepo.MarkRemoved(ctx(), exclusionID)
}

// ============================================================================
// QuickExcludeImpl
// ============================================================================

// QuickExcludeImpl implements screens.ScreenQuickExclude using the
// QuickExcludeService. This provides the direct scan-and-exclude flow
// that EXCLUSION-SCRIPT had, without needing projects in the database.
type QuickExcludeImpl struct {
	quickService *services.QuickExcludeService
}

// NewQuickExcludeImpl creates a QuickExcludeImpl.
func NewQuickExcludeImpl(qs *services.QuickExcludeService) *QuickExcludeImpl {
	return &QuickExcludeImpl{quickService: qs}
}

// ScanPath scans a path for development artifacts.
func (q *QuickExcludeImpl) ScanPath(path string) ([]screens.QuickScanResult, error) {
	results, err := q.quickService.Scan(ctx(), path)
	if err != nil {
		return nil, err
	}
	converted := make([]screens.QuickScanResult, len(results))
	for i, r := range results {
		converted[i] = screens.QuickScanResult{
			Path:        r.Path,
			PatternName: r.PatternName,
			SizeBytes:   r.SizeBytes,
			AlreadyDone: r.AlreadyDone,
		}
	}
	return converted, nil
}

// ExcludePaths applies exclusions to all configured backup systems.
func (q *QuickExcludeImpl) ExcludePaths(paths []string) map[string]error {
	return q.quickService.ExcludePaths(ctx(), paths)
}

// RemoveExclusions removes exclusions from all configured backup systems.
func (q *QuickExcludeImpl) RemoveExclusions(paths []string) map[string]error {
	return q.quickService.RemoveExclusions(ctx(), paths)
}

// ListExclusions returns all currently excluded paths per backup system.
func (q *QuickExcludeImpl) ListExclusions() (map[string][]string, error) {
	return q.quickService.ListExclusions(ctx())
}

// GetVolumes returns available volumes (Home + mounted drives).
func (q *QuickExcludeImpl) GetVolumes() []domain.Volume {
	return services.GetVolumes()
}

// ============================================================================
// Helpers
// ============================================================================

func filepathJoin(elem ...string) string {
	var b strings.Builder
	for i, e := range elem {
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(e)
	}
	return b.String()
}

func filepathBase(path string) string {
	idx := strings.LastIndexByte(path, '/')
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}
