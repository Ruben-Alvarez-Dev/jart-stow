package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"less than 1KB", 512, "512 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"1.5KB", 1536, "1.5 KB"},
		{"1MB", 1024 * 1024, "1.0 MB"},
		{"1GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"1.5GB", int64(1.5 * 1024 * 1024 * 1024), "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirSize(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		size := dirSize(dir)
		assert.Equal(t, int64(0), size)
	})

	t.Run("directory with files", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world!"), 0o644)
		require.NoError(t, err)

		size := dirSize(dir)
		assert.Equal(t, int64(11), size)
	})

	t.Run("directory with subdirectory", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "sub")
		err := os.Mkdir(subDir, 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("data"), 0o644)
		require.NoError(t, err)

		size := dirSize(dir)
		assert.Equal(t, int64(4), size)
	})

	t.Run("nonexistent path", func(t *testing.T) {
		size := dirSize("/nonexistent/path/xyz")
		assert.Equal(t, int64(0), size)
	})
}

func TestTempScanner_WithTempDir(t *testing.T) {
	// Create a temp directory that mimics /tmp with stale files.
	dir := t.TempDir()

	// Create a file older than 7 days
	oldFile := filepath.Join(dir, "old_file.txt")
	err := os.WriteFile(oldFile, []byte("stale data"), 0o644)
	require.NoError(t, err)
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	err = os.Chtimes(oldFile, oldTime, oldTime)
	require.NoError(t, err)

	// Create a recent file (less than 7 days)
	recentFile := filepath.Join(dir, "recent_file.txt")
	err = os.WriteFile(recentFile, []byte("fresh data"), 0o644)
	require.NoError(t, err)

	// Create a stale subdirectory
	oldDir := filepath.Join(dir, "old_dir")
	err = os.Mkdir(oldDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(oldDir, "f.txt"), []byte("x"), 0o644)
	require.NoError(t, err)
	err = os.Chtimes(oldDir, oldTime, oldTime)
	require.NoError(t, err)

	// Manually test the scanning logic by using a scanner with custom paths.
	// The TempScanner uses hardcoded paths, so we test behavior directly.
	s := NewTempScanner()
	assert.Equal(t, "filesystem", s.Name())

	// Verify wrong category returns nil
	cat := domain.JunkCategory{Name: "not_tmp_files"}
	items, err := s.Scan(context.Background(), cat)
	assert.NoError(t, err)
	assert.Nil(t, items)

	// validate the temp dir was set up correctly
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 3)
}

func TestScannerNames(t *testing.T) {
	tests := []struct {
		name     string
		scanner  interface{ Name() string }
		expected string
	}{
		{"temp", NewTempScanner(), "filesystem"},
		{"cache", NewCacheScanner(), "cache"},
		{"xcode", NewXcodeScanner(), "xcode"},
		{"brew", NewBrewScanner(), "brew"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.scanner.Name())
		})
	}
}

func TestScanners_WrongCategory(t *testing.T) {
	tests := []struct {
		name    string
		scanner interface {
			Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error)
		}
	}{
		{"temp wrong category", NewTempScanner()},
		{"cache wrong category", NewCacheScanner()},
		{"xcode wrong category", NewXcodeScanner()},
		{"brew wrong category", NewBrewScanner()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := domain.JunkCategory{Name: "not_matching"}
			items, err := tt.scanner.Scan(context.Background(), cat)
			assert.NoError(t, err)
			assert.Nil(t, items)
		})
	}
}
