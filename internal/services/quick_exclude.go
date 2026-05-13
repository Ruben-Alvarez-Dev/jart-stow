// Package services — QuickExcludeService provides a simple scan-and-exclude flow
// without needing a project in the database. This replicates the workflow from
// EXCLUSION-SCRIPT but using jart-stow's adapters and architecture.
package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// QuickExcludeService provides a simple scan-and-exclude workflow for development
// dependency folders (node_modules, .venv, target/, etc.) across Time Machine and CCC.
//
// Unlike ExcludeService, this does not require projects to exist in the database.
// It's designed for the direct "scan volume / folder → apply exclusions" flow.
type QuickExcludeService struct {
	scanService *ScanService
	backups     []ports.BackupProvider
}

// NewQuickExcludeService creates a new QuickExcludeService.
func NewQuickExcludeService(scanService *ScanService, backups ...ports.BackupProvider) *QuickExcludeService {
	return &QuickExcludeService{
		scanService: scanService,
		backups:     backups,
	}
}

// QuickScanResult holds the result of a quick scan.
type QuickScanResult struct {
	Path        string
	PatternName string
	SizeBytes   int64
	AlreadyDone bool // true if already excluded from all available backup systems
}

// Scan scans a path for development artifact directories.
// Depth and patterns are configured in the underlying ScanService.
func (s *QuickExcludeService) Scan(ctx context.Context, root string) ([]QuickScanResult, error) {
	artifacts, err := s.scanService.FindArtifacts(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", root, err)
	}

	results := make([]QuickScanResult, len(artifacts))
	for i, a := range artifacts {
		alreadyDone := false
		if len(s.backups) > 0 {
			allExcluded := true
			for _, b := range s.backups {
				excluded, err := b.IsExcluded(ctx, a.Path)
				if err != nil || !excluded {
					allExcluded = false
					break
				}
			}
			alreadyDone = allExcluded
		}

		results[i] = QuickScanResult{
			Path:        a.Path,
			PatternName: a.PatternName,
			SizeBytes:   a.SizeBytes,
			AlreadyDone: alreadyDone,
		}
	}

	return results, nil
}

// ExcludePaths applies exclusions to all configured backup systems for the given paths.
// Returns a map of path → error for paths that failed.
func (s *QuickExcludeService) ExcludePaths(ctx context.Context, paths []string) map[string]error {
	failures := make(map[string]error)

	for _, path := range paths {
		for _, backup := range s.backups {
			available, err := backup.IsAvailable(ctx)
			if err != nil || !available {
				log.Printf("backup %s not available: %v", backup.Name(), err)
				continue
			}

			if err := backup.Exclude(ctx, path); err != nil {
				failures[path] = fmt.Errorf("%s: %w", backup.Name(), err)
			}
		}
	}

	return failures
}

// RemoveExclusions removes exclusions from all configured backup systems for the given paths.
func (s *QuickExcludeService) RemoveExclusions(ctx context.Context, paths []string) map[string]error {
	failures := make(map[string]error)

	for _, path := range paths {
		for _, backup := range s.backups {
			available, err := backup.IsAvailable(ctx)
			if err != nil || !available {
				continue
			}

			if err := backup.Remove(ctx, path); err != nil {
				failures[path] = fmt.Errorf("%s: %w", backup.Name(), err)
			}
		}
	}

	return failures
}

// ListExclusions returns all paths currently excluded from any configured backup system.
func (s *QuickExcludeService) ListExclusions(ctx context.Context) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, backup := range s.backups {
		paths, err := backup.ListExcluded(ctx)
		if err != nil {
			log.Printf("listing exclusions from %s: %v", backup.Name(), err)
			continue
		}
		result[backup.Name()] = paths
	}

	return result, nil
}

// GetVolumes returns a list of available volumes (Home + mounted drives).
func GetVolumes() []domain.Volume {
	var volumes []domain.Volume

	// Home
	home, err := os.UserHomeDir()
	if err == nil {
		volumes = append(volumes, domain.Volume{
			Path: home,
			Name: "🏠 Home (" + home + ")",
		})
	}

	// /Volumes/
	out, err := exec.Command("df", "-h").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "/Volumes/") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					mnt := fields[len(fields)-1]
					if strings.HasPrefix(mnt, "/Volumes/") {
						name := filepath.Base(mnt)
						volumes = append(volumes, domain.Volume{
							Path: mnt,
							Name: "💿 " + name + " (" + mnt + ")",
						})
					}
				}
			}
		}
	}

	return volumes
}
