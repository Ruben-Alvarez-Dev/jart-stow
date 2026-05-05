package tmutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter("")
	assert.Equal(t, "time_machine", a.Name())
	assert.Equal(t, "tmutil", a.tmutilPath)
}

func TestNewAdapter_CustomPath(t *testing.T) {
	a := NewAdapter("/usr/bin/tmutil")
	assert.Equal(t, "/usr/bin/tmutil", a.tmutilPath)
}

func TestIsAvailable(t *testing.T) {
	a := NewAdapter("")
	available, err := a.IsAvailable(context.Background())
	require.NoError(t, err)
	assert.True(t, available)
}

func TestIsAvailable_NotFound(t *testing.T) {
	a := NewAdapter("/nonexistent/tmutil")
	available, err := a.IsAvailable(context.Background())
	require.NoError(t, err)
	assert.False(t, available)
}

func TestIsExcluded_NonExistentPath(t *testing.T) {
	a := NewAdapter("")
	excluded, err := a.IsExcluded(context.Background(), "/tmp/jart-stow-test-nonexistent-12345")
	require.NoError(t, err)
	assert.False(t, excluded)
}

func TestName(t *testing.T) {
	a := NewAdapter("")
	assert.Equal(t, "time_machine", a.Name())
}

func TestExclude_NonExistentPath(t *testing.T) {
	a := NewAdapter("")
	// tmutil requires the path to exist to add exclusion
	err := a.Exclude(context.Background(), "/tmp/jart-stow-nonexistent-7890123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tmutil addexclusion")
}

func TestRemove_NonExistentPath(t *testing.T) {
	a := NewAdapter("")
	err := a.Remove(context.Background(), "/tmp/jart-stow-nonexistent-4567890")
	// removeexclusion on a non-excluded path returns an error from tmutil
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tmutil removeexclusion")
}

func TestListExcluded(t *testing.T) {
	a := NewAdapter("")
	paths, err := a.ListExcluded(context.Background())
	// If mdfind is available, this returns paths; if not, returns an error
	// Either way, if it succeeds, paths is a slice (possibly empty)
	if err == nil {
		assert.NotNil(t, paths)
	}
}

func TestRemove_NotExcluded(t *testing.T) {
	// Create a temp dir, check it's not excluded, try to remove (should fail)
	dir := t.TempDir()
	a := NewAdapter("")

	// Verify not excluded
	excluded, err := a.IsExcluded(context.Background(), dir)
	require.NoError(t, err)

	if excluded {
		t.Skip("temp dir is unexpectedly excluded")
	}

	// Try to remove — should fail since it's not excluded
	err = a.Remove(context.Background(), dir)
	assert.Error(t, err)
}
