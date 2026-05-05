// Package main is the entry point for the jart-stow binary.
// It wires all dependencies and starts the application.
package main

import (
	"fmt"
	"os"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
