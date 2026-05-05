package cli

import "github.com/spf13/cobra"

// TUIRunner is a function that starts the Bubble Tea TUI.
type TUIRunner func(cmd *cobra.Command, args []string) error

// NewTUICommand creates the TUI subcommand for launching the terminal interface.
func NewTUICommand(runner TUIRunner) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the terminal user interface",
		Long:  `Start the interactive Bubble Tea TUI for managing projects, exclusions, rules, and junk items.`,
		RunE:  runner,
	}
}
