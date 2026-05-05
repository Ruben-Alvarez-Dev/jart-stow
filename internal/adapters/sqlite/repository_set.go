package sqlite

import (
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// RepositorySet bundles all SQLite repository implementations for dependency injection.
// Create one via NewRepositorySet(db) and pass individual repos to services.
type RepositorySet struct {
	Projects       ports.ProjectRepository
	Exclusions     ports.ExclusionRepository
	Rules          ports.RuleRepository
	Events         ports.EventRepository
	ScanJobs       ports.ScanJobRepository
	WatchRoots     ports.WatchRootRepository
	JunkCategories ports.JunkCategoryRepository
	JunkItems      ports.JunkItemRepository
	CleanupJobs    ports.CleanupJobRepository
}

// NewRepositorySet creates all repository implementations backed by the given database connection.
func NewRepositorySet(db *Connection) *RepositorySet {
	sqlDB := db.DB()
	return &RepositorySet{
		Projects:       NewProjectRepo(sqlDB),
		Exclusions:     NewExclusionRepo(sqlDB),
		Rules:          NewRuleRepo(sqlDB),
		Events:         NewEventRepo(sqlDB),
		ScanJobs:       NewScanJobRepo(sqlDB),
		WatchRoots:     NewWatchRootRepo(sqlDB),
		JunkCategories: NewJunkCategoryRepo(sqlDB),
		JunkItems:      NewJunkItemRepo(sqlDB),
		CleanupJobs:    NewCleanupJobRepo(sqlDB),
	}
}
