package main

import (
	"fmt"
	"os"

	"github.com/0merUfuk/the-matrix/internal/neo"
	"github.com/0merUfuk/the-matrix/internal/registry"
	"github.com/spf13/cobra"
)

var version = "1.7.2-dev"

func init() {
	registry.ClientVersion = version
}

func main() {
	rootCmd := &cobra.Command{
		Use:     "neo",
		Short:   "Meta-CLI orchestrator — provisions and maintains Claude Code agent ecosystems",
		Version: version,
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(analyzeCmd())
	rootCmd.AddCommand(doctorCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(registryCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	var dryRun bool
	var path string
	var preset string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Provision a complete Claude Code agent ecosystem",
		Example: `  # Provision a new .claude/ ecosystem interactively (wizard)
  neo init

  # Preview what would be generated, without writing files
  neo init --dry-run

  # Non-interactive run for CI using a preset
  neo init --path . --preset go-service`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunInit(neo.InitOpts{
				DryRun: dryRun,
				Path:   path,
				Preset: preset,
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without writing")
	cmd.Flags().StringVar(&path, "path", "", "Project directory (default: wizard-selected)")
	cmd.Flags().StringVar(&preset, "preset", "", "Pre-configured project type (internal-go-service, go-service, nextjs-solo)")

	return cmd
}

func analyzeCmd() *cobra.Command {
	var path string
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze an existing codebase and generate a Claude Code ecosystem",
		Long: `Analyze scans an existing project directory, detects the language, framework,
architecture pattern, testing framework, and build commands, then generates a
tailored .claude/ agent ecosystem — no wizard or oracle research needed.`,
		Example: `  # Auto-detect the current directory and generate .claude/
  neo analyze

  # Analyze a specific project path
  neo analyze --path ~/repos/my-service

  # Preview the generated ecosystem without writing files
  neo analyze --path ~/repos/my-service --dry-run

  # Overwrite existing .claude/ without prompting
  neo analyze --force`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunAnalyze(neo.AnalyzeOpts{
				Path:   path,
				DryRun: dryRun,
				Force:  force,
			})
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Path to the project to analyze (default: current directory)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without writing")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing .claude/ without prompting")

	return cmd
}

func doctorCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate ecosystem integrity",
		Example: `  # Check the current project's .claude/ ecosystem
  neo doctor

  # Check a specific project directory
  neo doctor --path ~/repos/my-service`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunDoctor(path)
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Project directory (default: cwd)")

	return cmd
}

func statusCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Dashboard view of ecosystem state",
		Example: `  # Show ecosystem dashboard for the current project
  neo status

  # Check a specific project
  neo status --path ~/repos/my-service`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunStatus(path)
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Project directory (default: cwd)")

	return cmd
}

func updateCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for stale knowledge and guide refresh",
		Example: `  # Detect stale docs in the current project and print refresh commands
  neo update

  # Check a specific project directory
  neo update --path ~/repos/my-service`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunUpdate(path)
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Project directory (default: cwd)")

	return cmd
}

func registryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Browse and pull pre-researched knowledge packs",
	}

	cmd.AddCommand(registryListCmd())
	cmd.AddCommand(registrySearchCmd())
	cmd.AddCommand(registryInfoCmd())
	cmd.AddCommand(registryPullCmd())

	return cmd
}

func registryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available knowledge packs",
		Example: `  # List every pack available in the knowledge registry
  neo registry list`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunRegistryList()
		},
	}
}

func registrySearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search packs by name, language, or framework",
		Args:  cobra.ExactArgs(1),
		Example: `  # Find packs for the Go language
  neo registry search go

  # Find packs matching a framework keyword
  neo registry search nextjs`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunRegistrySearch(args[0])
		},
	}
}

func registryInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <pack-name>",
		Short: "Show detailed information about a pack",
		Args:  cobra.ExactArgs(1),
		Example: `  # Show metadata and topic list for a specific pack
  neo registry info go-rules`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunRegistryInfo(args[0])
		},
	}
}

func registryPullCmd() *cobra.Command {
	var toDir string

	cmd := &cobra.Command{
		Use:   "pull <pack-name>",
		Short: "Download a knowledge pack to a directory",
		Args:  cobra.ExactArgs(1),
		Example: `  # Download a pack into a project's .claude/knowledge/ directory
  neo registry pull go-rules --to .claude/knowledge/go-rules`,
		Run: func(cmd *cobra.Command, args []string) {
			neo.RunRegistryPull(args[0], toDir)
		},
	}

	cmd.Flags().StringVar(&toDir, "to", "", "Target directory for knowledge docs (required)")
	cmd.MarkFlagRequired("to")

	return cmd
}
