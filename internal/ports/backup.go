package ports

import "context"

// BackupProvider defines the interface for applying and removing backup exclusions.
// Each backup system (Time Machine, Carbon Copy Cloner) has its own adapter
// implementing this interface, making them interchangeable (Liskov substitution).
type BackupProvider interface {
	// Name returns the human-readable name of the backup system.
	Name() string

	// IsAvailable checks whether the backup system is installed and accessible.
	IsAvailable(ctx context.Context) (bool, error)

	// Exclude adds a path to the backup exclusion list.
	Exclude(ctx context.Context, path string) error

	// Remove removes a path from the backup exclusion list.
	Remove(ctx context.Context, path string) error

	// IsExcluded checks whether a path is currently excluded.
	IsExcluded(ctx context.Context, path string) (bool, error)

	// ListExcluded returns all currently excluded paths.
	ListExcluded(ctx context.Context) ([]string, error)
}
