package main

import (
	"fmt"
	"os"

	"github.com/0merUfuk/the-matrix/internal/matrixcfg"
	"github.com/0merUfuk/the-matrix/internal/oracle"
	"github.com/0merUfuk/the-matrix/internal/oracle/export"
	"github.com/spf13/cobra"
)

var version = "1.7.2-dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "oracle",
		Short:   "Automated knowledge collector — generates best-practice docs for any tech stack",
		Version: version,
	}

	rootCmd.AddCommand(researchCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(injectCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(updateCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// loadConfig loads .matrix.yaml from CWD. Returns a zero-value Config on error
// (after printing a warning), so callers can always use the result safely.
func loadConfig() matrixcfg.Config {
	cfg, err := matrixcfg.Load(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  warning: %v (using defaults)\n", err)
		return matrixcfg.Config{}
	}
	return cfg
}

func researchCmd() *cobra.Command {
	var dryRun bool
	var outputDir string
	var configFile string
	var goldStandardsDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "research",
		Short: "Wizard-guided knowledge collection: research cycles → human review gate → synthesis",
		Example: `  # Start a wizard-guided research session
  oracle research

  # Write output to a specific directory
  oracle research --output-dir ./oracle-output/go-rules

  # Non-interactive run using a pre-built StackContext JSON
  oracle research --config ./stack-context.json --output-dir ./oracle-output/go-rules

  # Preview generated plan without running research cycles
  oracle research --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			if !cmd.Flags().Changed("output-dir") {
				outputDir = matrixcfg.StringOr(cfg.Oracle.OutputDir, outputDir)
			}
			oracle.RunResearch(oracle.ResearchOpts{
				DryRun:           dryRun,
				Force:            force,
				OutputDir:        outputDir,
				ConfigFile:       configFile,
				GoldStandardsDir: goldStandardsDir,
			})
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be generated without writing files")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (default: /tmp/oracle-output/{stack-name})")
	cmd.Flags().StringVar(&configFile, "config", "", "Skip wizard and load StackContext from a JSON file (non-interactive)")
	cmd.Flags().StringVar(&goldStandardsDir, "gold-standards", "", "Path to gold standard docs for synthesis calibration")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing output directory without prompting")

	return cmd
}

func statusCmd() *cobra.Command {
	var knowledgeDir string
	var maxAgeMonths float64

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show loop progress, or check knowledge freshness with --knowledge",
		Example: `  # Show progress of the current research loop
  oracle status

  # Check freshness of injected knowledge docs
  oracle status --knowledge .claude/knowledge/go-rules

  # Use a custom staleness threshold (in months)
  oracle status --knowledge .claude/knowledge/go-rules --max-age-months 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			if !cmd.Flags().Changed("max-age-months") {
				maxAgeMonths = matrixcfg.Float64Or(cfg.Trinity.MaxAgeMonths, maxAgeMonths)
			}
			oracle.RunStatus(oracle.StatusOpts{
				KnowledgeDir: knowledgeDir,
				MaxAgeMonths: maxAgeMonths,
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&knowledgeDir, "knowledge", "", "Check freshness of injected docs in a .claude/knowledge/ directory")
	cmd.Flags().Float64Var(&maxAgeMonths, "max-age-months", 6, "Months before a doc is considered stale")

	return cmd
}

func injectCmd() *cobra.Command {
	var from, to string

	cmd := &cobra.Command{
		Use:   "inject",
		Short: "Validate oracle output docs and inject them into a .claude/knowledge/ directory",
		Example: `  # Inject oracle output into a project's knowledge directory
  oracle inject --from ./oracle-output/go-rules --to .claude/knowledge/go-rules`,
		Run: func(cmd *cobra.Command, args []string) {
			oracle.RunInject(oracle.InjectOpts{
				From: from,
				To:   to,
			})
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Source directory: oracle output (e.g. output/go-internal/)")
	cmd.Flags().StringVar(&to, "to", "", "Target directory: .claude/knowledge/ in your project")
	cmd.MarkFlagRequired("from")
	cmd.MarkFlagRequired("to")

	return cmd
}

func exportCmd() *cobra.Command {
	var from, to, format, language string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export oracle knowledge docs to other AI coding tool formats (Cursor, Copilot, AGENTS.md, Windsurf)",
		Example: `  # Export knowledge as AGENTS.md into the current project
  oracle export --from .claude/knowledge/go-rules --format agents-md

  # Export to Cursor rules format in a specific project
  oracle export --from .claude/knowledge/go-rules --format cursor --to ~/repos/my-service

  # Emit every supported format at once
  oracle export --from .claude/knowledge/go-rules --format all

  # Override language detection
  oracle export --from ./oracle-output/custom --format copilot --language go`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return export.RunExport(export.RunExportOpts{
				From:     from,
				Format:   format,
				To:       to,
				Language: language,
			})
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Source directory: oracle output or .claude/knowledge/ dir")
	cmd.Flags().StringVar(&format, "format", "", "Target format: agents-md, cursor, copilot, windsurf, all")
	cmd.Flags().StringVar(&to, "to", ".", "Target project directory (default: current directory)")
	cmd.Flags().StringVar(&language, "language", "", "Override language detection (go, javascript, python, dart, etc.)")
	cmd.MarkFlagRequired("from")
	cmd.MarkFlagRequired("format")

	return cmd
}

func updateCmd() *cobra.Command {
	var workspace, knowledge, topicsRaw string
	var maxAgeMonths int
	var dryRun, autoApprove bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Incremental knowledge refresh — re-research only stale topics, then inject",
		Long: `Re-research and re-synthesize specific stale topics from an existing oracle workspace.

Two usage modes:

  1. Auto-discover stale topics from the knowledge directory:
     oracle update --workspace ~/oracle-go/ --knowledge .claude/knowledge/go-rules/

  2. Explicit topic list:
     oracle update --workspace ~/oracle-go/ --topics 04-error-handling,11-redis-caching --knowledge .claude/knowledge/go-rules/

The update creates a sub-workspace under {workspace}/updates/update-{date}/, runs
focused research cycles (2 by default), skips the human review gate, synthesizes
only the specified topics, and injects the results into the knowledge directory.`,
		Example: `  # Auto-discover stale topics and refresh them
  oracle update --workspace ~/oracle-go --knowledge .claude/knowledge/go-rules

  # Refresh a specific list of topics
  oracle update --workspace ~/oracle-go --knowledge .claude/knowledge/go-rules \
      --topics 04-error-handling,11-redis-caching

  # Preview stale topics without running research
  oracle update --workspace ~/oracle-go --knowledge .claude/knowledge/go-rules --dry-run

  # Non-interactive run for CI
  oracle update --workspace ~/oracle-go --knowledge .claude/knowledge/go-rules --auto-approve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			if !cmd.Flags().Changed("max-age-months") {
				maxAgeMonths = int(matrixcfg.Float64Or(cfg.Trinity.MaxAgeMonths, float64(maxAgeMonths)))
			}
			topics, err := oracle.ParseTopicsFlag(topicsRaw)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			oracle.RunUpdate(oracle.UpdateOpts{
				WorkspaceDir: workspace,
				KnowledgeDir: knowledge,
				Topics:       topics,
				MaxAgeMonths: maxAgeMonths,
				DryRun:       dryRun,
				AutoApprove:  autoApprove,
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Path to an existing oracle research workspace directory")
	cmd.Flags().StringVar(&knowledge, "knowledge", "", "Path to the target .claude/knowledge/ directory")
	cmd.Flags().StringVar(&topicsRaw, "topics", "", "Comma-separated topic file prefixes (e.g. 04-error-handling,11-redis-caching)")
	cmd.Flags().IntVar(&maxAgeMonths, "max-age-months", 6, "Staleness threshold for auto-discovery mode (months)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print stale topics and planned actions without running research")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt before running")
	cmd.MarkFlagRequired("workspace")
	cmd.MarkFlagRequired("knowledge")

	return cmd
}
