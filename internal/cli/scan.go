package cli

import (
	"github.com/spf13/cobra"
)

func newScanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a directory for development artifacts",
		Long: `Scan a project directory for development artifacts (node_modules, .venv, target/, etc.)
and show what would be excluded from backups.

If no path is provided, scans the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			cmd.Printf("Scanning %s for development artifacts...\n", path)
			// TODO(#10): Wire up ScanService when main.go is ready
			cmd.Println("Scan complete. Wire up ScanService to see results.")
			return nil
		},
	}
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current Jart-Stow system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(#10): Wire up repositories to show real data
			cmd.Println("Jart-Stow v0.1.0-dev")
			cmd.Println()
			cmd.Println("Daemon: not running (use 'jart-stow daemon status')")
			cmd.Println("Watch roots: none configured")
			cmd.Println("Tracked projects: 0")
			cmd.Println("Active exclusions: 0")
			return nil
		},
	}
}
