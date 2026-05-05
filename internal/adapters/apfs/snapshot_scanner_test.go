package apfs
package apfs

import (
	"testing"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestExtractSnapshotDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid snapshot",
			input:    "com.apple.TimeMachine.2026-05-05-123456",
			expected: "2026-05-05-123456",
		},
		{
			name:     "valid snapshot with suffix",
			input:    "com.apple.TimeMachine.2024-12-25-000000.local",
			expected: "2024-12-25-000000",
		},
		{
			name:     "no dots",
			input:    "nodot",
			expected: "",
		},
		{
			name:     "single dot",
			input:    "prefix.suffix",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSnapshotDate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSnapshots(t *testing.T) {
	t.Run("empty output", func(t *testing.T) {
		items := parseSnapshots("", 1)
		assert.Empty(t, items)
	})

	t.Run("header only", func(t *testing.T) {
		output := `Snapshots for volume group containing disk "Macintosh HD":`
		items := parseSnapshots(output, 1)
		assert.Empty(t, items)
	})

	t.Run("single snapshot", func(t *testing.T) {
		output := `Snapshots for volume group containing disk "Macintosh HD":
com.apple.TimeMachine.2026-05-05-123456`
		items := parseSnapshots(output, 1)
		assert.Len(t, items, 1)
		assert.Equal(t, "apfs://com.apple.TimeMachine.2026-05-05-123456", items[0].Path)
		assert.Contains(t, items[0].Description, "APFS snapshot")
		assert.Equal(t, int64(0), items[0].SizeBytes)
		assert.Equal(t, domain.VerificationPending, items[0].VerifiedByUser)
		assert.Equal(t, int64(1), items[0].CategoryID)
		assert.NotNil(t, items[0].LastAccessed)
		expectedDate := time.Date(2026, 5, 5, 12, 34, 56, 0, time.UTC)
		assert.Equal(t, expectedDate, *items[0].LastAccessed)
	})

	t.Run("multiple snapshots", func(t *testing.T) {
		output := `Snapshots for volume group:
com.apple.TimeMachine.2026-05-05-123456
com.apple.TimeMachine.2026-05-04-123456`
		items := parseSnapshots(output, 1)
		assert.Len(t, items, 2)
	})

	t.Run("snapshot with local suffix", func(t *testing.T) {
		output := "com.apple.TimeMachine.2026-01-01-000000.local"
		items := parseSnapshots(output, 1)
		assert.Len(t, items, 1)
		assert.Equal(t, "apfs://com.apple.TimeMachine.2026-01-01-000000.local", items[0].Path)
	})

	t.Run("snapshot with unparseable date", func(t *testing.T) {
		output := "com.apple.TimeMachine.invalid-date"
		items := parseSnapshots(output, 2)
		assert.Len(t, items, 1)
		assert.Nil(t, items[0].LastAccessed)
	})
}

func TestScannerName(t *testing.T) {
	scanner := NewScanner()
	assert.Equal(t, "apfs", scanner.Name())
}

func TestScannerIsAvailable_TmutilNotFound(t *testing.T) {
	scanner := &Scanner{tmutilPath: "no_such_tmutil_binary_xyz"}
	// We don't pass a real context since LookPath doesn't need it
	assert.False(t, scanner.IsAvailable(nil))
}
