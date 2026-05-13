package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// printPaged prints content to stdout. If the content is longer than the
// terminal height, it pipes through less -R -F -X so the user can scroll
// without overflowing the viewport.
//
// less flags:
//   -R  preserve ANSI colors
//   -F  exit immediately if content fits one screen
//   -X  don't clear screen on exit (keeps content visible after quit)
func printPaged(cmd *cobra.Command, content string) {
	lines := strings.Split(content, "\n")
	_, termHeight := terminalSize()

	// If content fits on screen, print directly
	if len(lines) <= termHeight || termHeight <= 0 {
		cmd.Print(content)
		return
	}

	// Pipe through less
	pager := exec.Command("less", "-R", "-F", "-X")
	pager.Stdin = strings.NewReader(content)
	pager.Stdout = os.Stdout
	pager.Stderr = os.Stderr

	if err := pager.Run(); err != nil {
		// Fallback: print directly if less fails
		cmd.Print(content)
	}
}

// printPagedLines is a convenience wrapper that joins lines and calls printPaged.
func printPagedLines(cmd *cobra.Command, lines []string) {
	printPaged(cmd, strings.Join(lines, "\n"))
}

// terminalSize returns the terminal width and height.
// Falls back to 80x24 on error.
func terminalSize() (width, height int) {
	width = 80
	height = 24

	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return
	}
	_, err = fmt.Sscanf(string(out), "%d %d", &height, &width)
	if err != nil {
		width = 80
		height = 24
	}
	return
}
