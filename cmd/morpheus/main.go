package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/0merUfuk/the-matrix/internal/morpheus"
)

var version = "1.7.2-dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "morpheus",
		Short:   "Autonomous development infrastructure generator",
		Version: version,
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(loopCmd())
	rootCmd.AddCommand(finalizeCmd())
	rootCmd.AddCommand(doctorCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(reportCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	var dryRun bool
	var outputDir string
	var contextPath string
	var serviceName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a new service with complete .claude/ + .autonomous/ setup",
		Example: `  # Wizard-guided scaffold in the current directory
  morpheus init

  # Scaffold into a specific output directory
  morpheus init --output-dir ./billing-api

  # Context-driven init from a neo-generated project profile
  morpheus init --context .claude/project-context.json --service-name billing-api

  # Preview what would be generated without writing
  morpheus init --dry-run`,
		Run: func(cmd *cobra.Command, args []string) {
			morpheus.RunInit(morpheus.InitOpts{
				DryRun:      dryRun,
				OutputDir:   outputDir,
				ContextPath: contextPath,
				ServiceName: serviceName,
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be generated without writing files")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory (defaults to ./{service-name})")
	cmd.Flags().StringVar(&contextPath, "context", "", "Path to neo project profile (skips project type and tech stack questions)")
	cmd.Flags().StringVar(&serviceName, "service-name", "", "Override service name (used when context is provided by neo)")

	return cmd
}

func loopCmd() *cobra.Command {
	var dryRun bool
	var path string
	var nonInteractive bool
	var goal string
	var maxCycles int
	var devTurns int
	var testTurns int
	var reviewTurns int

	cmd := &cobra.Command{
		Use:   "loop",
		Short: "Set up autonomous development loop on an existing project",
		Long:  "Generates .autonomous/ + tasks/ for any project with a CLAUDE.md. Does NOT scaffold service code.",
		Example: `  # Wizard-guided loop setup in the current project
  morpheus loop

  # Set up a loop in a specific project directory
  morpheus loop --path ~/repos/my-service

  # Fully non-interactive run (for CI / automation)
  morpheus loop --non-interactive --goal "Implement user auth" --max-cycles 10

  # Tune per-agent turn budgets
  morpheus loop --non-interactive --goal "Add billing" --dev-turns 40 --test-turns 25 --review-turns 15`,
		Run: func(cmd *cobra.Command, args []string) {
			morpheus.RunLoop(morpheus.LoopOpts{
				DryRun:            dryRun,
				Path:              path,
				NonInteractive:    nonInteractive,
				Goal:              goal,
				MaxCycles:         maxCycles,
				DeveloperMaxTurns: devTurns,
				TesterMaxTurns:    testTurns,
				ReviewerMaxTurns:  reviewTurns,
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be generated without writing files")
	cmd.Flags().StringVar(&path, "path", "", "Project directory (defaults to current directory)")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Skip all prompts — use flags for inputs (for automation pipelines)")
	cmd.Flags().StringVar(&goal, "goal", "", "Loop goal description (non-interactive mode)")
	cmd.Flags().IntVar(&maxCycles, "max-cycles", 0, "Max development cycles (default: 12, non-interactive mode)")
	cmd.Flags().IntVar(&devTurns, "dev-turns", 0, "Developer max turns per cycle (default: 30, non-interactive mode)")
	cmd.Flags().IntVar(&testTurns, "test-turns", 0, "Tester max turns per cycle (default: 20, non-interactive mode)")
	cmd.Flags().IntVar(&reviewTurns, "review-turns", 0, "Reviewer max turns per cycle (default: 12, non-interactive mode)")

	return cmd
}

func finalizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "finalize",
		Short: "Populate .claude/ context files after loop completes",
		Example: `  # Post-loop context update in the current project
  morpheus finalize`,
		Run: func(cmd *cobra.Command, args []string) {
			morpheus.RunFinalize()
		},
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate scaffolded service before running loop",
		Example: `  # Verify the current service is ready for the autonomous loop
  morpheus doctor`,
		Run: func(cmd *cobra.Command, args []string) {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: getting working directory: %v\n", err)
				os.Exit(1)
			}
			morpheus.RunDoctor(cwd)
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show loop progress at a glance",
		Example: `  # Display loop progress for the current project
  morpheus status`,
		Run: func(cmd *cobra.Command, args []string) {
			morpheus.RunStatus()
		},
	}
}

func reportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Generate post-loop analytics report",
		Example: `  # Analyze a completed autonomous loop
  morpheus report`,
		Run: func(cmd *cobra.Command, args []string) {
			morpheus.RunReport()
		},
	}
}
