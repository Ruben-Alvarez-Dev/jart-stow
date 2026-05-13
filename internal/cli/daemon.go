package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	launchdLabel     = "dev.rubenalvarez.jart-stow"
	launchdPlistPath = "~/Library/LaunchAgents/" + launchdLabel + ".plist"

	plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>JART_STOW_DB_PATH</key>
        <string>%s</string>
        <key>JART_STOW_LOG_LEVEL</key>
        <string>info</string>
    </dict>
</dict>
</plist>`
)

func newDaemonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the Jart-Stow background daemon",
		Long:  `Install, start, stop, and manage the Jart-Stow background daemon via launchd.`,
	}

	cmd.AddCommand(
		newDaemonInstallCmd(),
		newDaemonUninstallCmd(),
		newDaemonStartCmd(),
		newDaemonStopCmd(),
		newDaemonRestartCmd(),
		newDaemonStatusCmd(),
		newDaemonLogsCmd(),
		newDaemonRunCmd(),
	)

	return cmd
}

func newDaemonInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the daemon as a launchd user agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("finding home directory: %w", err)
			}

			plistPath := filepath.Join(home, "Library/LaunchAgents", launchdLabel+".plist")
			dataDir := filepath.Join(home, ".local", "share", "jart-stow")
			dbPath := filepath.Join(dataDir, "jart-stow.db")
			logPath := filepath.Join(dataDir, "daemon.log")

			// Ensure data directory exists
			if err := os.MkdirAll(dataDir, 0o755); err != nil {
				return fmt.Errorf("creating data directory: %w", err)
			}

			// Find the jart-stow binary
			binaryPath, err := exec.LookPath("jart-stow")
			if err != nil {
				binaryPath, err = os.Executable()
				if err != nil {
					return fmt.Errorf("cannot find jart-stow binary: %w", err)
				}
			}

			plistContent := fmt.Sprintf(plistTemplate,
				launchdLabel,
				binaryPath,
				logPath, logPath,
				dbPath,
			)

			if err := os.WriteFile(plistPath, []byte(plistContent), 0o644); err != nil {
				return fmt.Errorf("writing plist: %w", err)
			}

			fmt.Printf("Daemon installed at %s\n", plistPath)
			fmt.Printf("Data directory: %s\n", dataDir)

			// Load into launchd
			loadCmd := exec.Command("launchctl", "load", plistPath)
			if output, err := loadCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("loading daemon: %s: %w", string(output), err)
			}

			fmt.Println("Daemon loaded into launchd and will start at login.")
			return nil
		},
	}
}

func newDaemonUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the daemon from launchd",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("finding home directory: %w", err)
			}

			plistPath := filepath.Join(home, "Library/LaunchAgents", launchdLabel+".plist")

			// Unload from launchd
			unloadCmd := exec.Command("launchctl", "unload", plistPath)
			unloadCmd.Run() // Ignore error if not loaded

			// Remove plist
			if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing plist: %w", err)
			}

			fmt.Println("Daemon uninstalled.")
			return nil
		},
	}
}

func newDaemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the daemon via launchd",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLaunchctl("load")
		},
	}
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon via launchd",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLaunchctl("unload")
		},
	}
}

func newDaemonRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runLaunchctl("unload"); err != nil {
				// It may not be running, that's fine
				fmt.Println("Daemon was not running.")
			}
			return runLaunchctl("load")
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check if the daemon is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			listCmd := exec.Command("launchctl", "list", launchdLabel)
			output, err := listCmd.CombinedOutput()
			if err != nil {
				fmt.Println("Daemon is not running.")
				return nil
			}
			fmt.Printf("Daemon is running:\n%s", string(output))
			return nil
		},
	}
}

func newDaemonLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Show recent daemon logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("finding home directory: %w", err)
			}
			logPath := filepath.Join(home, ".local", "share", "jart-stow", "daemon.log")

			data, err := os.ReadFile(logPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No daemon log file found.")
					return nil
				}
				return fmt.Errorf("reading log: %w", err)
			}

			fmt.Print(string(data))
			return nil
		},
	}
}

func newDaemonRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the daemon in the foreground (for debugging)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("daemon run is wired in main.go — use the compiled binary")
		},
	}
}

func runLaunchctl(action string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}

	plistPath := filepath.Join(home, "Library/LaunchAgents", launchdLabel+".plist")
	cmd := exec.Command("launchctl", action, plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl %s: %s: %w", action, string(output), err)
	}

	fmt.Printf("Daemon %sed.\n", action)
	return nil
}
