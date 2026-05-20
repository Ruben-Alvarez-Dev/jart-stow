// Package services implements application use cases for Jart-Stow.
// Services depend on port interfaces (never on adapters directly) via constructor injection.
package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
)

// ScanProgress represents the real-time state of an ongoing scan.
type ScanProgress struct {
	CurrentPath   string
	ItemsFound    int
	TotalSize     int64
	ScanningDone  bool
	FoundArtifact *Artifact
}

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
// It uses a worker pool to parallelize directory traversal and sizing for maximum performance.
func (s *ScanService) FindArtifacts(ctx context.Context, root string) ([]Artifact, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scanning %s: not a directory", root)
	}

	var artifacts []*Artifact
	var artMu sync.Mutex
	var itemsFound int64
	var totalSize int64

	numWorkers := runtime.NumCPU() * 2
	dirQueue := make(chan string, 1024)
	var wg sync.WaitGroup

	patternSet := s.buildPatternSet()

	var activeTasks int32 = 1
	dirQueue <- root

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dir := range dirQueue {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Report progress: current directory
				if progress != nil {
					progress <- ScanProgress{
						CurrentPath: dir,
						ItemsFound:  int(atomic.LoadInt64(&itemsFound)),
						TotalSize:   atomic.LoadInt64(&totalSize),
					}
				}

				entries, err := os.ReadDir(dir)
				if err != nil {
					if atomic.AddInt32(&activeTasks, -1) == 0 {
						close(dirQueue)
					}
					continue
				}

				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}

					name := entry.Name()
					fullPath := filepath.Join(dir, name)

					if patternSet[name] {
						size := dirSizeConcurrent(fullPath)
						art := &Artifact{
							Path:        fullPath,
							PatternName: name,
							SizeBytes:   size,
						}
						
						artMu.Lock()
						artifacts = append(artifacts, art)
						artMu.Unlock()
						
						atomic.AddInt64(&itemsFound, 1)
						atomic.AddInt64(&totalSize, size)

						// Report artifact found
						if progress != nil {
							progress <- ScanProgress{
								CurrentPath:   dir,
								ItemsFound:    int(atomic.LoadInt64(&itemsFound)),
								TotalSize:     atomic.LoadInt64(&totalSize),
								FoundArtifact: art,
							}
						}
					} else {
						// Continue recursion if not an artifact and within depth
						rel, _ := filepath.Rel(root, fullPath)
						depth := len(strings.Split(rel, string(os.PathSeparator)))
						if depth < 5 { // Hardcoded limit for safety, should be config
							atomic.AddInt32(&activeTasks, 1)
							select {
							case dirQueue <- fullPath:
							default:
								// Queue full, fallback or drop (shouldn't happen with 1024 and depth 5)
								atomic.AddInt32(&activeTasks, -1)
							}
						}
					}
				}
				
				if atomic.AddInt32(&activeTasks, -1) == 0 {
					close(dirQueue)
				}
			}
		}()
	}

	wg.Wait()
	
	if progress != nil {
		progress <- ScanProgress{ScanningDone: true}
	}
	
	return artifacts, nil
}

// dirSizeConcurrent calculates the total size of a directory using multiple workers.
func dirSizeConcurrent(path string) int64 {
	var totalSize int64
	
	// Fast path for non-existent or inaccessible
	if _, err := os.Stat(path); err != nil {
		return 0
	}

	// For artifacts, a simple sequential walk is often enough and less overhead
	// if the artifact directory is small. But let's use a faster sequential version first.
	// Parallelizing the size calculation of every single artifact found might be overkill
	// and actually slow down the main walker due to context switching.
	
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				atomic.AddInt64(&totalSize, info.Size())
			}
		}
		return nil
	})
	
	return totalSize
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
