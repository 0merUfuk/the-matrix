package oracle

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/safewrite"
	"github.com/charmbracelet/x/term"
)

// ResearchOpts configures the oracle research command.
type ResearchOpts struct {
	DryRun           bool
	Force            bool
	OutputDir        string
	ConfigFile       string
	GoldStandardsDir string // --gold-standards flag value
}

// RunResearch scaffolds a 3-phase autonomous research workspace.
func RunResearch(opts ResearchOpts) {
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("oracle — automated knowledge collector")))

	// Step 1: Collect stack config (wizard or --config file)
	var ctx StackContext
	if opts.ConfigFile != "" {
		configPath, _ := filepath.Abs(opts.ConfigFile)
		if !config.FileExists(configPath) {
			fmt.Fprintf(os.Stderr, "%s\n", cli.Red(fmt.Sprintf("Config file not found: %s", configPath)))
			os.Exit(1)
		}
		loaded, err := config.ReadJSON[StackContext](configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", cli.Red(fmt.Sprintf("Invalid config file: %v", err)))
			os.Exit(1)
		}
		ctx = loaded
		// Ensure derived fields
		if ctx.TopicCount == 0 && len(ctx.SelectedTopics) > 0 {
			ctx.TopicCount = len(ctx.SelectedTopics)
		}
		if ctx.CreatedDate == "" {
			ctx.CreatedDate = todayISO()
		}
		cli.PrintDim(fmt.Sprintf("Using config: %s", configPath))
		cli.PrintDim(fmt.Sprintf("Stack: %s (%s)", ctx.StackName, ctx.FrameworkVersion))
		fmt.Println()
	} else {
		var err error
		ctx, err = RunStackWizard()
		if err != nil {
			fmt.Println("\nAborted.")
			os.Exit(0)
		}
	}

	// Step 2: Determine output directory
	originalOutputDir := opts.OutputDir
	if originalOutputDir == "" {
		originalOutputDir = ctx.OutputPath
	}
	if originalOutputDir == "" {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red("Error: output directory is required (got empty path)"))
		os.Exit(1)
	}
	outputDir, err := safewrite.ResolveOutputPath(originalOutputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Error resolving output path: %v", err)))
		os.Exit(1)
	}

	// Step 3: Show resolved path (skip confirmation in --config mode)
	// In Go version, config mode skips interactive confirmation
	if opts.ConfigFile == "" {
		// In interactive mode, we'd prompt to confirm path — for now, use the resolved path
		fmt.Printf("  Creating workspace at: %s\n\n", outputDir)
	}

	// Step 4: Dry-run — print manifest and exit
	if opts.DryRun {
		manifest := BuildDryRunManifest(&ctx, outputDir)
		fmt.Printf("  %s\n\n", cli.Bold(cli.Cyan(fmt.Sprintf("── Dry Run: %d files would be generated to %s ──", len(manifest), outputDir))))
		cwd, _ := os.Getwd()
		for _, entry := range manifest {
			rel, _ := filepath.Rel(cwd, entry.OutputPath)
			tag := cli.Green("[tmpl]  ")
			if entry.IsStatic {
				tag = cli.Cyan("[static]")
			}
			fmt.Printf("  %s %s\n", tag, rel)
			cli.PrintDim(fmt.Sprintf("       ← %s", entry.TemplateName))
		}
		cli.PrintDim("\nNo files written (--dry-run mode).")
		return
	}

	// Step 5: Conflict check
	if config.IsDir(outputDir) {
		isTTY := term.IsTerminal(os.Stdin.Fd())
		ok, err := safewrite.ConfirmOverwrite(safewrite.OverwriteOpts{
			Path:  outputDir,
			Force: opts.Force,
			IsTTY: isTTY,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}
		if !ok {
			if isTTY {
				fmt.Println("\nAborted.")
				return
			}
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Error: %s already exists and stdin is not a TTY. Re-run with --force to overwrite.", outputDir)))
			os.Exit(1)
		}
		cli.PrintDim(fmt.Sprintf("Overwriting %s...", outputDir))
		if err := os.RemoveAll(outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Failed to remove existing directory: %v", err)))
			os.Exit(1)
		}
	}

	// Step 5b: Resolve gold standards
	if opts.GoldStandardsDir != "" {
		absGold, err := filepath.Abs(opts.GoldStandardsDir)
		if err != nil || !config.IsDir(absGold) {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Gold standards directory not found: %s", opts.GoldStandardsDir)))
			os.Exit(1)
		}
		if !hasMDFiles(absGold) {
			cli.PrintWarning(fmt.Sprintf("--gold-standards directory contains no .md files: %s", absGold))
		}
		ctx.GoldStandardsDir = absGold
	}
	goldDir := ResolveGoldStandardsDir(ctx.StackName, ctx.GoldStandardsDir)
	if goldDir != "" {
		ctx.GoldStandardsDir = goldDir
		ctx.GoldStandards = LoadGoldStandardsFromDir(goldDir)
		if len(ctx.GoldStandards) > 0 {
			cli.PrintDim(fmt.Sprintf("Gold standards: %d docs from %s", len(ctx.GoldStandards), cli.PathToHome(goldDir)))
		} else {
			cli.PrintDim("Gold standards: directory found but no .md files — using placeholder")
		}
	} else {
		ctx.GoldStandardsDir = "" // clear — no usable gold standards found
		ctx.GoldStandards = map[string]string{}
		cli.PrintDim("Gold standards: none — synthesizer will rely on research quality alone")
	}
	fmt.Println()

	// Step 6: Generate files
	fmt.Printf("  Generating research workspace...\n")

	writtenFiles, err := GenerateFiles(&ctx, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("File generation failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("✓ Generated %d files", len(writtenFiles))))

	// Step 7: Print summary
	printResearchSummary(&ctx, outputDir, len(writtenFiles))
}

func printResearchSummary(ctx *StackContext, outputDir string, fileCount int) {
	cwd, _ := os.Getwd()
	rel, _ := filepath.Rel(cwd, outputDir)
	if rel == "" {
		rel = outputDir
	}

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Green("✓ oracle research workspace ready")))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Stack:    %s (%s)", ctx.StackName, ctx.FrameworkVersion)))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Location: %s/", rel)))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Topics:   %d selected", ctx.TopicCount)))
	fmt.Printf("  %s\n\n", cli.Bold(fmt.Sprintf("Files:    %d generated", fileCount)))

	fmt.Printf("  %s\n\n", cli.Cyan("Next steps:"))
	cli.PrintWhite(fmt.Sprintf("cd %s", rel))
	cli.PrintWhite("bash .autonomous/loop.sh   # Start the 3-phase research loop")
	cli.PrintDim("\n  Phase 1 (automated): 4 research cycles — researcher + reviewer agents")
	cli.PrintDim("  Phase 2 (human gate): review tasks/research/*.md, then:")
	cli.PrintWhite("touch tasks/SYNTHESIZE_APPROVED   # Approve and continue")
	cli.PrintDim("  Phase 3 (automated): synthesizer writes 19 knowledge docs to output/")
	fmt.Println()
	cli.PrintDim("  Monitor progress:")
	cli.PrintWhite("oracle status               # Loop progress at a glance")
	cli.PrintWhite("tail -f tasks/loop.log      # Live logs")
	fmt.Println()
}

func todayISO() string {
	return time.Now().Format("2006-01-02")
}
