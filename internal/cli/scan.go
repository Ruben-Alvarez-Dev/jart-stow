package cli

import (
	"context"
	"fmt"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/spf13/cobra"
)

func newScanCommand(qs *services.QuickExcludeService) *cobra.Command {
	return &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a path for development artifacts",
		Long: `Scan a directory for development dependency folders (node_modules,
.venv, target, vendor, build, dist, __pycache__, etc.) and display
their size and exclusion status.

If no path is given, scans the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			cmd.Printf("Scanning %s for development artifacts...\n\n", path)

			if qs == nil {
				cmd.Println("No scan service available.")
				return nil
			}

			results, err := qs.Scan(ctx, path)
			if err != nil {
				return fmt.Errorf("scanning %s: %w", path, err)
			}

			if len(results) == 0 {
				cmd.Println("No development artifacts found.")
				return nil
			}

			var totalSize int64
			for _, r := range results {
				totalSize += r.SizeBytes
				status := " "
				if r.AlreadyDone {
					status = "✓"
				}
				cmd.Printf("  %s %-30s %s\n", status, formatSize(r.SizeBytes), r.Path)
			}

			cmd.Printf("\n%d artifacts found, %s total\n", len(results), formatSize(totalSize))

			excludedCount := 0
			for _, r := range results {
				if r.AlreadyDone {
					excludedCount++
				}
			}
			if excludedCount > 0 {
				cmd.Printf("%d already excluded from backups.\n", excludedCount)
				cmd.Printf("Run 'jart-stow exclude on %s' to exclude the remaining artifacts.\n", path)
			} else {
				cmd.Printf("None excluded yet. Run 'jart-stow exclude on %s' to apply exclusions.\n", path)
			}

			return nil
		},
	}
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current Jart-Stow system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Jart-Stow v0.1.0-dev")
			cmd.Println()
			cmd.Println("Run 'jart-stow daemon status' for daemon-specific status.")
			cmd.Println("Run 'jart-stow report' for a comprehensive report.")
			cmd.Println()
			cmd.Println("Commands:")
			cmd.Println("  scan [path]     Scan a path for dev artifacts")
			cmd.Println("  exclude on|off  Manage backup exclusions")
			cmd.Println("  inspect [path]  Inspect a project's details")
			cmd.Println("  audit           Verify exclusion consistency")
			cmd.Println("  rule list|add   Manage hygiene rules")
			cmd.Println("  report          Generate comprehensive report")
			cmd.Println("  daemon          Manage the background daemon")
			cmd.Println("  tui             Launch the terminal UI")
			return nil
		},
	}
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	var i int
	for i = 0; i < len(units)-1 && size >= 1024; i++ {
		size /= 1024
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}


