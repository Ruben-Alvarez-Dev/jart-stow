// Package cli provides the Cobra command-line interface for jart-stow.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCommand creates the root Cobra command for jart-stow.
func NewRootCommand() *cobra.Command {
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

	// Register subcommands
	cmd.AddCommand(newDaemonCommand())
	cmd.AddCommand(newScanCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newInspectCommand())
	cmd.AddCommand(newAuditCommand())
	cmd.AddCommand(newRuleCommand())
	cmd.AddCommand(newReportCommand())

	return cmd
}

// Execute runs the root command and exits on error.
func Execute() error {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("jart-stow: %w", err)
	}
	return nil
}
