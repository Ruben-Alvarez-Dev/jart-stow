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


