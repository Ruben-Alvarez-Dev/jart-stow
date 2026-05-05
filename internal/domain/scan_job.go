package domain

import "time"

// ScanStatus represents the current state of a scan job.
type ScanStatus string

const (
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
)

// ScanJob records a single scan execution against a project.
type ScanJob struct {
	ID              int64      `json:"id"`
	ProjectID       int64      `json:"project_id"`
	Status          ScanStatus `json:"status"`
	FoldersFound    int        `json:"folders_found"`
	TotalSizeBytes  int64      `json:"total_size_bytes"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

// IsFinished returns true if the scan job has completed or failed.
func (s ScanJob) IsFinished() bool {
	return s.Status == ScanStatusCompleted || s.Status == ScanStatusFailed
}
