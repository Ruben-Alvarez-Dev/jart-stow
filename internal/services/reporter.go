// Package services — ReportService generates hygiene and exclusion reports
// with aggregate statistics, breakdowns by pattern and system, and event history.
package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
)

// PatternBreakdown groups exclusion data by matched pattern (node_modules, .venv, etc.).
type PatternBreakdown struct {
	Pattern    string `json:"pattern"`
	Count      int    `json:"count"`
	SizeBytes  int64  `json:"size_bytes"`
}

// SystemBreakdown groups exclusion data by backup system.
type SystemBreakdown struct {
	System   string `json:"system"`
	Count    int    `json:"count"`
	SizeBytes int64 `json:"size_bytes"`
}

// ReportSummary holds all aggregate statistics for the report.
type ReportSummary struct {
	ProjectsTotal        int                `json:"projects_total"`
	ProjectsActive       int                `json:"projects_active"`
	ExclusionsActive     int                `json:"exclusions_active"`
	ExclusionsTotalSize  int64              `json:"exclusions_total_size"`
	PatternBreakdowns    []PatternBreakdown `json:"pattern_breakdowns"`
	SystemBreakdowns     []SystemBreakdown  `json:"system_breakdowns"`
	EventsToday          int                `json:"events_today"`
	JunkItemsPending     int                `json:"junk_items_pending"`
}

// HistoryEntry represents one day of exclusion activity.
type HistoryEntry struct {
	Date           string `json:"date"`
	AddedCount     int    `json:"added_count"`
	RemovedCount   int    `json:"removed_count"`
	AddedSizeBytes int64  `json:"added_size_bytes"`
}

// ReportService generates aggregate reports from all data sources.
type ReportService struct {
	projectRepo   ports.ProjectRepository
	exclusionRepo ports.ExclusionRepository
	eventRepo     ports.EventRepository
	junkItemRepo  ports.JunkItemRepository
}

// NewReportService creates a ReportService with constructor injection.
func NewReportService(
	projectRepo ports.ProjectRepository,
	exclusionRepo ports.ExclusionRepository,
	eventRepo ports.EventRepository,
	junkItemRepo ports.JunkItemRepository,
) *ReportService {
	return &ReportService{
		projectRepo:   projectRepo,
		exclusionRepo: exclusionRepo,
		eventRepo:     eventRepo,
		junkItemRepo:  junkItemRepo,
	}
}

// GenerateSummary collects all aggregate statistics for the report.
func (s *ReportService) GenerateSummary(ctx context.Context) (*ReportSummary, error) {
	summary := &ReportSummary{}

	// Projects
	if projects, err := s.projectRepo.FindAll(ctx); err == nil {
		summary.ProjectsTotal = len(projects)
		for _, p := range projects {
			if p.Status == domain.ProjectStatusActive {
				summary.ProjectsActive++
			}
		}
	}

	// Exclusions
	if exclusions, err := s.exclusionRepo.FindActive(ctx); err == nil {
		summary.ExclusionsActive = len(exclusions)

		patternMap := make(map[string]*PatternBreakdown)
		systemMap := make(map[string]*SystemBreakdown)

		for _, e := range exclusions {
			summary.ExclusionsTotalSize += e.SizeBytes

			// By pattern
			pb, ok := patternMap[e.PatternMatched]
			if !ok {
				pb = &PatternBreakdown{Pattern: e.PatternMatched}
				patternMap[e.PatternMatched] = pb
			}
			pb.Count++
			pb.SizeBytes += e.SizeBytes

			// By system
			systemKey := string(e.BackupSystem)
			sb, ok := systemMap[systemKey]
			if !ok {
				sb = &SystemBreakdown{System: systemKey}
				systemMap[systemKey] = sb
			}
			sb.Count++
			sb.SizeBytes += e.SizeBytes
		}

		// Sort breakdowns by size descending
		for _, pb := range patternMap {
			summary.PatternBreakdowns = append(summary.PatternBreakdowns, *pb)
		}
		sort.Slice(summary.PatternBreakdowns, func(i, j int) bool {
			return summary.PatternBreakdowns[i].SizeBytes > summary.PatternBreakdowns[j].SizeBytes
		})

		for _, sb := range systemMap {
			summary.SystemBreakdowns = append(summary.SystemBreakdowns, *sb)
		}
		sort.Slice(summary.SystemBreakdowns, func(i, j int) bool {
			return summary.SystemBreakdowns[i].SizeBytes > summary.SystemBreakdowns[j].SizeBytes
		})
	}

	// Events today
	if count, err := s.eventRepo.CountToday(ctx); err == nil {
		summary.EventsToday = count
	}

	// Pending junk items
	if items, err := s.junkItemRepo.FindPending(ctx); err == nil {
		summary.JunkItemsPending = len(items)
	}

	return summary, nil
}

// GenerateHistory returns exclusion activity history grouped by day.
func (s *ReportService) GenerateHistory(ctx context.Context, days int) ([]HistoryEntry, error) {
	// Load all exclusions and group by date
	exclusions, err := s.exclusionRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading exclusions for history: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	// Build a map of date -> entry
	dateMap := make(map[string]*HistoryEntry)

	for _, e := range exclusions {
		createdDate := e.CreatedAt.Format("2006-01-02")

		// Skip entries older than the cutoff
		if e.CreatedAt.Before(cutoff) {
			continue
		}

		entry, ok := dateMap[createdDate]
		if !ok {
			entry = &HistoryEntry{Date: createdDate}
			dateMap[createdDate] = entry
		}
		entry.AddedCount++
		entry.AddedSizeBytes += e.SizeBytes

		// If removed, count as removal on the removed date
		if e.RemovedAt != nil {
			removedDate := e.RemovedAt.Format("2006-01-02")
			if e.RemovedAt.After(cutoff) || e.RemovedAt.Equal(cutoff) {
				removedEntry, ok := dateMap[removedDate]
				if !ok {
					removedEntry = &HistoryEntry{Date: removedDate}
					dateMap[removedDate] = removedEntry
				}
				removedEntry.RemovedCount++
			}
		}
	}

	// Convert to sorted slice
	entries := make([]HistoryEntry, 0, len(dateMap))
	for _, entry := range dateMap {
		entries = append(entries, *entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date > entries[j].Date // most recent first
	})

	return entries, nil
}

// FormatSummary returns a human-readable report.
func FormatSummary(summary *ReportSummary) string {
	var b strings.Builder

	b.WriteString("=== Jart-Stow Report ===\n\n")

	b.WriteString("Projects:\n")
	b.WriteString(fmt.Sprintf("  Total:  %d\n", summary.ProjectsTotal))
	b.WriteString(fmt.Sprintf("  Active: %d\n\n", summary.ProjectsActive))

	b.WriteString("Exclusions:\n")
	b.WriteString(fmt.Sprintf("  Active:     %d\n", summary.ExclusionsActive))
	b.WriteString(fmt.Sprintf("  Total Size: %s\n\n", formatBytesInt64(summary.ExclusionsTotalSize)))

	if len(summary.PatternBreakdowns) > 0 {
		b.WriteString("Breakdown by Pattern:\n")
		for _, pb := range summary.PatternBreakdowns {
			b.WriteString(fmt.Sprintf("  %-20s %4d  %s\n", pb.Pattern, pb.Count, formatBytesInt64(pb.SizeBytes)))
		}
		b.WriteString("\n")
	}

	if len(summary.SystemBreakdowns) > 0 {
		b.WriteString("Breakdown by System:\n")
		for _, sb := range summary.SystemBreakdowns {
			b.WriteString(fmt.Sprintf("  %-25s %4d  %s\n", sb.System, sb.Count, formatBytesInt64(sb.SizeBytes)))
		}
		b.WriteString("\n")
	}

	b.WriteString("Activity:\n")
	b.WriteString(fmt.Sprintf("  Events Today:      %d\n", summary.EventsToday))
	b.WriteString(fmt.Sprintf("  Junk Items Pending: %d\n", summary.JunkItemsPending))

	return b.String()
}

// FormatHistory returns a human-readable history.
func FormatHistory(entries []HistoryEntry) string {
	var b strings.Builder

	b.WriteString("=== Exclusion History ===\n\n")
	b.WriteString(fmt.Sprintf("%-12s %6s %8s %12s\n", "Date", "Added", "Removed", "Size"))
	b.WriteString(strings.Repeat("-", 42) + "\n")

	for _, e := range entries {
		b.WriteString(fmt.Sprintf("%-12s %6d %8d %12s\n",
			e.Date, e.AddedCount, e.RemovedCount, formatBytesInt64(e.AddedSizeBytes)))
	}

	return b.String()
}
