package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
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

			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			info, err := os.Stat(absPath)
			if err != nil {
				return fmt.Errorf("accessing path: %w", err)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", path)
			}

			scanSvc := services.NewScanService(0, nil)
			artifacts, err := scanSvc.FindArtifacts(context.Background(), absPath)
			if err != nil {
				return fmt.Errorf("scanning: %w", err)
			}

			// Build output in buffer to paginate if needed
			var b strings.Builder
			b.WriteString(fmt.Sprintf("🔍 Scanning %s for development artifacts...\n\n", absPath))

			if len(artifacts) == 0 {
				b.WriteString("✅ No development artifacts found.\n")
				cmd.Print(b.String())
				return nil
			}

			var totalSize int64
			b.WriteString(fmt.Sprintf("📦 Found %d development artifact directories:\n\n", len(artifacts)))
			for _, a := range artifacts {
				totalSize += a.SizeBytes
				b.WriteString(fmt.Sprintf("  • %s  (%s, %s)\n", a.Path, a.PatternName, formatBytesQuick(a.SizeBytes)))
			}

			b.WriteString(fmt.Sprintf("\nTotal: %d directories, %s\n", len(artifacts), formatBytesQuick(totalSize)))
			b.WriteString("\nRun 'jart-stow exclude on <path>' to add these to backup exclusion lists.\n")

			printPaged(cmd, b.String())
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
			cmd.Println("Daemon: not running (use 'jart-stow daemon status')")
			cmd.Println("Watch roots: none configured")
			cmd.Println("Tracked projects: 0")
			cmd.Println("Active exclusions: 0")
			return nil
		},
	}
}
