package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Ruben-Alvarez-Dev/jart-stow/internal/services"
	"github.com/spf13/cobra"
)

// NewExcludeCommand creates the "exclude" CLI command.
// It provides the same workflow as EXCLUSION-SCRIPT's tm-exclude.py on|off.
func NewExcludeCommand(quickService *services.QuickExcludeService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exclude",
		Short: "Manage backup exclusions for dev dependencies",
		Long: `Scan volumes for development dependency folders (node_modules, .venv,
target, vendor, build, dist, __pycache__, etc.) and add/remove them from
Time Machine and Carbon Copy Cloner exclusion lists.

This is the main tool for keeping dev junk out of your backups.`,
	}

	cmd.AddCommand(newExcludeOnCmd(quickService))
	cmd.AddCommand(newExcludeOffCmd(quickService))
	cmd.AddCommand(newExcludeListCmd(quickService))

	return cmd
}

func newExcludeOnCmd(qs *services.QuickExcludeService) *cobra.Command {
	return &cobra.Command{
		Use:   "on [path]",
		Short: "Scan and exclude dev dependencies from backups",
		Long: `Scan a path (volume, project, or current directory) for development
dependency folders and add them to backup exclusion lists.

If no path is given, shows an interactive volume selector.
If a path is given, scans that path directly.

Supports Time Machine (tmutil) and Carbon Copy Cloner (CCC).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExcludeOn(cmd, qs, args)
		},
	}
}

func newExcludeOffCmd(qs *services.QuickExcludeService) *cobra.Command {
	return &cobra.Command{
		Use:   "off [path]",
		Short: "Remove exclusions from backups",
		Long: `List current backup exclusions and remove selected ones.
If no path is given, shows all current exclusions interactively.
If a path is given, removes exclusions for that specific path.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExcludeOff(cmd, qs, args)
		},
	}
}

func newExcludeListCmd(qs *services.QuickExcludeService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backup exclusions (summary by default)",
		Long: `Show current backup exclusions.

By default shows a summary grouped by backup system with total counts and sizes.
Use --all to see the full detailed list (may be long).
Use --pager to see the full list through a pager (less).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExcludeList(cmd, qs)
		},
	}
	cmd.Flags().BoolP("all", "a", false, "Show full detailed list (auto-paginated if long)")
	return cmd
}

// ─── Interactive "on" workflow ────────────────────────────────────────────

func runExcludeOn(cmd *cobra.Command, qs *services.QuickExcludeService, args []string) error {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	var scanPath string

	if len(args) > 0 {
		scanPath = args[0]
	} else {
		// Interactive volume selector
		volumes := services.GetVolumes()
		if len(volumes) == 0 {
			return fmt.Errorf("no volumes found")
		}

		cmd.Println("📂 Volúmenes disponibles:")
		for i, v := range volumes {
			cmd.Printf("  [%d] %s\n", i+1, v.Name)
		}
		cmd.Println("  [q] Cancelar")

		cmd.Print("Selecciona volumen: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "q" || choice == "" {
			return nil
		}

		var idx int
		if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 || idx > len(volumes) {
			return fmt.Errorf("opción inválida")
		}

		scanPath = volumes[idx-1].Path
	}

	// Scan
	cmd.Printf("\n🔍 Escaneando %s...\n", scanPath)
	results, err := qs.Scan(ctx, scanPath)
	if err != nil {
		return fmt.Errorf("error escaneando: %w", err)
	}

	if len(results) == 0 {
		cmd.Println("\n✅ No se encontraron dependencias que excluir.")
		return nil
	}

	// Show results (paginated if many)
	{
		var b strings.Builder
		b.WriteString(fmt.Sprintf("\n📦 Encontradas %d carpetas:\n\n", len(results)))
		for i, r := range results {
			already := ""
			if r.AlreadyDone {
				already = " [ya excluido]"
			}
			b.WriteString(fmt.Sprintf("  [%d] %s  (%s, %s)%s\n", i+1,
				r.Path, r.PatternName, formatBytesQuick(r.SizeBytes), already))
		}
		b.WriteString("\n")
		printPaged(cmd, b.String())
	}

	// Select which to exclude
	cmd.Print("Selecciona para excluir (ej: 1,3,5 o 'todos'): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var selectedPaths []string
	if choice == "todos" || choice == "all" {
		for _, r := range results {
			if !r.AlreadyDone {
				selectedPaths = append(selectedPaths, r.Path)
			}
		}
	} else if choice == "" {
		return nil
	} else {
		parts := strings.Split(choice, ",")
		for _, p := range parts {
			var idx int
			if _, err := fmt.Sscanf(strings.TrimSpace(p), "%d", &idx); err == nil && idx >= 1 && idx <= len(results) {
				if !results[idx-1].AlreadyDone {
					selectedPaths = append(selectedPaths, results[idx-1].Path)
				}
			}
		}
	}

	if len(selectedPaths) == 0 {
		cmd.Println("Nada que excluir.")
		return nil
	}

	// Select backup system
	cmd.Println("\nSelecciona sistema de backup:")
	cmd.Println("  [1] Time Machine")
	cmd.Println("  [2] Carbon Copy Cloner")
	cmd.Println("  [3] Ambos")
	cmd.Print("Opción: ")
	toolChoice, _ := reader.ReadString('\n')
	toolChoice = strings.TrimSpace(toolChoice)

	// We always have both backup providers registered by default,
	// but we can filter by availability. For simplicity, we let
	// the user choose and then just try the selected ones.

	cmd.Println("")
	cmd.Println("🚀 Aplicando exclusiones...")

	// Check sudo
	cmd.Print("🔐 Se necesita sudo para tmutil. Solicitando permisos...\n")
	sudoCmd := exec.Command("sudo", "-v")
	if err := sudoCmd.Run(); err != nil {
		cmd.Println("⚠️  No se pudo obtener sudo. Time Machine se saltará.")
	}

	failures := qs.ExcludePaths(ctx, selectedPaths)
	successCount := 0

	for _, p := range selectedPaths {
		name := filepath.Base(p)
		if _, failed := failures[p]; failed {
			cmd.Printf("  ✗ %s: falló (%v)\n", name, failures[p])
		} else {
			cmd.Printf("  ✓ %s: excluido\n", name)
			successCount++
		}
	}

	cmd.Printf("\n✅ %d/%d exclusiones aplicadas.\n", successCount, len(selectedPaths))
	return nil
}

// ─── Interactive "off" workflow ────────────────────────────────────────────

func runExcludeOff(cmd *cobra.Command, qs *services.QuickExcludeService, args []string) error {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	if len(args) > 0 {
		// Remove specific path
		path := args[0]
		cmd.Printf("Quitando exclusión de %s...\n", path)

		sudoCmd := exec.Command("sudo", "-v")
		sudoCmd.Run()

		failures := qs.RemoveExclusions(ctx, []string{path})
		if len(failures) > 0 {
			for p, err := range failures {
				cmd.Printf("  ✗ %s: %v\n", p, err)
			}
			return fmt.Errorf("errores al quitar exclusiones")
		}
		cmd.Println("✅ Exclusión quitada.")
		return nil
	}

	// Interactive: list current exclusions
	cmd.Println("⚡ Exclusiones actuales:")

	exclusions, err := qs.ListExclusions(ctx)
	if err != nil {
		return fmt.Errorf("error listando exclusiones: %w", err)
	}

	var allPaths []string
	for systemName, paths := range exclusions {
		cmd.Printf("  ── %s ──\n", systemName)
		if len(paths) == 0 {
			cmd.Println("    (ninguna)")
		} else {
			for _, p := range paths {
				cmd.Printf("    • %s\n", p)
				allPaths = append(allPaths, p)
			}
		}
		cmd.Println()
	}

	if len(allPaths) == 0 {
		cmd.Println("No hay exclusiones activas.")
		return nil
	}

	// Remove specific exclusions using tmutil directly since ListExcluded
	// may use mdfind which can be incomplete. Try both approaches.
	if len(args) == 0 {
		cmd.Print("¿Qué deseas hacer?\n")
		cmd.Println("  [ruta completa]  - Quitar esa ruta específica")
		cmd.Println("  [q]             - Salir")
		cmd.Print("\nRuta a quitar (o q): ")

		pathChoice, _ := reader.ReadString('\n')
		pathChoice = strings.TrimSpace(pathChoice)

		if pathChoice == "q" || pathChoice == "" {
			return nil
		}

		sudoCmd := exec.Command("sudo", "-v")
		sudoCmd.Run()

		cmd.Printf("Quitando %s...\n", pathChoice)

		// Try tmutil directly first, then through the service
		exec.Command("sudo", "tmutil", "removeexclusion", pathChoice).Run()

		// Also try through CCC
		failures := qs.RemoveExclusions(ctx, []string{pathChoice})
		if len(failures) > 0 {
			// tmutil may have worked even if CCC didn't
			for p, err := range failures {
				if !strings.Contains(err.Error(), "tmutil") {
					cmd.Printf("  ⚠️  %s: %v\n", p, err)
				}
			}
		}
		cmd.Println("✅ Procesado.")
	}

	return nil
}

// ─── List workflow ────────────────────────────────────────────────────────

func runExcludeList(cmd *cobra.Command, qs *services.QuickExcludeService) error {
	ctx := context.Background()
	showAll, _ := cmd.Flags().GetBool("all")
	usePager, _ := cmd.Flags().GetBool("pager")

	exclusions, err := qs.ListExclusions(ctx)
	if err != nil {
		return fmt.Errorf("error listando exclusiones: %w", err)
	}

	// Count totals
	totalPaths := 0
	for _, paths := range exclusions {
		totalPaths += len(paths)
	}

	if totalPaths == 0 {
		cmd.Println("📋 No hay exclusiones activas en ningún sistema de backup.")
		return nil
	}

	// ── Summary mode (default) ──────────────────────────────────────────
	if !showAll && !usePager {
		cmd.Println("📋 Resumen de exclusiones de backups")
		cmd.Println()

		for systemName, paths := range exclusions {
			if len(paths) == 0 {
				continue
			}

			// Detect dev dependency patterns in the paths
			var depPaths, systemPaths, otherPaths []string
			depPatterns := []string{"node_modules", ".venv", "venv", "__pycache__", ".pytest_cache",
				"target", "vendor", "build", "dist", ".next", ".nuxt", ".cache", ".turbo"}

		outer:
			for _, p := range paths {
				for _, pat := range depPatterns {
					if strings.Contains(p, "/"+pat) || strings.Contains(p, "/"+pat+"/") {
						depPaths = append(depPaths, p)
						continue outer
					}
				}
				// System-managed exclusions (HTTPStorages, Chrome, etc.)
				if strings.Contains(p, "/Library/") ||
					strings.Contains(p, "/HTTPStorages/") ||
					strings.Contains(p, "/Chrome/") ||
					strings.Contains(p, "/Application Support/") {
					systemPaths = append(systemPaths, p)
				} else {
					otherPaths = append(otherPaths, p)
				}
			}

			// Calculate total size for dev dependencies
			var depSize int64
			for _, p := range depPaths {
				if info, err := os.Stat(p); err == nil {
					depSize += info.Size()
				}
			}

			systemLabel := systemName
			if systemLabel == "time_machine" {
				systemLabel = "⏰ Time Machine"
			} else if systemLabel == "carbon_copy_cloner" {
				systemLabel = "💿 Carbon Copy Cloner"
			}

			cmd.Printf("  %s — %d exclusiones totales\n", systemLabel, len(paths))
			if len(depPaths) > 0 {
				cmd.Printf("    🧩 Dependencias de desarrollo: %d (%s)\n", len(depPaths), formatBytesQuick(depSize))
			}
			if len(systemPaths) > 0 {
				cmd.Printf("    ⚙️  Exclusiones del sistema: %d\n", len(systemPaths))
			}
			if len(otherPaths) > 0 {
				cmd.Printf("    📦 Otras: %d\n", len(otherPaths))
			}
			cmd.Println()
		}

		cmd.Println("Usa 'jart-stow exclude list --all' para ver la lista completa.")
		return nil
	}

	// ── Full detail mode (always paginated if long) ────────────────────
	var output strings.Builder
	output.WriteString("📋 Exclusiones actuales de backups:\n\n")

	for systemName, paths := range exclusions {
		systemLabel := systemName
		if systemLabel == "time_machine" {
			systemLabel = "⏰ Time Machine"
		} else if systemLabel == "carbon_copy_cloner" {
			systemLabel = "💿 Carbon Copy Cloner"
		}
		output.WriteString(fmt.Sprintf("  ── %s (%d) ──\n", systemLabel, len(paths)))

		if len(paths) == 0 {
			output.WriteString("    (ninguna)\n")
		} else {
			for _, p := range paths {
				info, err := os.Stat(p)
				sizeStr := ""
				if err == nil {
					sizeStr = fmt.Sprintf(" (%s)", formatBytesQuick(info.Size()))
				}
				output.WriteString(fmt.Sprintf("    • %s%s\n", p, sizeStr))
			}
		}
		output.WriteString("\n")
	}

	printPaged(cmd, output.String())
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func formatBytesQuick(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	var i int
	for i = 0; i < len(units)-1 && size >= 1024; i++ {
		size /= 1024
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}
