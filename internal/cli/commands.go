package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/domain"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/ports"
	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/spf13/cobra"
)

func newInspectCommand(auditor *services.AuditService) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect [path]",
		Short: "Inspect a project's exclusion and hygiene status",
		Long: `Display detailed information about a project: its path, status,
active exclusions, artifact folders, total space saved, and any issues.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			// Verify the path exists
			if info, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("path %s: %w", absPath, err)
			} else if !info.IsDir() {
				return fmt.Errorf("path %s is not a directory", absPath)
			}

			inspection, err := auditor.InspectProject(ctx, absPath)
			if err != nil {
				return fmt.Errorf("inspecting %s: %w", absPath, err)
			}

			printPaged(cmd, services.FormatInspection(inspection))
			return nil
		},
	}
}

func newAuditCommand(auditor *services.AuditService) *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Audit backup exclusions for consistency",
		Long: `Check every active database exclusion against the actual backup system
state (Time Machine and Carbon Copy Cloner). Reports any exclusions
that are recorded in the database but missing from the backup system.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			cmd.PrintErrln("Running exclusion consistency audit...")
			summary, err := auditor.VerifyExclusions(ctx)
			if err != nil {
				return fmt.Errorf("running audit: %w", err)
			}

			printPaged(cmd, services.FormatAuditSummary(summary))
			return nil
		},
	}
}

func newRuleCommand(ruleRepo ports.RuleRepository) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage custom hygiene rules",
		Long: `List, add, and remove hygiene rules that control which development
artifact patterns are detected and how they are handled.`,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all custom rules",
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := context.Background()
				rules, err := ruleRepo.FindAll(ctx)
				if err != nil {
					return fmt.Errorf("listing rules: %w", err)
				}

				if len(rules) == 0 {
					cmd.Println("No rules configured.")
					return nil
				}

				cmd.Println("Global Rules:")
				globalFound := false
				for _, r := range rules {
					if r.ProjectID != nil {
						continue
					}
					globalFound = true
					cmd.Printf("  %-25s max=%d action=%-7s priority=%d enabled=%v\n",
						r.Pattern, r.MaxSizeBytes, r.Action, r.Priority, r.Enabled)
				}
				if !globalFound {
					cmd.Println("  (none)")
				}

				cmd.Println("\nProject Rules:")
				projectFound := false
				for _, r := range rules {
					if r.ProjectID == nil {
						continue
					}
					projectFound = true
					cmd.Printf("  project=%d %-25s max=%d action=%-7s priority=%d enabled=%v\n",
						*r.ProjectID, r.Pattern, r.MaxSizeBytes, r.Action, r.Priority, r.Enabled)
				}
				if !projectFound {
					cmd.Println("  (none)")
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "add <pattern> <action>",
			Short: "Add a custom rule",
			Long: `Add a new hygiene rule. Pattern is a directory name (e.g., node_modules).
Action is one of: warn, alert, exclude, clean.`,
			Args: cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := context.Background()
				pattern := args[0]
				action := domain.RuleAction(args[1])

				if action != domain.RuleActionWarn &&
					action != domain.RuleActionAlert &&
					action != domain.RuleActionExclude &&
					action != domain.RuleActionClean {
					return fmt.Errorf("invalid action %q: must be warn, alert, exclude, or clean", action)
				}

				rule, err := ruleRepo.Save(ctx, &domain.Rule{
					Pattern:      pattern,
					MaxSizeBytes: 0,
					Action:       action,
					Priority:     10,
					Enabled:      true,
				})
				if err != nil {
					return fmt.Errorf("adding rule: %w", err)
				}
				cmd.Printf("Rule added: #%d %s -> %s\n", rule.ID, pattern, action)
				return nil
			},
		},
		&cobra.Command{
			Use:   "remove <id>",
			Short: "Remove a custom rule by ID",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := context.Background()
				var id int64
				if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
					return fmt.Errorf("invalid rule ID %q: must be a number", args[0])
				}

				if err := ruleRepo.Delete(ctx, id); err != nil {
					return fmt.Errorf("removing rule #%d: %w", id, err)
				}
				cmd.Printf("Rule #%d removed.\n", id)
				return nil
			},
		},
	)

	return cmd
}

func newReportCommand(reporter *services.ReportService) *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Generate a hygiene and exclusion report",
		Long: `Generate a comprehensive report showing project counts, exclusion
statistics, breakdowns by pattern and backup system, and activity history.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			summary, err := reporter.GenerateSummary(ctx)
			if err != nil {
				return fmt.Errorf("generating report: %w", err)
			}

			history, err := reporter.GenerateHistory(ctx, 30)
			if err != nil {
				return fmt.Errorf("generating history: %w", err)
			}

			report := services.FormatSummary(summary) + "\n" + services.FormatHistory(history)
			printPaged(cmd, report)
			return nil
		},
	}
}
