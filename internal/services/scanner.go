// Package services implements application use cases for Jart-Stow.
// Services depend on port interfaces (never on adapters directly) via constructor injection.
package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultArtifactPatterns are the directory names that the scanner looks for.
// These cover the most common development artifact directories across languages.
var DefaultArtifactPatterns = []string{
	"node_modules",  // JavaScript / Node.js
	".venv",         // Python virtual environment
	"venv",          // Python virtual environment (alt)
	"__pycache__",   // Python bytecode cache
	".pytest_cache", // Python test cache
	"target",        // Rust build output
	"vendor",        // Go vendor directory
	"build",         // Generic build output
	"dist",          // Generic distribution output
	".next",         // Next.js build
	".nuxt",         // Nuxt.js build
	".cache",        // Generic cache
	".turbo",        // Turborepo cache
	".eslintcache",  // ESLint cache
	"coverage",      // Test coverage reports
}

// Artifact represents a development artifact found during scanning.
type Artifact struct {
	Path        string
	PatternName string
	SizeBytes   int64
}

// ScanService scans project directories for development artifacts (node_modules, .venv, etc.).
type ScanService struct {
	maxDepth int
	patterns []string
}

// NewScanService creates a new ScanService. If patterns is nil, DefaultArtifactPatterns are used.
func NewScanService(maxDepth int, patterns []string) *ScanService {
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if len(patterns) == 0 {
		patterns = DefaultArtifactPatterns
	}
	return &ScanService{
		maxDepth: maxDepth,
		patterns: patterns,
	}
}

// FindArtifacts scans the given path recursively for directories matching the configured patterns.
// Only directories with names matching a pattern are returned. Scanning respects maxDepth from root.
func (s *ScanService) FindArtifacts(ctx context.Context, root string) ([]Artifact, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scanning %s: not a directory", root)
	}

	patternSet := s.buildPatternSet()
	var artifacts []Artifact

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip paths we can't access
			return filepath.SkipDir
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !info.IsDir() {
			return nil
		}

		// Check if this directory name matches a pattern
		name := info.Name()
		if patternSet[name] {
			artifact := Artifact{
				Path:        path,
				PatternName: name,
				SizeBytes:   dirSize(path),
			}
			artifacts = append(artifacts, artifact)
			// Don't recurse into artifact directories
			return filepath.SkipDir
		}

		// Check depth
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			rel = ""
		}
		depth := len(strings.Split(rel, string(filepath.Separator)))
		if depth >= s.maxDepth {
			return filepath.SkipDir
		}

		return nil
	}

	if err := filepath.Walk(root, walkFn); err != nil {
		if err == ctx.Err() {
			return artifacts, err
		}
		return artifacts, fmt.Errorf("walking %s: %w", root, err)
	}

	return artifacts, nil
}

// buildPatternSet creates a set from the configured patterns for O(1) lookup.
func (s *ScanService) buildPatternSet() map[string]bool {
	set := make(map[string]bool, len(s.patterns))
	for _, p := range s.patterns {
		set[p] = true
	}
	return set
}

// dirSize calculates the total size of all files in a directory recursively.
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
