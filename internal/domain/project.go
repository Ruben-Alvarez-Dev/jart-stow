// Package domain defines the core entities and value objects for Jart-Stow.
// This package has zero external dependencies.
package domain

import "time"

// ProjectStatus represents the lifecycle state of a tracked project.
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusIgnored  ProjectStatus = "ignored"
	ProjectStatusArchived ProjectStatus = "archived"
)

// Project represents a development project directory discovered under a watch root.
type Project struct {
	ID          int64         `json:"id"`
	Path        string        `json:"path"`
	Name        string        `json:"name"`
	RootPath    string        `json:"root_path"`
	LastScanned *time.Time    `json:"last_scanned,omitempty"`
	Status      ProjectStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
