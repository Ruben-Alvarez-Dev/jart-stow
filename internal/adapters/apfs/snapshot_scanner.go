// Package apfs implements the JunkScanner for APFS snapshots.
package apfs

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
)

// Scanner discovers unused APFS snapshots (typically created by Time Machine).
type Scanner struct {
	tmutilPath string
}

// NewScanner creates a new APFS snapshot scanner.
func NewScanner() *Scanner {
	return &Scanner{tmutilPath: "tmutil"}
}

// Name returns the scanner identifier.
func (s *Scanner) Name() string {
	return "apfs"
}

// IsAvailable checks whether tmutil is accessible.
func (s *Scanner) IsAvailable(ctx context.Context) bool {
	_, err := exec.LookPath(s.tmutilPath)
	return err == nil
}

// Scan discovers unused APFS snapshots.
func (s *Scanner) Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error) {
	if category.Name != "unused_apfs_snapshots" {
		return nil, nil
	}

	if !s.IsAvailable(ctx) {
		return nil, nil
	}

	// tmutil listlocalsnapshots /
	cmd := exec.CommandContext(ctx, s.tmutilPath, "listlocalsnapshots", "/")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("listing local snapshots: %w", err)
	}

	return parseSnapshots(string(output), category.ID), nil
}

// parseSnapshots parses tmutil output like:
// Snapshots for volume group containing disk "Macintosh HD":
// com.apple.TimeMachine.2026-05-05-123456
// com.apple.TimeMachine.2026-05-04-123456
func parseSnapshots(output string, categoryID int64) []domain.JunkItem {
	var items []domain.JunkItem
	now := time.Now()
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Snapshots for") {
			continue
		}

		// Extract date from snapshot name
		// Format: com.apple.TimeMachine.2026-05-05-123456
		dateStr := extractSnapshotDate(line)
		var snapshotTime *time.Time
		if dateStr != "" {
			if t, err := time.Parse("2006-01-02-150405", dateStr); err == nil {
				snapshotTime = &t
			}
		}

		// Estimate snapshot size using tmutil (this is slow, so default to 0)
		sizeBytes := estimateSnapshotSize(line)

		items = append(items, domain.JunkItem{
			CategoryID:     categoryID,
			Path:           "apfs://" + line,
			Description:    fmt.Sprintf("APFS snapshot %s", line),
			SizeBytes:      sizeBytes,
			LastAccessed:   snapshotTime,
			VerifiedByUser: domain.VerificationPending,
			CreatedAt:      now,
		})
	}
	return items
}

// extractSnapshotDate extracts the date portion from a snapshot name.
// Snapshot names are like "com.apple.TimeMachine.2026-05-05-123456" or
// "com.apple.TimeMachine.2026-05-05-123456.local".
// The date is the dot-separated token matching YYYY-MM-DD-HHmmss.
func extractSnapshotDate(name string) string {
	parts := strings.Split(name, ".")
	// Scan backwards to find the date token (always the first numeric token from the end).
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// Date tokens match YYYY-MM-DD-HHmmss: 4 digits, dash, 2, dash, 2, dash, 6
		if len(part) == 17 && part[4] == '-' && part[7] == '-' && part[10] == '-' {
			return part
		}
	}
	return ""
}

// estimateSnapshotSize estimates the size of an APFS snapshot.
// Since direct size queries are expensive, we return 0 as a placeholder.
// Future enhancement: use `tmutil thinlocalsnapshots / <size>` dry-run.
func estimateSnapshotSize(_ string) int64 {
	return 0
}
