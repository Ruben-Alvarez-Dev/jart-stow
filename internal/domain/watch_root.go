package domain

import "time"

// WatchRoot represents a directory that the daemon monitors for project detection.
type WatchRoot struct {
	ID         int64     `json:"id"`
	Path       string    `json:"path"`
	VolumeUUID string    `json:"volume_uuid,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}
