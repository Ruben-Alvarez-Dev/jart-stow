// Package docker implements the JunkScanner for Docker resources.
package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
)

// Scanner discovers unused Docker resources: images, containers, volumes, and build cache.
type Scanner struct {
	dockerPath string
}

// NewScanner creates a new Docker junk scanner.
func NewScanner() *Scanner {
	return &Scanner{dockerPath: "docker"}
}

// Name returns the scanner identifier.
func (s *Scanner) Name() string {
	return "docker"
}

// IsAvailable checks whether Docker is installed and the daemon is reachable.
func (s *Scanner) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, s.dockerPath, "info")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// Scan discovers Docker junk based on the category type.
func (s *Scanner) Scan(ctx context.Context, category domain.JunkCategory) ([]domain.JunkItem, error) {
	if !s.IsAvailable(ctx) {
		return nil, nil
	}

	switch category.Name {
	case "unused_docker_images":
		return s.scanUnusedImages(ctx, category.ID)
	case "unused_docker_containers":
		return s.scanUnusedContainers(ctx, category.ID)
	case "unused_docker_volumes":
		return s.scanUnusedVolumes(ctx, category.ID)
	case "docker_build_cache":
		return s.scanBuildCache(ctx, category.ID)
	default:
		return nil, nil
	}
}

func (s *Scanner) scanUnusedImages(ctx context.Context, categoryID int64) ([]domain.JunkItem, error) {
	// docker images --filter "dangling=true" --format "{{.ID}}\t{{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
	cmd := exec.CommandContext(ctx, s.dockerPath, "images",
		"--filter", "dangling=true",
		"--format", "{{.ID}}\t{{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("listing dangling docker images: %w", err)
	}

	return parseDockerOutput(string(output), categoryID, "docker image")
}

func (s *Scanner) scanUnusedContainers(ctx context.Context, categoryID int64) ([]domain.JunkItem, error) {
	// docker ps -a --filter "status=exited" --format "{{.ID}}\t{{.Names}}\t{{.Size}}\t{{.CreatedAt}}"
	cmd := exec.CommandContext(ctx, s.dockerPath, "ps", "-a",
		"--filter", "status=exited",
		"--format", "{{.ID}}\t{{.Names}}\t{{.Size}}\t{{.CreatedAt}}",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("listing exited containers: %w", err)
	}

	return parseDockerOutput(string(output), categoryID, "docker container")
}

func (s *Scanner) scanUnusedVolumes(ctx context.Context, categoryID int64) ([]domain.JunkItem, error) {
	// docker volume ls --filter "dangling=true" --format "{{.Name}}\t{{.Mountpoint}}"
	cmd := exec.CommandContext(ctx, s.dockerPath, "volume", "ls",
		"--filter", "dangling=true",
		"--format", "{{.Name}}\t{{.Mountpoint}}",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("listing dangling volumes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var items []domain.JunkItem
	now := time.Now()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		items = append(items, domain.JunkItem{
			CategoryID:     categoryID,
			Path:           "docker://" + name,
			Description:    fmt.Sprintf("Docker volume %s (unused)", name),
			SizeBytes:      0, // Volumes don't report size easily
			VerifiedByUser: domain.VerificationPending,
			CreatedAt:      now,
		})
	}
	return items, nil
}

func (s *Scanner) scanBuildCache(ctx context.Context, categoryID int64) ([]domain.JunkItem, error) {
	// docker builder prune --all --force --filter "until=720h" actually prunes.
	// For scanning we use: docker system df --format "{{.Type}}\t{{.TotalCount}}\t{{.Size}}"
	cmd := exec.CommandContext(ctx, s.dockerPath, "system", "df",
		"--format", "{{.Type}}\t{{.TotalCount}}\t{{.Size}}\t{{.Reclaimable}}",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("checking docker disk usage: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var items []domain.JunkItem
	now := time.Now()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		dockerType := parts[0]
		reclaimable := "false"
		if len(parts) >= 4 {
			reclaimable = parts[3]
		}
		if dockerType != "Build Cache" || reclaimable != "true" {
			continue
		}
		sizeStr := parts[2]
		sizeBytes := parseDockerSize(sizeStr)
		count := parts[1]

		items = append(items, domain.JunkItem{
			CategoryID:     categoryID,
			Path:           "docker://build-cache",
			Description:    fmt.Sprintf("Docker build cache (%s items, %s reclaimable)", count, sizeStr),
			SizeBytes:      sizeBytes,
			VerifiedByUser: domain.VerificationPending,
			CreatedAt:      now,
		})
	}
	return items, nil
}

// parseDockerOutput parses tab-separated docker output.
// Format: ID\tName\tSize\tCreatedAt
func parseDockerOutput(output string, categoryID int64, prefix string) ([]domain.JunkItem, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var items []domain.JunkItem
	now := time.Now()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		id := truncateID(parts[0])
		name := parts[1]
		sizeStr := "0B"
		if len(parts) >= 3 {
			sizeStr = parts[2]
		}
		sizeBytes := parseDockerSize(sizeStr)
		createdAt := ""
		if len(parts) >= 4 {
			createdAt = parts[3]
		}

		var lastAccessed *time.Time
		if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", createdAt); err == nil {
			lastAccessed = &t
		}

		items = append(items, domain.JunkItem{
			CategoryID:     categoryID,
			Path:           "docker://" + id,
			Description:    fmt.Sprintf("%s %s (%s)", prefix, name, sizeStr),
			SizeBytes:      sizeBytes,
			LastAccessed:   lastAccessed,
			VerifiedByUser: domain.VerificationPending,
			CreatedAt:      now,
		})
	}
	return items, nil
}

// parseDockerSize converts docker size strings like "1.5GB", "500MB", "0B" to bytes.
func parseDockerSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" || s == "0B" {
		return 0
	}

	// Ordered longest-first to avoid "B" matching before "KB" or "MB"
	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, sfx := range suffixes {
		if strings.HasSuffix(s, sfx.suffix) {
			numStr := strings.TrimSuffix(s, sfx.suffix)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0
			}
			return int64(num * float64(sfx.mult))
		}
	}
	return 0
}

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
