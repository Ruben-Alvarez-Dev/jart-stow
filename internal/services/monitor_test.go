package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/stretchr/testify/assert"
)

func TestDefaultMonitorConfig(t *testing.T) {
	cfg := DefaultMonitorConfig()
	assert.Equal(t, 2*time.Second, cfg.DebounceDelay)
	assert.Equal(t, 24*time.Hour, cfg.JunkInterval)
}

func TestNewMonitorService_Smoke(t *testing.T) {
	// Verify the constructor does not panic with nil dependencies.
	monitor := NewMonitorService(
		nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil,
		DefaultMonitorConfig(),
	)

	assert.NotNil(t, monitor)
	assert.Equal(t, 2*time.Second, monitor.debounceDelay)
	assert.Equal(t, 24*time.Hour, monitor.junkInterval)
}

// stubWatcher implements ports.FileSystemWatcher for testing.
type stubWatcher struct {
	events chan ports.FileSystemEvent
	errors chan error
	closed bool
}

func newStubWatcher() *stubWatcher {
	return &stubWatcher{
		events: make(chan ports.FileSystemEvent, 1),
		errors: make(chan error, 1),
	}
}

func (w *stubWatcher) Watch(_ context.Context, _ string) error { return nil }
func (w *stubWatcher) Unwatch(_ string) error                  { return nil }
func (w *stubWatcher) Events() <-chan ports.FileSystemEvent    { return w.events }
func (w *stubWatcher) Errors() <-chan error                    { return w.errors }
func (w *stubWatcher) AddIgnore(_ string) error                { return nil }
func (w *stubWatcher) Start()                                  {}
func (w *stubWatcher) Close() error {
	w.closed = true
	return nil
}

// stubWatchRootRepo implements ports.WatchRootRepository for testing.
type stubWatchRootRepo struct {
	roots []domain.WatchRoot
}

func (r *stubWatchRootRepo) FindAll(_ context.Context) ([]domain.WatchRoot, error) {
	return r.roots, nil
}
func (r *stubWatchRootRepo) FindByPath(_ context.Context, _ string) (*domain.WatchRoot, error) {
	return nil, domain.ErrWatchRootNotFound
}
func (r *stubWatchRootRepo) Save(_ context.Context, _ *domain.WatchRoot) (*domain.WatchRoot, error) {
	return nil, errors.New("not implemented")
}
func (r *stubWatchRootRepo) Delete(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

func TestShutdown_ClosesWatcher(t *testing.T) {
	watcher := newStubWatcher()

	monitor := &MonitorService{
		watcher: watcher,
	}

	err := monitor.shutdown()
	assert.NoError(t, err)
	assert.True(t, watcher.closed, "watcher should be closed after shutdown")
}

func TestShutdown_Error(t *testing.T) {
	monitor := &MonitorService{
		watcher: &failingWatcher{},
	}

	err := monitor.shutdown()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closing watcher")
}

type failingWatcher struct{ stubWatcher }

func (w *failingWatcher) Close() error { return errors.New("close failed") }
