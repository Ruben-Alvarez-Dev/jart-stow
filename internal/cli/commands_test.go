package cli

import (
	"bytes"
	"testing"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDeps creates minimal CLIDependencies for testing.
func testDeps() *CLIDependencies {
	qs := services.NewQuickExcludeService(services.NewScanService(0, nil))
	return &CLIDependencies{
		QuickExclude: qs,
	}
}

// testDepsFull creates CLIDependencies with all services wired (where possible).
func testDepsFull() *CLIDependencies {
	deps := testDeps()
	// Auditor and Reporter require repos that we can't easily mock without DB.
	// They will be nil, meaning inspect/audit/report commands won't be registered.
	return deps
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
	cmd := NewRootCommand(testDeps())
	assert.Equal(t, "jart-stow", cmd.Use)
	assert.Contains(t, cmd.Short, "macOS")
	assert.True(t, cmd.HasSubCommands())
}

func TestRootCommand_NoArgs(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd)
	assert.NoError(t, err)
	assert.Contains(t, out, "Usage:")
}

func TestRootCommand_HelpFlag(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "Available Commands:")
}

func TestRootCommand_VersionFlag(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "--version")
	assert.NoError(t, err)
	assert.Contains(t, out, "0.1.0-dev")
}

func TestRootCommand_Subcommands(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}

	// With minimal deps (no Auditor, Reporter, RuleRepo), only db-less commands appear
	expected := []string{"daemon", "scan", "status", "exclude"}
	for _, name := range expected {
		assert.True(t, names[name], "expected subcommand %q with minimal deps", name)
	}

	// These require DB-backed services
	assert.False(t, names["inspect"], "inspect requires Auditor service")
	assert.False(t, names["audit"], "audit requires Auditor service")
	assert.False(t, names["rule"], "rule requires RuleRepo")
	assert.False(t, names["report"], "report requires Reporter service")
}

func TestRootCommand_FullSubcommands(t *testing.T) {
	deps := testDeps()
	// Wire full services using an in-memory test setup
	// For this test, just verify the structure is correct with nil-safe wiring
	depsFull := &CLIDependencies{
		QuickExclude:  deps.QuickExclude,
		Auditor:       nil, // Would need real DB
		Reporter:      nil,
		RuleRepo:      nil,
		ProjectRepo:   nil,
		ExclusionRepo: nil,
		EventRepo:     nil,
	}
	cmd := NewRootCommand(depsFull)
	assert.True(t, cmd.HasSubCommands())
}

func TestScanCommand_Help(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "scan", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scan")
	assert.Contains(t, out, "node_modules")
}

func TestScanCommand_DefaultPath(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "scan")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scanning")
}

func TestScanCommand_CustomPath(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "scan", "/tmp")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scanning /tmp")
}

func TestStatusCommand(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "status")
	assert.NoError(t, err)
	assert.Contains(t, out, "Jart-Stow")
	assert.Contains(t, out, "daemon")
}

func TestInspectCommand(t *testing.T) {
	// With minimal deps, inspect is not registered
	cmd := NewRootCommand(testDeps())
	_, err := executeCommand(cmd, "inspect")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestAuditCommand(t *testing.T) {
	// With minimal deps, audit is not registered
	cmd := NewRootCommand(testDeps())
	_, err := executeCommand(cmd, "audit")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestReportCommand(t *testing.T) {
	// With minimal deps, report is not registered
	cmd := NewRootCommand(testDeps())
	_, err := executeCommand(cmd, "report")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestRuleCommand_NotRegistered(t *testing.T) {
	// With minimal deps, rule is not registered
	cmd := NewRootCommand(testDeps())
	_, err := executeCommand(cmd, "rule")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestDaemonCommand_Structure(t *testing.T) {
	cmd := NewRootCommand(testDeps())
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
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "daemon", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "manage the Jart-Stow background daemon")
	assert.Contains(t, out, "install")
	assert.Contains(t, out, "run")
}

func TestDaemonRun_Help(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "daemon", "run", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "foreground")
}

func TestDaemonStatus_Syntax(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	_, err := executeCommand(cmd, "daemon", "status")
	assert.NoError(t, err, "daemon status command should not error")
}

func TestDaemonLogs_Syntax(t *testing.T) {
	cmd := NewRootCommand(testDeps())
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
	cmd := NewRootCommand(testDeps())
	out, err := executeCommand(cmd, "exclude", "--help")
	assert.NoError(t, err)
	assert.Contains(t, out, "Scan volumes for development dependency folders")
}

func TestExcludeListCommand(t *testing.T) {
	cmd := NewRootCommand(testDeps())
	_, err := executeCommand(cmd, "exclude", "list")
	assert.NoError(t, err)
}

// Ensure domain is used
var _ = domain.ProjectStatusActive
