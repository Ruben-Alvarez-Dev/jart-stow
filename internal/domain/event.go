package domain

import "time"

// EventType categorizes daemon lifecycle and operational events.
type EventType string

const (
	EventTypeProjectDetected   EventType = "project_detected"
	EventTypeScanCompleted     EventType = "scan_completed"
	EventTypeExclusionApplied  EventType = "exclusion_applied"
	EventTypeExclusionRemoved  EventType = "exclusion_removed"
	EventTypeError             EventType = "error"
	EventTypeDaemonStarted     EventType = "daemon_started"
	EventTypeDaemonStopped     EventType = "daemon_stopped"
)

// DaemonEvent represents an auditable event emitted by the daemon.
type DaemonEvent struct {
	ID         int64      `json:"id"`
	EventType  EventType  `json:"event_type"`
	ProjectID  *int64     `json:"project_id,omitempty"`
	FolderPath string     `json:"folder_path,omitempty"`
	Details    string     `json:"details,omitempty"` // JSON payload
	CreatedAt  time.Time  `json:"created_at"`
}
