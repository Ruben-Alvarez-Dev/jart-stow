package ccc
// Package ccc implements the BackupProvider port for Carbon Copy Cloner exclusions.
package ccc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// Ensure Adapter implements ports.BackupProvider.
var _ ports.BackupProvider = (*Adapter)(nil)

// DefaultExclusionsPath is the standard CCC exclusions file path.
const DefaultExclusionsPath = "/Library/Application Support/com.bombich.ccc/Exclusions.txt"

// Adapter manages Carbon Copy Cloner exclusion rules via Exclusions.txt.
type Adapter struct {
	exclusionsPath string
}

// NewAdapter creates a new CCC adapter. If exclusionsPath is empty, it defaults
// to the standard CCC Exclusions.txt location.
func NewAdapter(exclusionsPath string) *Adapter {
	if exclusionsPath == "" {
		exclusionsPath = DefaultExclusionsPath
	}
	return &Adapter{exclusionsPath: exclusionsPath}
}

// Name returns the backup system identifier.
func (a *Adapter) Name() string {
	return "carbon_copy_cloner"
}

// IsAvailable checks whether the CCC exclusions file exists and is writable.
func (a *Adapter) IsAvailable(ctx context.Context) (bool, error) {
	info, err := os.Stat(a.exclusionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking CCC exclusions: %w", err)
	}
	return !info.IsDir(), nil
}

// Exclude adds a path to the CCC exclusions file.
func (a *Adapter) Exclude(ctx context.Context, path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(a.exclusionsPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating CCC directory %s: %w", dir, err)
	}

	// Check if already excluded
	excluded, err := a.IsExcluded(ctx, path)
	if err != nil {
		return err
	}
	if excluded {
		return nil
	}

	// Append to file
	f, err := os.OpenFile(a.exclusionsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening CCC exclusions: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, path); err != nil {
		return fmt.Errorf("writing CCC exclusion: %w", err)
	}

	return nil
}

// Remove removes a path from the CCC exclusions file.
func (a *Adapter) Remove(ctx context.Context, path string) error {
	excluded, err := a.IsExcluded(ctx, path)
	if err != nil {
		return err
	}
	if !excluded {
		return nil
	}

	// Read all lines, filter out the path, write back
	lines, err := a.readAllLines()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(a.exclusionsPath, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("opening CCC exclusions for removal: %w", err)
	}
	defer f.Close()

	for _, line := range lines {
		if strings.TrimSpace(line) == path {
			continue
		}
		fmt.Fprintln(f, line)
	}

	return nil
}

// IsExcluded checks whether a path is currently in the CCC exclusions file.
func (a *Adapter) IsExcluded(ctx context.Context, path string) (bool, error) {
	lines, err := a.readAllLines()
	if err != nil {
		return false, err
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == path {
			return true, nil
		}
	}
	return false, nil
}

// ListExcluded returns all paths currently in the CCC exclusions file.
func (a *Adapter) ListExcluded(ctx context.Context) ([]string, error) {
	lines, err := a.readAllLines()
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

// readAllLines reads all lines from the exclusions file.
// Returns an empty slice if the file does not exist.
func (a *Adapter) readAllLines() ([]string, error) {
	f, err := os.Open(a.exclusionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading CCC exclusions: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
