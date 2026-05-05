// Package tmutil implements the BackupProvider port for Time Machine exclusions.
package tmutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure Adapter implements ports.BackupProvider.
var _ ports.BackupProvider = (*Adapter)(nil)

// Adapter wraps macOS tmutil for Time Machine exclusion management.
type Adapter struct {
	tmutilPath string
}

// NewAdapter creates a new tmutil adapter. If tmutilPath is empty, it defaults
// to finding tmutil via $PATH.
func NewAdapter(tmutilPath string) *Adapter {
	if tmutilPath == "" {
		tmutilPath = "tmutil"
	}
	return &Adapter{tmutilPath: tmutilPath}
}

// Name returns the backup system identifier.
func (a *Adapter) Name() string {
	return "time_machine"
}

// IsAvailable checks whether tmutil is installed and accessible.
func (a *Adapter) IsAvailable(ctx context.Context) (bool, error) {
	_, err := exec.LookPath(a.tmutilPath)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Exclude adds a path to the Time Machine exclusion list.
func (a *Adapter) Exclude(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, a.tmutilPath, "addexclusion", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmutil addexclusion %s: %w: %s", path, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Remove removes a path from the Time Machine exclusion list.
func (a *Adapter) Remove(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, a.tmutilPath, "removeexclusion", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmutil removeexclusion %s: %w: %s", path, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// IsExcluded checks whether a path is currently excluded from Time Machine.
func (a *Adapter) IsExcluded(ctx context.Context, path string) (bool, error) {
	cmd := exec.CommandContext(ctx, a.tmutilPath, "isexcluded", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("tmutil isexcluded %s: %w: %s", path, err, strings.TrimSpace(string(output)))
	}
	// tmutil isexcluded prints "[Excluded] /path" if excluded
	return strings.Contains(string(output), "[Excluded]"), nil
}

// ListExcluded returns all currently excluded paths from Time Machine.
func (a *Adapter) ListExcluded(ctx context.Context) ([]string, error) {
	// tmutil doesn't have a direct "list all exclusions" command.
	// We parse the output of `tmutil isexcluded` for common paths
	// or use `mdfind` with the com_apple_backup_excludeItem attribute as fallback.
	cmd := exec.CommandContext(ctx, "mdfind", "com_apple_backup_excludeItem == 'com.apple.backupd'")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// mdfind may not be available or Spotlight may be disabled
		return nil, fmt.Errorf("listing excluded paths: %w: %s", err, strings.TrimSpace(string(output)))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var paths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}
