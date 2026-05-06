// Package screens — Action interfaces for functional TUI screens.
// These interfaces define the operations that screens can perform,
// bridging the gap between display and real service execution.
package screens

import "github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"

// ============================================================================
// ScreenScanEngine defines scan operations for the Scanner screen.
// ============================================================================

// ScreenScanEngine allows the Scanner screen to list volumes, browse directories, and trigger scans.
type ScreenScanEngine interface {
	// ScanPath scans a directory for development artifacts.
	ScanPath(path string) ([]ScanResult, error)

	// ListVolumes returns available mount points for browsing.
	ListVolumes() ([]Volume, error)

	// BrowseDirectory lists immediate subdirectories of a path.
	BrowseDirectory(path string) ([]Volume, error)
}

// ScanResult represents a single found artifact.
type ScanResult struct {
	Path        string
	PatternName string
	SizeBytes   int64
}

// Volume represents a mount point or directory for browsing.
type Volume struct {
	Path  string
	Name  string
	IsDir bool
}

// ============================================================================
// ScreenJunkScanRunner defines junk scan operations for the Hygiene screen.
// ============================================================================

// ScreenJunkScanRunner allows the Hygiene screen to trigger junk scans and manage items.
type ScreenJunkScanRunner interface {
	// ScanCategory runs a junk scan for the given category.
	ScanCategory(category domain.JunkCategory) ([]domain.JunkItem, error)

	// ApproveItem marks a junk item as approved for cleanup.
	ApproveItem(itemID int64) error

	// SkipItem marks a junk item as skipped (ignored).
	SkipItem(itemID int64) error

	// BatchApprove marks multiple items as approved.
	BatchApprove(ids []int64) error

	// BatchSkip marks multiple items as skipped.
	BatchSkip(ids []int64) error

	// ListScanners returns registered scanner names.
	ListScanners() []string
}

// ============================================================================
// ScreenExclusionManager defines exclusion operations for the Exclusions screen.
// ============================================================================

// ScreenExclusionManager allows the Exclusions screen to manage backup exclusions.
type ScreenExclusionManager interface {
	// ExcludeProject scans and excludes a project path.
	ExcludeProject(projectPath string) (int, int64, error)

	// RemoveExclusion removes an exclusion (marks it as removed in DB).
	RemoveExclusion(exclusionID int64) error
}
