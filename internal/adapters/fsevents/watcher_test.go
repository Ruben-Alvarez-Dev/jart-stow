package fsevents

import (
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestMapOp(t *testing.T) {
	tests := []struct {
		name     string
		op       fsnotify.Op
		expected ports.FileSystemOp
	}{
		{"create", fsnotify.Create, ports.OpCreate},
		{"remove", fsnotify.Remove, ports.OpRemove},
		{"rename", fsnotify.Rename, ports.OpRename},
		{"write", fsnotify.Write, ports.OpModify},
		{"unknown", fsnotify.Op(0x100), ports.FileSystemOp(0)},
		{"multi_create_write", fsnotify.Create | fsnotify.Write, ports.OpCreate},
		{"multi_remove_rename", fsnotify.Remove | fsnotify.Rename, ports.OpRemove},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapOp(tt.op)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewWatcher(t *testing.T) {
	w, err := NewWatcher()
	assert.NoError(t, err)
	assert.NotNil(t, w)
	assert.NotNil(t, w.events)
	assert.NotNil(t, w.errors)
	assert.NotNil(t, w.done)
	w.Close()
}

func TestAddIgnore(t *testing.T) {
	w, err := NewWatcher()
	assert.NoError(t, err)
	defer w.Close()

	err = w.AddIgnore("*.log")
	assert.NoError(t, err)

	w.mu.RLock()
	_, ok := w.ignored["*.log"]
	w.mu.RUnlock()
	assert.True(t, ok)
}

func TestAddIgnoreMultiple(t *testing.T) {
	w, err := NewWatcher()
	assert.NoError(t, err)
	defer w.Close()

	patterns := []string{"*.log", "node_modules", ".git"}
	for _, p := range patterns {
		assert.NoError(t, w.AddIgnore(p))
	}

	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, p := range patterns {
		_, ok := w.ignored[p]
		assert.True(t, ok, "pattern %q should be ignored", p)
	}
}
