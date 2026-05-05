// Package ports defines the interfaces (contracts) that services depend on.
// All adapters implement these interfaces. Services depend on ports, never on adapters.
package ports

import (
	"context"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
)

// ProjectRepository defines persistence operations for projects.
type ProjectRepository interface {
	FindAll(ctx context.Context) ([]domain.Project, error)
	FindByID(ctx context.Context, id int64) (*domain.Project, error)
	FindByPath(ctx context.Context, path string) (*domain.Project, error)
	FindByRootPath(ctx context.Context, rootPath string) ([]domain.Project, error)
	Upsert(ctx context.Context, project *domain.Project) (*domain.Project, error)
	UpdateStatus(ctx context.Context, id int64, status domain.ProjectStatus) error
	Delete(ctx context.Context, id int64) error
}

// ExclusionRepository defines persistence operations for exclusions.
type ExclusionRepository interface {
	FindAll(ctx context.Context) ([]domain.Exclusion, error)
	FindByID(ctx context.Context, id int64) (*domain.Exclusion, error)
	FindByProjectID(ctx context.Context, projectID int64) ([]domain.Exclusion, error)
	FindActive(ctx context.Context) ([]domain.Exclusion, error)
	FindByPath(ctx context.Context, folderPath string) (*domain.Exclusion, error)
	Save(ctx context.Context, exclusion *domain.Exclusion) (*domain.Exclusion, error)
	MarkRemoved(ctx context.Context, id int64) error
}

// RuleRepository defines persistence operations for hygiene rules.
type RuleRepository interface {
	FindAll(ctx context.Context) ([]domain.Rule, error)
	FindByID(ctx context.Context, id int64) (*domain.Rule, error)
	FindByProjectID(ctx context.Context, projectID int64) ([]domain.Rule, error)
	FindGlobalRules(ctx context.Context) ([]domain.Rule, error)
	Save(ctx context.Context, rule *domain.Rule) (*domain.Rule, error)
	Update(ctx context.Context, rule *domain.Rule) (*domain.Rule, error)
	Delete(ctx context.Context, id int64) error
}

// EventRepository defines persistence operations for daemon events.
type EventRepository interface {
	Log(ctx context.Context, event *domain.DaemonEvent) error
	FindRecent(ctx context.Context, limit int) ([]domain.DaemonEvent, error)
	FindByType(ctx context.Context, eventType domain.EventType, limit int) ([]domain.DaemonEvent, error)
	CountToday(ctx context.Context) (int, error)
}

// ScanJobRepository defines persistence operations for scan jobs.
type ScanJobRepository interface {
	Create(ctx context.Context, job *domain.ScanJob) (*domain.ScanJob, error)
	MarkCompleted(ctx context.Context, id int64, foldersFound int, totalSize int64) error
	MarkFailed(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*domain.ScanJob, error)
	FindByProjectID(ctx context.Context, projectID int64) ([]domain.ScanJob, error)
}

// WatchRootRepository defines persistence operations for watch roots.
type WatchRootRepository interface {
	FindAll(ctx context.Context) ([]domain.WatchRoot, error)
	FindByID(ctx context.Context, id int64) (*domain.WatchRoot, error)
	FindByPath(ctx context.Context, path string) (*domain.WatchRoot, error)
	Save(ctx context.Context, root *domain.WatchRoot) (*domain.WatchRoot, error)
	SetEnabled(ctx context.Context, id int64, enabled bool) error
	Delete(ctx context.Context, id int64) error
}

// JunkCategoryRepository defines persistence operations for junk categories.
type JunkCategoryRepository interface {
	FindAll(ctx context.Context) ([]domain.JunkCategory, error)
	FindByID(ctx context.Context, id int64) (*domain.JunkCategory, error)
	SetEnabled(ctx context.Context, id int64, enabled bool) error
	InsertDefaults(ctx context.Context) error
}

// JunkItemRepository defines persistence operations for junk items.
type JunkItemRepository interface {
	FindByCategory(ctx context.Context, categoryID int64) ([]domain.JunkItem, error)
	FindPending(ctx context.Context) ([]domain.JunkItem, error)
	FindByID(ctx context.Context, id int64) (*domain.JunkItem, error)
	SetVerification(ctx context.Context, id int64, status domain.VerificationStatus) error
	BatchSetVerification(ctx context.Context, ids []int64, status domain.VerificationStatus) (int, error)
	MarkCleaned(ctx context.Context, id int64) error
	Save(ctx context.Context, item *domain.JunkItem) (*domain.JunkItem, error)
}

// CleanupJobRepository defines persistence operations for cleanup jobs.
type CleanupJobRepository interface {
	Save(ctx context.Context, job *domain.CleanupJob) (*domain.CleanupJob, error)
	FindByID(ctx context.Context, id int64) (*domain.CleanupJob, error)
}
