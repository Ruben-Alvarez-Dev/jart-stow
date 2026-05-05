// Package filesystem implements JunkScanners for filesystem-based junk (temp files, caches).
package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
)

// TempScanner discovers stale temporary files in /tmp, /var/tmp, and user tmp dirs.
type TempScanner struct{}

// NewTempScanner creates a new temp files scanner.
func NewTempScanner() *TempScanner {
	return &TempScanner{}
}

// Name returns the scanner identifier (matches junk_categories.scanner = "filesystem").
func (s *TempScanner) Name() string {
	return "filesystem"
}

// Scan discovers temporary files older than 7 days.
func (s *TempScanner) Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error) {
	if category.Name != "tmp_files" {
		return nil, nil
	}

	var items []domain.JunkItem
	now := time.Now()
	cutoff := now.Add(-7 * 24 * time.Hour)

	paths := []string{"/tmp", "/var/tmp"}
	home, _ := os.UserHomeDir()
	if home != "" {
		paths = append(paths, filepath.Join(home, "tmp"))
	}

	for _, basePath := range paths {
		info, err := os.Stat(basePath)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(basePath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return items, ctx.Err()
			default:
			}

			fullPath := filepath.Join(basePath, entry.Name())
			fileInfo, err := entry.Info()
			if err != nil {
				continue
			}

			if fileInfo.ModTime().After(cutoff) {
				continue
			}

			size := fileInfo.Size()
			if fileInfo.IsDir() {
				size = dirSize(fullPath)
			}

			modTime := fileInfo.ModTime()
			items = append(items, domain.JunkItem{
				CategoryID:     category.ID,
				Path:           fullPath,
				Description:    fmt.Sprintf("Stale temp file/dir: %s (%s old)", entry.Name(),
					time.Since(modTime).Truncate(time.Hour).String()),
				SizeBytes:      size,
				LastAccessed:   &modTime,
				VerifiedByUser: domain.VerificationPending,
				CreatedAt:      now,
			})
		}
	}
	return items, nil
}

// CacheScanner discovers system and user cache directories for review.
type CacheScanner struct{}

// NewCacheScanner creates a new cache scanner.
func NewCacheScanner() *CacheScanner {
	return &CacheScanner{}
}

// Name returns the scanner identifier.
func (s *CacheScanner) Name() string {
	return "cache"
}

// Scan discovers cache directories and their sizes.
func (s *CacheScanner) Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error) {
	var cacheDirs []string
	now := time.Now()

	switch category.Name {
	case "system_caches":
		cacheDirs = []string{"/Library/Caches"}
	case "user_caches":
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("finding home directory: %w", err)
		}
		cacheDirs = []string{filepath.Join(home, "Library/Caches")}
	default:
		return nil, nil
	}

	var items []domain.JunkItem
	for _, basePath := range cacheDirs {
		info, err := os.Stat(basePath)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(basePath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return items, ctx.Err()
			default:
			}

			if !entry.IsDir() {
				continue
			}

			fullPath := filepath.Join(basePath, entry.Name())
			size := dirSize(fullPath)
			fileInfo, _ := entry.Info()
			var modTime *time.Time
			if fileInfo != nil {
				t := fileInfo.ModTime()
				modTime = &t
			}

			items = append(items, domain.JunkItem{
				CategoryID:     category.ID,
				Path:           fullPath,
				Description:    fmt.Sprintf("Cache directory: %s (%s)", entry.Name(),
					formatBytes(size)),
				SizeBytes:      size,
				LastAccessed:   modTime,
				VerifiedByUser: domain.VerificationPending,
				CreatedAt:      now,
			})
		}
	}
	return items, nil
}

// XcodeScanner discovers Xcode derived data and archives.
type XcodeScanner struct{}

// NewXcodeScanner creates a new Xcode junk scanner.
func NewXcodeScanner() *XcodeScanner {
	return &XcodeScanner{}
}

// Name returns the scanner identifier.
func (s *XcodeScanner) Name() string {
	return "xcode"
}

// Scan discovers Xcode DerivedData directories.
func (s *XcodeScanner) Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error) {
	if category.Name != "xcode_derived_data" {
		return nil, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}

	derivedDataPath := filepath.Join(home, "Library/Developer/Xcode/DerivedData")
	info, err := os.Stat(derivedDataPath)
	if err != nil {
		return nil, nil
	}
	if !info.IsDir() {
		return nil, nil
	}

	var items []domain.JunkItem
	now := time.Now()

	entries, err := os.ReadDir(derivedDataPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return items, ctx.Err()
		default:
		}

		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(derivedDataPath, entry.Name())
		size := dirSize(fullPath)
		fileInfo, _ := entry.Info()
		var modTime *time.Time
		if fileInfo != nil {
			t := fileInfo.ModTime()
			modTime = &t
		}

		items = append(items, domain.JunkItem{
			CategoryID:     category.ID,
			Path:           fullPath,
			Description:    fmt.Sprintf("Xcode DerivedData: %s (%s)", entry.Name(),
				formatBytes(size)),
			SizeBytes:      size,
			LastAccessed:   modTime,
			VerifiedByUser: domain.VerificationPending,
			CreatedAt:      now,
		})
	}
	return items, nil
}

// BrewScanner discovers Homebrew cache files.
type BrewScanner struct{}

// NewBrewScanner creates a new Homebrew cache scanner.
func NewBrewScanner() *BrewScanner {
	return &BrewScanner{}
}

// Name returns the scanner identifier.
func (s *BrewScanner) Name() string {
	return "brew"
}

// Scan discovers the Homebrew cache directory.
func (s *BrewScanner) Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error) {
	if category.Name != "brew_cache" {
		return nil, nil
	}

	now := time.Now()

	// Try common Homebrew cache locations
	cachePaths := []string{
		"/opt/homebrew/Library/Caches",
		"/usr/local/Homebrew/Library/Caches",
		"/home/linuxbrew/.linuxbrew/Library/Caches",
	}

	for _, cachePath := range cachePaths {
		info, err := os.Stat(cachePath)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			continue
		}

		size := dirSize(cachePath)

		return []domain.JunkItem{{
			CategoryID:     category.ID,
			Path:           cachePath,
			Description:    fmt.Sprintf("Homebrew cache: %s", formatBytes(size)),
			SizeBytes:      size,
			VerifiedByUser: domain.VerificationPending,
			CreatedAt:      now,
		}}, nil
	}

	return nil, nil
}

// dirSize calculates the total size of all files in a directory.
func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

// formatBytes returns a human-readable size string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
