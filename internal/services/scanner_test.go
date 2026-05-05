package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScanService_Defaults(t *testing.T) {
	s := NewScanService(0, nil)
	assert.Equal(t, 3, s.maxDepth)
	assert.Equal(t, DefaultArtifactPatterns, s.patterns)
}

func TestNewScanService_Custom(t *testing.T) {
	patterns := []string{"custom_pattern"}
	s := NewScanService(5, patterns)
	assert.Equal(t, 5, s.maxDepth)
	assert.Equal(t, patterns, s.patterns)
}

func TestFindArtifacts_Basic(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Create project structure
	// /project/
	//   node_modules/     <- artifact
	//   src/
	//     .venv/          <- artifact (max depth 1 from root)
	//     app.js
	createDir(t, filepath.Join(dir, "node_modules"))
	createDir(t, filepath.Join(dir, "src", ".venv"))
	createFile(t, filepath.Join(dir, "src", "app.js"), "content")

	s := NewScanService(3, nil)
	artifacts, err := s.FindArtifacts(ctx, dir)
	require.NoError(t, err)

	assert.Len(t, artifacts, 2)

	// Verify patterns detected
	patterns := map[string]string{}
	for _, a := range artifacts {
		patterns[a.PatternName] = a.Path
	}
	assert.Contains(t, patterns, "node_modules")
	assert.Contains(t, patterns, ".venv")
}

func TestFindArtifacts_RespectsMaxDepth(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Deep structure: project/a/b/node_modules
	createDir(t, filepath.Join(dir, "a", "b", "node_modules"))

	// maxDepth=1: should NOT find node_modules
	s := NewScanService(1, nil)
	artifacts, err := s.FindArtifacts(ctx, dir)
	require.NoError(t, err)
	assert.Empty(t, artifacts, "maxDepth=1 should not find deep artifacts")

	// maxDepth=2: should NOT find node_modules (it's at depth 2)
	s2 := NewScanService(2, nil)
	artifacts, err = s2.FindArtifacts(ctx, dir)
	require.NoError(t, err)
	assert.Empty(t, artifacts, "maxDepth=2 should not find artifacts at depth 2")

	// maxDepth=3: should find it
	s3 := NewScanService(3, nil)
	artifacts, err = s3.FindArtifacts(ctx, dir)
	require.NoError(t, err)
	assert.Len(t, artifacts, 1)
	assert.Equal(t, "node_modules", artifacts[0].PatternName)
}

func TestFindArtifacts_DoesNotRecurseIntoArtifacts(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// node_modules/
	//   .cache/        <- should NOT be detected (parent is artifact)
	//   dist/          <- should NOT be detected
	createDir(t, filepath.Join(dir, "node_modules", ".cache"))
	createDir(t, filepath.Join(dir, "node_modules", "dist"))

	s := NewScanService(3, nil)
	artifacts, err := s.FindArtifacts(ctx, dir)
	require.NoError(t, err)

	assert.Len(t, artifacts, 1)
	assert.Equal(t, "node_modules", artifacts[0].PatternName)
}

func TestFindArtifacts_SizeCalculation(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// node_modules/
	//   package.json (50 bytes)
	//   index.js (30 bytes)
	createDir(t, filepath.Join(dir, "node_modules"))
	createFile(t, filepath.Join(dir, "node_modules", "package.json"), string(make([]byte, 50)))
	createFile(t, filepath.Join(dir, "node_modules", "index.js"), string(make([]byte, 30)))

	s := NewScanService(3, nil)
	artifacts, err := s.FindArtifacts(ctx, dir)
	require.NoError(t, err)

	assert.Len(t, artifacts, 1)
	assert.Equal(t, int64(80), artifacts[0].SizeBytes)
	assert.Equal(t, "node_modules", artifacts[0].PatternName)
	assert.Contains(t, artifacts[0].Path, "node_modules")
}

func TestFindArtifacts_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	// Create many directories to make the walk take time
	for i := 0; i < 100; i++ {
		createDir(t, filepath.Join(dir, "dir_"+string(rune('a'+i%26))))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	s := NewScanService(3, nil)
	_, err := s.FindArtifacts(ctx, dir)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFindArtifacts_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	createFile(t, file, "content")

	s := NewScanService(3, nil)
	_, err := s.FindArtifacts(context.Background(), file)
	assert.ErrorContains(t, err, "not a directory")
}

func TestFindArtifacts_NonexistentPath(t *testing.T) {
	s := NewScanService(3, nil)
	_, err := s.FindArtifacts(context.Background(), "/nonexistent/path/12345")
	assert.ErrorContains(t, err, "scanning")
}

func TestFindArtifacts_CustomPatterns(t *testing.T) {
	dir := t.TempDir()
	createDir(t, filepath.Join(dir, "my_custom_cache"))
	createDir(t, filepath.Join(dir, "node_modules"))

	s := NewScanService(3, []string{"my_custom_cache"})
	artifacts, err := s.FindArtifacts(context.Background(), dir)
	require.NoError(t, err)

	assert.Len(t, artifacts, 1)
	assert.Equal(t, "my_custom_cache", artifacts[0].PatternName)
}

func TestFindArtifacts_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	s := NewScanService(3, nil)
	artifacts, err := s.FindArtifacts(context.Background(), dir)
	require.NoError(t, err)
	assert.Empty(t, artifacts)
}

// --- helpers ---

func createDir(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(path, 0o755)
	require.NoError(t, err)
}

func createFile(t *testing.T, path string, content string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}
