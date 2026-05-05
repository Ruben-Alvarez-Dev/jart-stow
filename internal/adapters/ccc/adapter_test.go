package ccc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	a := NewAdapter("")
	assert.Equal(t, "carbon_copy_cloner", a.Name())
	assert.Equal(t, DefaultExclusionsPath, a.exclusionsPath)
}

func TestNewAdapter_CustomPath(t *testing.T) {
	a := NewAdapter("/custom/path.txt")
	assert.Equal(t, "/custom/path.txt", a.exclusionsPath)
}

func TestIsAvailable_NotExists(t *testing.T) {
	a := NewAdapter("/tmp/jart-stow-ccc-test-nonexistent.txt")
	available, err := a.IsAvailable(context.Background())
	require.NoError(t, err)
	assert.False(t, available)
}

func TestFullLifecycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Exclusions.txt")
	a := NewAdapter(path)
	ctx := context.Background()

	// Initially not available
	available, err := a.IsAvailable(ctx)
	require.NoError(t, err)
	assert.False(t, available)

	// Exclude - should create file
	err = a.Exclude(ctx, "/Users/test/Code/project/node_modules")
	require.NoError(t, err)

	// Now available
	available, err = a.IsAvailable(ctx)
	require.NoError(t, err)
	assert.True(t, available)

	// Add more exclusions
	err = a.Exclude(ctx, "/Users/test/Code/project/.venv")
	require.NoError(t, err)
	err = a.Exclude(ctx, "/Users/test/Code/project/build")
	require.NoError(t, err)

	// Check excluded
	excluded, err := a.IsExcluded(ctx, "/Users/test/Code/project/node_modules")
	require.NoError(t, err)
	assert.True(t, excluded)

	excluded, err = a.IsExcluded(ctx, "/Users/test/Code/project/not_excluded")
	require.NoError(t, err)
	assert.False(t, excluded)

	// List excluded
	paths, err := a.ListExcluded(ctx)
	require.NoError(t, err)
	assert.Len(t, paths, 3)

	// Remove one
	err = a.Remove(ctx, "/Users/test/Code/project/.venv")
	require.NoError(t, err)

	paths, err = a.ListExcluded(ctx)
	require.NoError(t, err)
	assert.Len(t, paths, 2)

	// Verify removal
	excluded, err = a.IsExcluded(ctx, "/Users/test/Code/project/.venv")
	require.NoError(t, err)
	assert.False(t, excluded)

	// File content verification
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "node_modules")
	assert.NotContains(t, content, ".venv")
	assert.Contains(t, content, "build")
}

func TestDuplicateExclusion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Exclusions.txt")
	a := NewAdapter(path)
	ctx := context.Background()

	err := a.Exclude(ctx, "/test/path")
	require.NoError(t, err)

	// Duplicate should be idempotent
	err = a.Exclude(ctx, "/test/path")
	require.NoError(t, err)

	paths, err := a.ListExcluded(ctx)
	require.NoError(t, err)
	assert.Len(t, paths, 1)
}

func TestRemove_NonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Exclusions.txt")
	a := NewAdapter(path)
	ctx := context.Background()

	// Remove non-existent should be a no-op
	err := a.Remove(ctx, "/nonexistent/path")
	require.NoError(t, err)
}

func TestListExcluded_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Exclusions.txt")
	a := NewAdapter(path)
	ctx := context.Background()

	// Ensure file exists but is empty
	err := a.Exclude(ctx, "/tmp/path")
	require.NoError(t, err)
	err = a.Remove(ctx, "/tmp/path")
	require.NoError(t, err)

	paths, err := a.ListExcluded(ctx)
	require.NoError(t, err)
	assert.Empty(t, paths)
}

func TestListExcluded_IgnoresComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Exclusions.txt")
	a := NewAdapter(path)
	ctx := context.Background()

	// Write file with comments
	f, err := os.Create(path)
	require.NoError(t, err)
	f.WriteString("# This is a comment\n")
	f.WriteString("/valid/path\n")
	f.WriteString("  # indented comment\n")
	f.Close()

	paths, err := a.ListExcluded(ctx)
	require.NoError(t, err)
	assert.Len(t, paths, 1)
	assert.Equal(t, "/valid/path", paths[0])
}
