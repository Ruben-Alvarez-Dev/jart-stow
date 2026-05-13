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

	tea "github.com/charmbracelet/bubbletea"
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

	// Create root command with quick exclude wired up
	cmd := cli.NewRootCommand(quickExcludeSvc)

	// Wire up the TUI subcommand
	cmd.AddCommand(cli.NewTUICommand(runTUI))

	// Override daemon run with real wiring
	if daemonCmd := findCobraSubcommand(cmd, "daemon"); daemonCmd != nil {
		if runCmd := findCobraSubcommand(daemonCmd, "run"); runCmd != nil {
			runCmd.RunE = runDaemon
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

// runTUI starts the Bubble Tea TUI with real data providers and action services wired to SQLite.
func runTUI(_ *cobra.Command, _ []string) error {
	dbPath := getDBPath()
	ctx := context.Background()

	conn, err := sqlite.NewConnection(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer conn.Close()

	repos := sqlite.NewRepositorySet(conn)

	// Create real services for interactive functionality
	scanService := services.NewScanService(0, nil)

	backups := []ports.BackupProvider{
		tmutil.NewAdapter(""),
		ccc.NewAdapter(""),
	}
	excludeService := services.NewExcludeService(
		repos.Projects, repos.Exclusions, scanService, backups...,
	)
	quickExcludeSvc := services.NewQuickExcludeService(scanService, backups...)
	quickExcludeImpl := tui.NewQuickExcludeImpl(quickExcludeSvc)
	junkService := createJunkScanService()

	// Create action providers
	scanEngine := tui.NewScanEngineImpl(scanService)
	junkRunner := tui.NewJunkScanRunnerImpl(junkService, repos.JunkItems)
	exclusionMgr := tui.NewExclusionManagerImpl(excludeService, repos.Exclusions)

	providers := tui.NewTUIProviders(
		repos.Projects, repos.Exclusions, repos.Events, repos.Rules,
		repos.WatchRoots, repos.JunkCategories, repos.JunkItems,
		scanEngine, junkRunner, exclusionMgr, quickExcludeImpl,
	)

	model := tui.NewMainModel(
		providers.Daemon, providers.WatchRoots, providers.Projects,
		providers.Exclusions, providers.Events, providers.Rules, providers.Junk,
		providers.ScanEngine, providers.JunkScanRunner, providers.ExclusionManager,
		providers.QuickExclude,
	)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
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
