// Package tui provides the Bubble Tea terminal user interface.
// This file implements the data providers that bridge the TUI screens to the SQLite repositories.
package tui

import (
	"context"
	"os/exec"
	"strings"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/screens"
)

// TUIProviders bundles all data providers and action interfaces for the TUI screens.
type TUIProviders struct {
	Daemon     *DaemonProvider
	WatchRoots *WatchRootsProvider
	Projects   *ProjectsProvider
	Exclusions *ExclusionsProvider
	Events     *EventsProvider
	Rules      *RulesProvider
	Junk       *JunkProvider

	// Action interfaces
	ScanEngine       screens.ScreenScanEngine
	JunkScanRunner   screens.ScreenJunkScanRunner
	ExclusionManager screens.ScreenExclusionManager
}

// NewTUIProviders creates all TUI data providers wired to real SQLite repositories.
func NewTUIProviders(
	projectRepo ports.ProjectRepository,
	exclusionRepo ports.ExclusionRepository,
	eventRepo ports.EventRepository,
	ruleRepo ports.RuleRepository,
	watchRootRepo ports.WatchRootRepository,
	junkCategoryRepo ports.JunkCategoryRepository,
	junkItemRepo ports.JunkItemRepository,
	scanEngine screens.ScreenScanEngine,
	junkScanRunner screens.ScreenJunkScanRunner,
	exclusionManager screens.ScreenExclusionManager,
) *TUIProviders {
	return &TUIProviders{
		Daemon:     &DaemonProvider{},
		WatchRoots: &WatchRootsProvider{repo: watchRootRepo},
		Projects:   &ProjectsProvider{repo: projectRepo},
		Exclusions: &ExclusionsProvider{repo: exclusionRepo},
		Events:     &EventsProvider{repo: eventRepo},
		Rules:      &RulesProvider{repo: ruleRepo},
		Junk: &JunkProvider{
			categoryRepo: junkCategoryRepo,
			itemRepo:     junkItemRepo,
		},
		ScanEngine:       scanEngine,
		JunkScanRunner:   junkScanRunner,
		ExclusionManager: exclusionManager,
	}
}

// ctx returns a background context for database operations.
func ctx() context.Context {
	return context.Background()
}

// ---- DaemonStatusProvider ----

// DaemonProvider checks whether the jart-stow daemon is running via launchctl.
type DaemonProvider struct{}

// IsRunning returns true if the daemon launchd job is loaded.
func (p *DaemonProvider) IsRunning() bool {
	cmd := exec.Command("launchctl", "list", "dev.rubenalvarez.jart-stow")
	err := cmd.Run()
	return err == nil
}

// ---- WatchRootProvider ----

// WatchRootsProvider reads watch roots from the database.
type WatchRootsProvider struct {
	repo ports.WatchRootRepository
}

// ListWatchRoots returns all configured watch roots.
func (p *WatchRootsProvider) ListWatchRoots() ([]domain.WatchRoot, error) {
	if p.repo == nil {
		return nil, nil
	}
	return p.repo.FindAll(ctx())
}

// ---- ProjectLister ----

// ProjectsProvider reads project data from the database.
type ProjectsProvider struct {
	repo ports.ProjectRepository
}

// ListProjects returns all projects.
func (p *ProjectsProvider) ListProjects() ([]domain.Project, error) {
	if p.repo == nil {
		return nil, nil
	}
	return p.repo.FindAll(ctx())
}

// CountProjects returns the total number of active projects.
func (p *ProjectsProvider) CountProjects() (int, error) {
	if p.repo == nil {
		return 0, nil
	}
	projects, err := p.repo.FindAll(ctx())
	if err != nil {
		return 0, err
	}
	count := 0
	for _, proj := range projects {
		if proj.Status == domain.ProjectStatusActive {
			count++
		}
	}
	return count, nil
}

// ---- ExclusionLister ----

// ExclusionsProvider reads exclusion data from the database.
type ExclusionsProvider struct {
	repo ports.ExclusionRepository
}

// ListExclusions returns all active exclusions.
func (p *ExclusionsProvider) ListExclusions() ([]domain.Exclusion, error) {
	if p.repo == nil {
		return nil, nil
	}
	return p.repo.FindActive(ctx())
}

// CountExclusions returns the total number of active exclusions.
func (p *ExclusionsProvider) CountExclusions() (int, error) {
	if p.repo == nil {
		return 0, nil
	}
	exclusions, err := p.repo.FindActive(ctx())
	if err != nil {
		return 0, err
	}
	return len(exclusions), nil
}

// TotalSpaceSaved returns the total bytes excluded.
func (p *ExclusionsProvider) TotalSpaceSaved() (int64, error) {
	if p.repo == nil {
		return 0, nil
	}
	exclusions, err := p.repo.FindActive(ctx())
	if err != nil {
		return 0, err
	}
	var total int64
	for _, e := range exclusions {
		total += e.SizeBytes
	}
	return total, nil
}

// ---- EventLister ----

// EventsProvider reads daemon events from the database.
type EventsProvider struct {
	repo ports.EventRepository
}

// ListRecentEvents returns the most recent daemon events.
func (p *EventsProvider) ListRecentEvents(limit int) ([]domain.DaemonEvent, error) {
	if p.repo == nil {
		return nil, nil
	}
	return p.repo.FindRecent(ctx(), limit)
}

// ---- RuleLister ----

// RulesProvider reads rules from the database.
type RulesProvider struct {
	repo ports.RuleRepository
}

// ListGlobalRules returns global-level rules (no project_id).
func (p *RulesProvider) ListGlobalRules() ([]domain.Rule, error) {
	if p.repo == nil {
		return nil, nil
	}
	return p.repo.FindGlobalRules(ctx())
}

// ListProjectRules returns project-specific rules.
func (p *RulesProvider) ListProjectRules() ([]domain.Rule, error) {
	if p.repo == nil {
		return nil, nil
	}
	all, err := p.repo.FindAll(ctx())
	if err != nil {
		return nil, err
	}
	var projectRules []domain.Rule
	for _, r := range all {
		if r.ProjectID != nil {
			projectRules = append(projectRules, r)
		}
	}
	return projectRules, nil
}

// ---- JunkLister ----

// JunkProvider reads junk categories and items from the database.
type JunkProvider struct {
	categoryRepo ports.JunkCategoryRepository
	itemRepo     ports.JunkItemRepository
}

// ListCategories returns all junk categories.
func (p *JunkProvider) ListCategories() ([]domain.JunkCategory, error) {
	if p.categoryRepo == nil {
		return nil, nil
	}
	return p.categoryRepo.FindAll(ctx())
}

// ListPendingItemsByCategory returns pending items for a specific category.
func (p *JunkProvider) ListPendingItemsByCategory(categoryID int64) ([]domain.JunkItem, error) {
	if p.itemRepo == nil {
		return nil, nil
	}
	items, err := p.itemRepo.FindByCategory(ctx(), categoryID)
	if err != nil {
		return nil, err
	}
	var pending []domain.JunkItem
	for _, item := range items {
		if item.VerifiedByUser == domain.VerificationPending {
			pending = append(pending, item)
		}
	}
	return pending, nil
}

// CountPendingItems returns the total number of unverified junk items.
func (p *JunkProvider) CountPendingItems() (int, error) {
	if p.itemRepo == nil {
		return 0, nil
	}
	items, err := p.itemRepo.FindPending(ctx())
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

// normalizeEventType maps CLI event type strings to domain constants.
func normalizeEventType(t string) domain.EventType {
	switch strings.ToLower(t) {
	case "project_detected":
		return domain.EventTypeProjectDetected
	case "scan_completed":
		return domain.EventTypeScanCompleted
	case "exclusion_applied":
		return domain.EventTypeExclusionApplied
	case "exclusion_removed":
		return domain.EventTypeExclusionRemoved
	case "error":
		return domain.EventTypeError
	case "daemon_started":
		return domain.EventTypeDaemonStarted
	case "daemon_stopped":
		return domain.EventTypeDaemonStopped
	default:
		return domain.EventType(t)
	}
}
