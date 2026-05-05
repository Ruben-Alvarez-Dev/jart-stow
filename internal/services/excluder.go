package services

import (
	"context"
	"fmt"
	"log"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// ExcludeService coordinates development artifact exclusion across backup systems.
type ExcludeService struct {
	projectRepo   ports.ProjectRepository
	exclusionRepo ports.ExclusionRepository
	backups       []ports.BackupProvider
	scanService   *ScanService
}

// NewExcludeService creates a new ExcludeService with constructor injection.
func NewExcludeService(
	projectRepo ports.ProjectRepository,
	exclusionRepo ports.ExclusionRepository,
	scanService *ScanService,
	backups ...ports.BackupProvider,
) *ExcludeService {
	return &ExcludeService{
		projectRepo:   projectRepo,
		exclusionRepo: exclusionRepo,
		scanService:   scanService,
		backups:       backups,
	}
}

// ExcludeProject scans the project for artifacts and applies exclusions to all configured backup systems.
// Returns the number of new exclusions applied and the total size excluded.
func (s *ExcludeService) ExcludeProject(ctx context.Context, projectPath string) (int, int64, error) {
	// 1. Ensure project exists in DB
	project, err := s.projectRepo.FindByPath(ctx, projectPath)
	if err != nil {
		return 0, 0, fmt.Errorf("finding project %s: %w", projectPath, err)
	}

	// 2. Scan for artifacts
	artifacts, err := s.scanService.FindArtifacts(ctx, projectPath)
	if err != nil {
		return 0, 0, fmt.Errorf("scanning artifacts in %s: %w", projectPath, err)
	}

	// 3. Apply exclusions for each artifact
	var count int
	var totalSize int64

	for _, artifact := range artifacts {
		// Check if already excluded
		existing, err := s.exclusionRepo.FindByPath(ctx, artifact.Path)
		if err != nil && err != domain.ErrExclusionNotFound {
			return count, totalSize, fmt.Errorf("checking exclusion %s: %w", artifact.Path, err)
		}
		if existing != nil {
			continue
		}

		// Apply to each backup system
		allAvailable := true
		backupSystem := domain.BackupSystemBoth

		for _, backup := range s.backups {
			available, err := backup.IsAvailable(ctx)
			if err != nil {
				log.Printf("checking backup %s availability: %v", backup.Name(), err)
				allAvailable = false
				continue
			}
			if !available {
				allAvailable = false
				continue
			}

			if err := backup.Exclude(ctx, artifact.Path); err != nil {
				log.Printf("excluding %s from %s: %v", artifact.Path, backup.Name(), err)
				allAvailable = false
			}
		}

		if !allAvailable {
			// At least one backup system worked; mark as "timemachine" or "ccc" individually
			// We simplify here: if both work → "both", otherwise check individually
			backupSystem = determineBackupSystem(ctx, s.backups, artifact.Path)
		}

		// Save exclusion record
		_, err = s.exclusionRepo.Save(ctx, &domain.Exclusion{
			ProjectID:      project.ID,
			FolderPath:     artifact.Path,
			PatternMatched: artifact.PatternName,
			BackupSystem:   backupSystem,
			SizeBytes:      artifact.SizeBytes,
		})
		if err != nil {
			log.Printf("saving exclusion %s: %v", artifact.Path, err)
			continue
		}

		count++
		totalSize += artifact.SizeBytes
	}

	return count, totalSize, nil
}

// determineBackupSystem checks which backup systems successfully excluded a path.
func determineBackupSystem(ctx context.Context, backups []ports.BackupProvider, path string) domain.BackupSystem {
	var tmAvailable, cccAvailable bool
	for _, b := range backups {
		excluded, err := b.IsExcluded(ctx, path)
		if err != nil || !excluded {
			continue
		}
		switch b.Name() {
		case "time_machine":
			tmAvailable = true
		case "carbon_copy_cloner":
			cccAvailable = true
		}
	}
	if tmAvailable && cccAvailable {
		return domain.BackupSystemBoth
	}
	if tmAvailable {
		return domain.BackupSystemTimeMachine
	}
	if cccAvailable {
		return domain.BackupSystemCarbonCopyCloner
	}
	return domain.BackupSystemBoth // Default when unknown
}
