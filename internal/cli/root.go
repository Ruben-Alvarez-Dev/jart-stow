// Package cli provides the command-line interface for jart-stow.
// Uses the standard library; Cobra integration will come when dependencies are available.
package cli

import (
	"fmt"
	"os"
)

const usage = `jart-stow — macOS development hygiene & backup exclusion manager

Usage:
  jart-stow [command]

Available Commands (coming in Phase 1):
  scan       Scan workspace roots for development artifacts
  status     Show current daemon and exclusion status
  daemon     Manage the background daemon (install, start, stop)
  inspect    Inspect a specific project
  audit      Audit all projects against hygiene rules
  rule       Manage hygiene rules
  report     View exclusion and cleanup reports
  api        Start the REST API server
  help       Show this help message

Run 'jart-stow help' for more information.
`

// Execute processes the CLI arguments and dispatches to the appropriate handler.
func Execute() error {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "version", "--version":
		fmt.Println("jart-stow version 0.1.0-dev")
		return nil
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
