package domain

import "time"

// CleanupJob records a single cleanup operation performed by the user.
type CleanupJob struct {
	ID             int64     `json:"id"`
	CategoryID     *int64    `json:"category_id,omitempty"`
	ItemsCount     int       `json:"items_count"`
	TotalSizeBytes int64     `json:"total_size_bytes"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
}
