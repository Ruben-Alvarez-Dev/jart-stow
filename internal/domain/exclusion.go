package domain

import "time"

// BackupSystem identifies the target backup system for an exclusion.
type BackupSystem string

const (
	BackupSystemTimeMachine       BackupSystem = "time_machine"
	BackupSystemCarbonCopyCloner  BackupSystem = "carbon_copy_cloner"
	BackupSystemBoth              BackupSystem = "both"
)

// Exclusion records a folder excluded from a backup system.
type Exclusion struct {
	ID             int64        `json:"id"`
	ProjectID      int64        `json:"project_id"`
	FolderPath     string       `json:"folder_path"`
	PatternMatched string       `json:"pattern_matched"`
	BackupSystem   BackupSystem `json:"backup_system"`
	SizeBytes      int64        `json:"size_bytes"`
	AppliedAt      time.Time    `json:"applied_at"`
	RemovedAt      *time.Time   `json:"removed_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
}

// IsActive returns true if the exclusion has not been removed.
func (e Exclusion) IsActive() bool {
	return e.RemovedAt == nil
}
