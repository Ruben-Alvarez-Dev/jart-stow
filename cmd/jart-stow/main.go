// Package main is the entry point for the jart-stow binary.
// It wires all dependencies via constructor injection and starts the application.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/apfs"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/ccc"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/docker"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/filesystem"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/fsevents"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/sqlite"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/adapters/tmutil"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/cli"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/tui/core"

	"github.com/spf13/cobra"
)

func getDBPath() string {
	if p := os.Getenv("JART_STOW_DB_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "jart-stow", "jart-stow.db")
}

func main() {
	// Create the backup providers
	backups := []ports.BackupProvider{
		tmutil.NewAdapter(""),
		ccc.NewAdapter(""),
	}

	// Create the scan service
	scanService := services.NewScanService(0, nil)

	// Create the quick exclude service (no DB needed)
	quickExcludeSvc := services.NewQuickExcludeService(scanService, backups...)

	// Try to open DB for full-featured CLI commands
	var deps *cli.CLIDependencies
	var tuiDeps *core.TUIDependencies

	dbPath := getDBPath()
	conn, dbErr := sqlite.NewConnection(context.Background(), dbPath)
	if dbErr == nil {
		defer conn.Close()
		repos := sqlite.NewRepositorySet(conn)

		excludeService := services.NewExcludeService(
			repos.Projects, repos.Exclusions, scanService, backups...,
		)
		junkService := createJunkScanService()

		auditor := services.NewAuditService(
			repos.Projects, repos.Exclusions, backups...,
		)
		reporter := services.NewReportService(
			repos.Projects, repos.Exclusions, repos.Events, repos.JunkItems,
		)

		deps = &cli.CLIDependencies{
			QuickExclude:  quickExcludeSvc,
			Auditor:       auditor,
			Reporter:      reporter,
			RuleRepo:      repos.Rules,
			ProjectRepo:   repos.Projects,
			ExclusionRepo: repos.Exclusions,
			EventRepo:     repos.Events,
		}

		tuiDeps = &core.TUIDependencies{
			QuickExclude:  quickExcludeSvc,
			ScanService:   scanService,
			ExcludeService: excludeService,
			Auditor:       auditor,
			Reporter:      reporter,
			JunkService:   junkService,
			ProjectRepo:   repos.Projects,
			ExclusionRepo: repos.Exclusions,
			RuleRepo:      repos.Rules,
			EventRepo:     repos.Events,
			JunkCatRepo:   repos.JunkCategories,
			JunkItemRepo:  repos.JunkItems,
			WatchRootRepo: repos.WatchRoots,
			DBAvailable:   true,
		}

		_ = excludeService
		_ = junkService
	} else {
		log.Printf("warning: database not available (%v); some commands disabled", dbErr)
		deps = &cli.CLIDependencies{
			QuickExclude: quickExcludeSvc,
		}
		tuiDeps = &core.TUIDependencies{
			QuickExclude: quickExcludeSvc,
			ScanService:  scanService,
			DBAvailable:  false,
		}
	}

	// Create root command
	cmd := cli.NewRootCommand(deps)

	// Register tui subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "tui",
		Short: "Launch the terminal user interface",
		RunE: func(_ *cobra.Command, _ []string) error {
			return tui.Run(tuiDeps)
		},
	})

	// Override daemon run with real wiring
	if daemonCmd := findCobraSubcommand(cmd, "daemon"); daemonCmd != nil {
		if runCmd := findCobraSubcommand(daemonCmd, "run"); runCmd != nil {
			runCmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
				return runDaemon(cobraCmd, args)
			}
		}
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func findCobraSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// runDaemon runs the daemon in foreground mode with all real services wired.
func runDaemon(_ *cobra.Command, _ []string) error {
	log.SetFlags(0)

	dbPath := getDBPath()
	ctx := context.Background()

	conn, err := sqlite.NewConnection(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer conn.Close()

	repos := sqlite.NewRepositorySet(conn)

	backups := []ports.BackupProvider{
		tmutil.NewAdapter(""),
		ccc.NewAdapter(""),
	}

	scanService := services.NewScanService(0, nil)
	excludeService := services.NewExcludeService(
		repos.Projects, repos.Exclusions, scanService, backups...,
	)
	junkService := createJunkScanService()

	watcher, err := fsevents.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	watcher.Start()

	monitor := services.NewMonitorService(
		repos.Projects, repos.Exclusions, repos.Events, repos.ScanJobs,
		repos.WatchRoots, repos.JunkCategories, repos.JunkItems,
		watcher, excludeService, junkService,
		services.DefaultMonitorConfig(),
		backups...,
	)

	return monitor.Run(ctx)
}

// createJunkScanService wires all junk scanner adapter implementations.
func createJunkScanService() *services.JunkScanService {
	return services.NewJunkScanService(
		docker.NewScanner(),
		apfs.NewScanner(),
		filesystem.NewTempScanner(),
		filesystem.NewCacheScanner(),
		filesystem.NewXcodeScanner(),
		filesystem.NewBrewScanner(),
	)
}
