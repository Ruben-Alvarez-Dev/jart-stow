package ports

import "context"

// FileSystemEvent represents a change detected by the file system watcher.
type FileSystemEvent struct {
	Path string
	Op   FileSystemOp
}

// FileSystemOp represents the type of file system operation detected.
type FileSystemOp int

const (
	OpCreate FileSystemOp = iota
	OpRemove
	OpRename
	OpModify
)

// String returns a human-readable representation of the operation.
func (op FileSystemOp) String() string {
	switch op {
	case OpCreate:
		return "create"
	case OpRemove:
		return "remove"
	case OpRename:
		return "rename"
	case OpModify:
		return "modify"
	default:
		return "unknown"
	}
}

// FileSystemWatcher defines the interface for monitoring directories for changes.
// The macOS implementation uses FSEvents via the fsnotify package.
type FileSystemWatcher interface {
	// Watch starts monitoring the given directory recursively.
	Watch(ctx context.Context, path string) error

	// Unwatch stops monitoring the given directory.
	Unwatch(path string) error

	// Events returns a channel that receives file system events.
	Events() <-chan FileSystemEvent

	// Errors returns a channel that receives watcher errors.
	Errors() <-chan error

	// Close stops all watching and releases resources.
	Close() error

	// AddIgnore adds a path pattern to ignore during watching.
	AddIgnore(pattern string) error
}
