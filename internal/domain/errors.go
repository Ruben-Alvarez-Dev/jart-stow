package domain

import "fmt"

// Domain errors — used across all layers.
var (
	ErrProjectNotFound          = fmt.Errorf("project not found")
	ErrProjectAlreadyExists     = fmt.Errorf("project already exists")
	ErrExclusionNotFound        = fmt.Errorf("exclusion not found")
	ErrExclusionAlreadyExists   = fmt.Errorf("exclusion already exists for this path")
	ErrRuleNotFound             = fmt.Errorf("rule not found")
	ErrWatchRootNotFound        = fmt.Errorf("watch root not found")
	ErrWatchRootAlreadyExists   = fmt.Errorf("watch root already exists")
	ErrJunkCategoryNotFound     = fmt.Errorf("junk category not found")
	ErrJunkItemNotFound         = fmt.Errorf("junk item not found")
	ErrScanJobNotFound          = fmt.Errorf("scan job not found")
	ErrInvalidProjectStatus     = fmt.Errorf("invalid project status")
	ErrInvalidBackupSystem      = fmt.Errorf("invalid backup system")
	ErrInvalidRuleAction        = fmt.Errorf("invalid rule action")
	ErrInvalidEventType         = fmt.Errorf("invalid event type")
	ErrInvalidScanStatus        = fmt.Errorf("invalid scan status")
	ErrInvalidVerificationState = fmt.Errorf("invalid verification state")
	ErrDatabaseLocked           = fmt.Errorf("database is locked")
	ErrPathNotAbsolute          = fmt.Errorf("path must be absolute")
	ErrPermissionDenied         = fmt.Errorf("permission denied")
)
