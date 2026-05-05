// Package fsevents implements the FileSystemWatcher port using macOS FSEvents via fsnotify.
package fsevents

import (
	"context"
	"fmt"
	"sync"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/fsnotify/fsnotify"
)

// Watcher wraps fsnotify to implement the ports.FileSystemWatcher interface.
type Watcher struct {
	w       *fsnotify.Watcher
	events  chan ports.FileSystemEvent
	errors  chan error
	ignored map[string]struct{}
	mu      sync.RWMutex
	done    chan struct{}
}

// NewWatcher creates a new FSEvents-backed file system watcher.
func NewWatcher() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	return &Watcher{
		w:       w,
		events:  make(chan ports.FileSystemEvent, 256),
		errors:  make(chan error, 16),
		ignored: make(map[string]struct{}),
		done:    make(chan struct{}),
	}, nil
}

// Watch starts monitoring the given directory recursively.
func (fw *Watcher) Watch(_ context.Context, path string) error {
	return fw.w.Add(path)
}

// Unwatch stops monitoring the given directory.
func (fw *Watcher) Unwatch(path string) error {
	return fw.w.Remove(path)
}

// Events returns a channel that receives file system events.
// Callers must call Start() before receiving events.
func (fw *Watcher) Events() <-chan ports.FileSystemEvent {
	return fw.events
}

// Errors returns a channel that receives watcher errors.
func (fw *Watcher) Errors() <-chan error {
	return fw.errors
}

// Close stops all watching and releases resources.
func (fw *Watcher) Close() error {
	close(fw.done)
	return fw.w.Close()
}

// AddIgnore adds a path pattern to ignore during watching.
func (fw *Watcher) AddIgnore(pattern string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.ignored[pattern] = struct{}{}
	return nil
}

// Start begins forwarding fsnotify events to the Events channel.
// It runs in a goroutine until Close() is called.
func (fw *Watcher) Start() {
	go func() {
		defer close(fw.events)
		defer close(fw.errors)

		for {
			select {
			case <-fw.done:
				return
			case event, ok := <-fw.w.Events:
				if !ok {
					return
				}
				fw.mu.RLock()
				_, ignored := fw.ignored[event.Name]
				fw.mu.RUnlock()
				if ignored {
					continue
				}
				fw.events <- ports.FileSystemEvent{
					Path: event.Name,
					Op:   mapOp(event.Op),
				}
			case err, ok := <-fw.w.Errors:
				if !ok {
					return
				}
				select {
				case fw.errors <- err:
				case <-fw.done:
					return
				}
			}
		}
	}()
}

// mapOp converts fsnotify operations to our FileSystemOp type.
func mapOp(op fsnotify.Op) ports.FileSystemOp {
	switch {
	case op&fsnotify.Create != 0:
		return ports.OpCreate
	case op&fsnotify.Remove != 0:
		return ports.OpRemove
	case op&fsnotify.Rename != 0:
		return ports.OpRename
	case op&fsnotify.Write != 0:
		return ports.OpModify
	default:
		return 0
	}
}
