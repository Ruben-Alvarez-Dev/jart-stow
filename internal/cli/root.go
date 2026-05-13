// Package cli provides the Cobra command-line interface for jart-stow.
package cli

import (
	"fmt"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/spf13/cobra"
)

// CLIDependencies bundles all optional service dependencies for CLI commands.
// Fields may be nil; commands degrade gracefully when a dependency is missing.
type CLIDependencies struct {
	QuickExclude *services.QuickExcludeService
	Auditor      *services.AuditService
	Reporter     *services.ReportService
	RuleRepo     ports.RuleRepository
	ProjectRepo  ports.ProjectRepository
	ExclusionRepo ports.ExclusionRepository
	EventRepo    ports.EventRepository
}

// NewRootCommand creates the root Cobra command for jart-stow.
func NewRootCommand(deps *CLIDependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jart-stow",
		Short: "macOS development hygiene & backup exclusion manager",
		Long: `Jart-Stow monitors configurable workspace roots for development artifacts
and applies exclusions to Time Machine and Carbon Copy Cloner.

It also detects system junk (Docker, APFS snapshots, caches, temp files)
for user-reviewed cleanup.`,
		Version: "0.1.0-dev",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Register subcommands (order matches help output priority)
	cmd.AddCommand(newDaemonCommand())
	cmd.AddCommand(newScanCommand(deps.QuickExclude))
	cmd.AddCommand(newStatusCommand())

	if deps.QuickExclude != nil {
		cmd.AddCommand(NewExcludeCommand(deps.QuickExclude))
	}

	if deps.Auditor != nil {
		cmd.AddCommand(newInspectCommand(deps.Auditor))
		cmd.AddCommand(newAuditCommand(deps.Auditor))
	}

	if deps.RuleRepo != nil {
		cmd.AddCommand(newRuleCommand(deps.RuleRepo))
	}

	if deps.Reporter != nil {
		cmd.AddCommand(newReportCommand(deps.Reporter))
	}

	// TUI is registered in main.go via AddCommand after NewRootCommand returns
	// to avoid circular dependency on the Bubble Tea program

	return cmd
}

// Execute runs the root command with the given dependencies and exits on error.
func Execute(deps *CLIDependencies) error {
	cmd := NewRootCommand(deps)
	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("jart-stow: %w", err)
	}
	return nil
}
