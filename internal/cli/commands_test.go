package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testQuickService creates a minimal QuickExcludeService for testing.
func testQuickService() *services.QuickExcludeService {
	return services.NewQuickExcludeService(services.NewScanService(0, nil))
}

// executeCommand runs a cobra command with args and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	assert.Equal(t, "jart-stow", cmd.Use)
	assert.Contains(t, cmd.Short, "macOS")
	assert.True(t, cmd.HasSubCommands())
}

func TestRootCommand_NoArgs(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd)
	assert.NoError(t, err)
	assert.Contains(t, out, "Usage:")
}

func TestRootCommand_HelpFlag(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "Available Commands:")
}

func TestRootCommand_VersionFlag(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "--version")
	assert.NoError(t, err)
	assert.Contains(t, out, "0.1.0-dev")
}

func TestRootCommand_Subcommands(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}

	expected := []string{"daemon", "scan", "status", "inspect", "audit", "rule", "report", "exclude"}
	for _, name := range expected {
		assert.True(t, names[name], "expected subcommand %q", name)
	}
}

func TestScanCommand_Help(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "scan", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scan")
	assert.Contains(t, out, "node_modules")
}

func TestScanCommand_DefaultPath(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "scan")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scanning")
}

func TestScanCommand_CustomPath(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "scan", "/tmp")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scanning /tmp")
}

func TestStatusCommand(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "status")
	assert.NoError(t, err)
	assert.Contains(t, out, "Jart-Stow")
	assert.Contains(t, out, "Daemon")
}

func TestInspectCommand(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "inspect")
	assert.NoError(t, err)
	assert.Contains(t, out, "Inspecting")
}

func TestInspectCommand_WithPath(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "inspect", "/some/project")
	assert.NoError(t, err)
	assert.Contains(t, out, "Inspecting /some/project")
}

func TestAuditCommand(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "audit")
	assert.NoError(t, err)
	assert.Contains(t, out, "Auditing")
}

func TestReportCommand(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "report")
	assert.NoError(t, err)
	assert.Contains(t, out, "report", "output should mention report")
}

func TestRuleCommand_List(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "rule", "list")
	assert.NoError(t, err)
	assert.Contains(t, out, "No rules configured")
}

func TestRuleCommand_Add(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "rule", "add", "*.log", "skip")
	assert.NoError(t, err)
	assert.Contains(t, out, "Rule added")
	assert.Contains(t, out, "*.log")
}

func TestRuleCommand_AddMissingArgs(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	_, err := executeCommand(cmd, "rule", "add", "*.log")
	assert.Error(t, err, "should error when missing pattern")
}

func TestRuleCommand_Remove(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "rule", "remove", "*.log")
	assert.NoError(t, err)
	assert.Contains(t, out, "Rule removed")
}

func TestNewTUICommand(t *testing.T) {
	called := false
	runner := func(cmd *cobra.Command, args []string) error {
		called = true
		return nil
	}
	cmd := NewTUICommand(runner)
	require.Equal(t, "tui", cmd.Use)
	require.Equal(t, "Launch the terminal user interface", cmd.Short)

	out, err := executeCommand(cmd)
	assert.NoError(t, err)
	assert.True(t, called, "runner should have been called")
	assert.Empty(t, strings.TrimSpace(out), "no output expected from stub runner")
}

func TestDaemonCommand_Structure(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	var daemon *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "daemon" {
			daemon = c
			break
		}
	}
	require.NotNil(t, daemon, "should find daemon subcommand")
	assert.Equal(t, "Manage the Jart-Stow background daemon", daemon.Short)

	expectedSubs := []string{"install", "uninstall", "start", "stop", "restart", "status", "logs", "run"}
	names := map[string]bool{}
	for _, c := range daemon.Commands() {
		names[c.Name()] = true
	}
	for _, name := range expectedSubs {
		assert.True(t, names[name], "daemon should have subcommand %q", name)
	}
}

func TestDaemonCommand_Help(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "daemon", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "manage the Jart-Stow background daemon")
	assert.Contains(t, out, "install")
	assert.Contains(t, out, "run")
}

func TestDaemonRun_Help(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "daemon", "run", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "foreground")
}

func TestDaemonStatus_Syntax(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	_, err := executeCommand(cmd, "daemon", "status")
	assert.NoError(t, err, "daemon status command should not error")
}

func TestDaemonLogs_Syntax(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	_, err := executeCommand(cmd, "daemon", "logs")
	assert.NoError(t, err, "daemon logs command should not error")
}

func TestPlistTemplate(t *testing.T) {
	assert.Contains(t, plistTemplate, "<key>Label</key>")
	assert.Contains(t, plistTemplate, "<key>ProgramArguments</key>")
	assert.Contains(t, plistTemplate, "<key>RunAtLoad</key>")
	assert.Contains(t, plistTemplate, "<key>KeepAlive</key>")
	assert.Contains(t, plistTemplate, "<key>StandardOutPath</key>")
	assert.Contains(t, plistTemplate, "<key>EnvironmentVariables</key>")
	assert.Contains(t, plistTemplate, "JART_STOW_DB_PATH")
	assert.Contains(t, plistTemplate, "JART_STOW_LOG_LEVEL")
}

func TestLaunchdLabel(t *testing.T) {
	assert.Equal(t, "dev.rubenalvarez.jart-stow", launchdLabel)
}

func TestExcludeCommand_Help(t *testing.T) {
	cmd := NewRootCommand(testQuickService())
	out, err := executeCommand(cmd, "exclude", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scan volumes for development dependency folders")
}

func TestExcludeListCommand(t *testing.T) {
	qs := testQuickService()
	cmd := NewRootCommand(qs)
	out, err := executeCommand(cmd, "exclude", "list")
	assert.NoError(t, err)
	// Should show that there are no exclusions (no backup providers in test)
	assert.Contains(t, out, "No hay exclusiones")
}
