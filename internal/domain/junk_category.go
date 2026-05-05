package domain

import "time"

// ScannerName identifies the scanner module responsible for a junk category.
type ScannerName string

const (
	ScannerDocker     ScannerName = "docker"
	ScannerAPFS       ScannerName = "apfs"
	ScannerFilesystem ScannerName = "filesystem"
	ScannerCache      ScannerName = "cache"
	ScannerLogs       ScannerName = "logs"
	ScannerXcode      ScannerName = "xcode"
	ScannerBrew       ScannerName = "brew"
)

// JunkCategory defines a type of system junk that can be scanned and cleaned.
type JunkCategory struct {
	ID             int64       `json:"id"`
	Name           string      `json:"name"`
	Scanner        ScannerName `json:"scanner"`
	VerifyRequired bool        `json:"verify_required"`
	Enabled        bool        `json:"enabled"`
	CreatedAt      time.Time   `json:"created_at"`
}

// DefaultCategories returns the built-in junk categories inserted on first run.
func DefaultCategories() []JunkCategory {
	now := time.Now()
	return []JunkCategory{
		{Name: "unused_docker_images", Scanner: ScannerDocker, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "unused_docker_containers", Scanner: ScannerDocker, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "unused_docker_volumes", Scanner: ScannerDocker, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "docker_build_cache", Scanner: ScannerDocker, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "unused_apfs_snapshots", Scanner: ScannerAPFS, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "system_caches", Scanner: ScannerCache, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "user_caches", Scanner: ScannerCache, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "tmp_files", Scanner: ScannerFilesystem, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "xcode_derived_data", Scanner: ScannerXcode, VerifyRequired: true, Enabled: false, CreatedAt: now},
		{Name: "brew_cache", Scanner: ScannerBrew, VerifyRequired: false, Enabled: false, CreatedAt: now},
	}
}
