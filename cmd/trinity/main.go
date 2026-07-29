package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/matrixcfg"
	"github.com/0merUfuk/the-matrix/internal/trinity"
	"github.com/spf13/cobra"
)

var version = "1.7.2-dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "trinity",
		Short:   "Maintenance runtime — keeps Claude Code agent ecosystems alive and fresh",
		Version: version,
	}

	rootCmd.AddCommand(healthCmd())
	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(refreshCmd())
	rootCmd.AddCommand(selfStateCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig loads .matrix.yaml from the given directory (the --path flag value).
// Falls back to CWD when pathFlag is empty. Returns a zero-value Config on error
// (after printing a warning), so callers can always use the result safely.
func loadConfig(pathFlag string) matrixcfg.Config {
	dir := pathFlag
	if dir == "" {
		dir = "."
	}
	cfg, err := matrixcfg.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  warning: %v (using defaults)\n", err)
		return matrixcfg.Config{}
	}
	return cfg
}

func healthCmd() *cobra.Command {
	var projectDir string
	var maxAgeMonths float64

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Validate ecosystem integrity (read-only)",
		Example: `  # Run health check against the current project
  trinity health

  # Check a specific project directory
  trinity health --path ~/repos/my-service

  # Use a stricter staleness threshold
  trinity health --max-age-months 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig(projectDir)
			if !cmd.Flags().Changed("max-age-months") {
				maxAgeMonths = matrixcfg.Float64Or(cfg.Trinity.MaxAgeMonths, maxAgeMonths)
			}
			trinity.RunHealth(trinity.HealthOpts{
				ProjectDir:   projectDir,
				MaxAgeMonths: maxAgeMonths,
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "path", "", "Project directory to check (default: cwd)")
	cmd.Flags().Float64Var(&maxAgeMonths, "max-age-months", 6, "Staleness threshold in months")

	return cmd
}

func syncCmd() *cobra.Command {
	var from, to, projectDir string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Push oracle output to .claude/knowledge/",
		Example: `  # Sync using flags
  trinity sync --from ./oracle-output/go-rules --to .claude/knowledge/go-rules

  # Sync using .trinity.json configuration in the current project
  trinity sync

  # Preview without writing any files
  trinity sync --dry-run

  # Target a different project directory
  trinity sync --path ~/repos/my-service`,
		Run: func(cmd *cobra.Command, args []string) {
			// Load config from target project directory for future config consumption.
			_ = loadConfig(projectDir)
			trinity.RunSync(trinity.SyncOpts{
				From:       from,
				To:         to,
				ProjectDir: projectDir,
				DryRun:     dryRun,
			})
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Source directory (oracle output)")
	cmd.Flags().StringVar(&to, "to", "", "Target directory (.claude/knowledge/...)")
	cmd.Flags().StringVar(&projectDir, "path", "", "Project root for .trinity.json and log (default: cwd)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be synced without writing")

	return cmd
}

func refreshCmd() *cobra.Command {
	var projectDir string
	var maxAgeMonths float64
	var staleOnly bool
	var topics string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Detect stale knowledge docs and guide refresh",
		Example: `  # Detect stale docs and print the suggested refresh workflow
  trinity refresh

  # Show only topics above the staleness threshold
  trinity refresh --stale-only

  # Check specific topic slugs
  trinity refresh --topics error-handling,redis-caching

  # Preview without taking action
  trinity refresh --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig(projectDir)
			if !cmd.Flags().Changed("max-age-months") {
				maxAgeMonths = matrixcfg.Float64Or(cfg.Trinity.MaxAgeMonths, maxAgeMonths)
			}

			var topicList []string
			if topics != "" {
				for _, t := range strings.Split(topics, ",") {
					topicList = append(topicList, strings.TrimSpace(t))
				}
			}

			trinity.RunRefresh(trinity.RefreshOpts{
				ProjectDir:   projectDir,
				MaxAgeMonths: maxAgeMonths,
				Topics:       topicList,
				DryRun:       dryRun,
				StaleOnly:    staleOnly,
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "path", "", "Project root (default: cwd)")
	cmd.Flags().Float64Var(&maxAgeMonths, "max-age-months", 6, "Staleness threshold in months")
	cmd.Flags().BoolVar(&staleOnly, "stale-only", false, "Only show topics above age threshold")
	cmd.Flags().StringVar(&topics, "topics", "", "Comma-separated topic slugs to check")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be refreshed without action")

	return cmd
}

func selfStateCmd() *cobra.Command {
	var projectDir, outPath string
	var jsonOnly, prune bool

	cmd := &cobra.Command{
		Use:   "self-state",
		Short: "Produce a state.v1 JSON snapshot of the-matrix ecosystem (deterministic)",
		Long: `self-state collects deterministic ecosystem inputs (git state, tool versions,
agent-memory counts, backlog) and emits a state.v1 JSON snapshot at
.claude/state/state-{ts}.json with a state-latest.json pointer (symlink with
copy fallback). Prose blocks (context_drift, ecosystem_health, security,
dispatch_signals) are explicit JSON null in this quick mode — they are
populated by the /self-state skill's full mode via subagent fan-out.`,
		Example: `  # Default: write a snapshot under .claude/state/ and update state-latest.json
  trinity self-state

  # Quick mode: emit JSON to stdout, no persistence
  trinity self-state --json

  # Custom output path (skips state-latest.json update)
  trinity self-state --out /tmp/state.json

  # Write a snapshot then enforce 12-snapshot retention
  trinity self-state --prune`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return trinity.RunSelfState(trinity.SelfStateOpts{
				ProjectDir: projectDir,
				JSONOnly:   jsonOnly,
				OutPath:    outPath,
				Prune:      prune,
			})
		},
	}

	cmd.Flags().StringVar(&projectDir, "path", "", "Project root (default: cwd)")
	cmd.Flags().BoolVar(&jsonOnly, "json", false, "Emit JSON to stdout instead of writing a snapshot file")
	cmd.Flags().StringVar(&outPath, "out", "", "Custom output path (skips state-latest.json update)")
	cmd.Flags().BoolVar(&prune, "prune", false, "After writing, retain only the 12 most recent snapshots")

	return cmd
}
