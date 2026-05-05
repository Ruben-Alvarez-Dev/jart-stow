package docker

import (
	"context"
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestParseDockerSize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"empty", "", 0},
		{"zero B", "0B", 0},
		{"zero lowercase", "0b", 0},
		{"bytes", "500B", 500},
		{"kilobytes", "1.5KB", 1536},
		{"megabytes", "10MB", 10 * 1024 * 1024},
		{"gigabytes", "1.5GB", int64(1.5 * 1024 * 1024 * 1024)},
		{"terabytes", "1TB", 1024 * 1024 * 1024 * 1024},
		{"lowercase kb", "2kb", 2048},
		{"spaces trimmed", "  100MB  ", 100 * 1024 * 1024},
		{"unknown suffix", "100XX", 0},
		{"invalid number", "abcGB", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDockerSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short id", "abc", "abc"},
		{"exactly 12 chars", "abcdefghijkl", "abcdefghijkl"},
		{"long id", "sha256:abcdef1234567890abcdef1234567890", "sha256:abcde"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDockerOutput(t *testing.T) {
	t.Run("empty output", func(t *testing.T) {
		items, err := parseDockerOutput("", 1, "docker image")
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("whitespace only", func(t *testing.T) {
		items, err := parseDockerOutput("\n  \n", 1, "docker image")
		assert.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("single image", func(t *testing.T) {
		output := "abc123def456\tabc123def456:<none>\t500MB\t2024-01-15 10:30:00 +0000 UTC"
		items, err := parseDockerOutput(output, 1, "docker image")
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "docker://abc123def456", items[0].Path)
		assert.Contains(t, items[0].Description, "docker image")
		assert.Contains(t, items[0].Description, "500MB")
		assert.Equal(t, int64(500*1024*1024), items[0].SizeBytes)
		assert.Equal(t, domain.VerificationPending, items[0].VerifiedByUser)
		assert.NotNil(t, items[0].LastAccessed)
		assert.Equal(t, int64(1), items[0].CategoryID)
	})

	t.Run("multiple images", func(t *testing.T) {
		output := "id1\timg1:tag1\t10MB\t2024-01-15 10:30:00 +0000 UTC\nid2\timg2:tag2\t20MB\t2024-01-16 10:30:00 +0000 UTC"
		items, err := parseDockerOutput(output, 1, "docker image")
		assert.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, int64(10*1024*1024), items[0].SizeBytes)
		assert.Equal(t, int64(20*1024*1024), items[1].SizeBytes)
	})

	t.Run("no size field", func(t *testing.T) {
		output := "id1\tname:tag"
		items, err := parseDockerOutput(output, 1, "docker image")
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, int64(0), items[0].SizeBytes)
	})

	t.Run("invalid date", func(t *testing.T) {
		output := "id1\tname:tag\t10MB\tnot-a-date"
		items, err := parseDockerOutput(output, 1, "docker image")
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Nil(t, items[0].LastAccessed)
	})

	t.Run("malformed line skipped", func(t *testing.T) {
		output := "only_one_field"
		items, err := parseDockerOutput(output, 1, "docker image")
		assert.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestScannerName(t *testing.T) {
	scanner := NewScanner()
	assert.Equal(t, "docker", scanner.Name())
}

func TestScannerIsAvailable_DockerNotFound(t *testing.T) {
	scanner := &Scanner{dockerPath: "no_such_docker_binary_xyz"}
	assert.False(t, scanner.IsAvailable(context.Background()))
}
