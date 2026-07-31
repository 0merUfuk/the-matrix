package morpheus

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/x/term"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/safewrite"
	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// InitOpts configures the morp init command.
type InitOpts struct {
	DryRun      bool
	Force       bool
	OutputDir   string
	ContextPath string // Path to .neo.json for pre-populated context
	ServiceName string // Override service name when using --context (NEO-06)
}

// RunInit scaffolds a new service with complete .claude/ + .autonomous/ setup.
func RunInit(opts InitOpts) {
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("morp — autonomous development infrastructure generator")))

	var ctx ProjectContext
	var err error

	if opts.ContextPath != "" {
		// Context-driven flow: load .neo.json → skip entry wizard → run delta wizard.
		// We run the delta wizard (remaining questions neo does not collect: ports,
		// entities, auth, etc.) rather than skipping all interaction. This is by design:
		// .neo.json does not capture all ProjectContext fields morp requires.
		// NEO-06 passes --service-name to override the project name with the actual
		// service name for each service in the loop.
		profile, loadErr := LoadContextProfile(opts.ContextPath)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Error: %v", loadErr)))
			os.Exit(1)
		}

		preCtx := MapContextToProject(profile)
		// Override service name if explicitly provided (e.g., by neo init --context).
		// MapContextToProject maps projectName → ServiceName, which is correct for
		// standalone use but wrong when neo loops over individual services.
		if opts.ServiceName != "" {
			preCtx.ServiceName = opts.ServiceName
			preCtx.ServiceNamePascal = wizard.ToPascalCase(opts.ServiceName)
		}
		cli.PrintDim(fmt.Sprintf("Context loaded from %s: %s (%s)", opts.ContextPath, preCtx.ServiceName, preCtx.Tech))
		fmt.Println()

		// When stdin is not a TTY (subprocess invocation by `neo init`, CI
		// pipelines, scripted runs), the delta wizards cannot prompt. The
		// previous behavior was to call them anyway, capture the huh "no TTY"
		// error, print "Aborted.", and exit 0 — silently producing no files
		// while looking like the wizard was skipped. Detect no-TTY here and
		// fill in defaults instead so the chain `neo init` → `morp init
		// --context` actually scaffolds files end-to-end.
		if !term.IsTerminal(os.Stdin.Fd()) {
			ctx = *preCtx
			ApplyContextDefaults(&ctx)
			cli.PrintDim("No TTY detected — using non-interactive defaults from context.")
			fmt.Println()
		} else {
			if preCtx.IsInternal {
				ctx, err = RunInternalWizardWithContext(*preCtx)
			} else {
				ctx, err = RunGeneralWizardWithContext(*preCtx)
			}
			if err != nil {
				fmt.Println("\nAborted.")
				os.Exit(0)
			}
		}
	} else {
		// Standard flow: entry wizard → tech wizard
		projectType, mode, entryErr := RunEntryWizard()
		if entryErr != nil {
			fmt.Println("\nAborted.")
			os.Exit(0)
		}

		// Post-v1 redirect
		if mode == "post-v1" {
			cli.PrintDim("Post-v1 mode: run morp finalize from your service directory after the loop completes.")
			fmt.Println()
			return
		}

		if projectType == "internal" {
			ctx, err = RunInternalWizard()
		} else {
			ctx, err = RunGeneralWizard()
		}
		if err != nil {
			fmt.Println("\nAborted.")
			os.Exit(0)
		}
	}

	// Step 3: Output directory
	outputDir := opts.OutputDir
	if outputDir == "" {
		cwd, _ := os.Getwd()
		outputDir = filepath.Join(cwd, ctx.ServiceName)
	}
	outputDir, _ = filepath.Abs(outputDir)

	fmt.Printf("  Creating workspace at: %s\n\n", outputDir)

	// Step 4: Dry-run
	if opts.DryRun {
		manifest := BuildMorpheusManifest(&ctx, outputDir)
		fmt.Printf("  %s\n\n", cli.Bold(cli.Cyan(fmt.Sprintf("── Dry Run: %d files would be generated to %s ──", len(manifest), outputDir))))
		cwd, _ := os.Getwd()
		for _, entry := range manifest {
			rel, _ := filepath.Rel(cwd, entry.OutputPath)
			tag := cli.Green("[tmpl]  ")
			if entry.IsStatic {
				tag = cli.Cyan("[static]")
			}
			fmt.Printf("  %s %s\n", tag, rel)
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

	// Step 6: Generate files
	fmt.Printf("  Generating project structure...\n")
	writtenFiles, err := GenerateMorpheusFiles(&ctx, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("File generation failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("✓ Generated %d files", len(writtenFiles))))

	// Step 7: Task decomposition (claude -p)
	fmt.Printf("  Generating tasks/todo.md...\n")
	todoContent, err := DecomposeTasksWithClaude(&ctx)
	if err != nil {
		cli.PrintDim(fmt.Sprintf("Claude unavailable — using skeleton template (%v)", err))
		todoContent = BuildSkeletonTodo(&ctx)
	}
	todoPath := filepath.Join(outputDir, "tasks/todo.md")
	if err := config.WriteFileString(todoPath, todoContent); err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Failed to write tasks/todo.md: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("  %s\n", cli.Green("✓ tasks/todo.md generated"))

	// Step 8: Summary
	printInitSummary(&ctx, outputDir, len(writtenFiles))
}

func printInitSummary(ctx *ProjectContext, outputDir string, fileCount int) {
	cwd, _ := os.Getwd()
	rel, _ := filepath.Rel(cwd, outputDir)
	if rel == "" {
		rel = outputDir
	}

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Green("✓ morp init complete")))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Service:  %s (%s)", ctx.ServiceName, ctx.Tech)))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Location: %s/", rel)))
	fmt.Printf("  %s\n\n", cli.Bold(fmt.Sprintf("Files:    %d generated", fileCount)))

	fmt.Printf("  %s\n\n", cli.Cyan("Next steps:"))
	cli.PrintWhite(fmt.Sprintf("cd %s", rel))
	cli.PrintWhite("cat tasks/todo.md               # Review the task list")
	cli.PrintWhite("morp doctor                  # Validate before running loop")
	cli.PrintWhite("bash .autonomous/loop.sh         # Start the autonomous dev loop")
	fmt.Println()

	if ctx.IsGo {
		cli.PrintDim("  Don't forget:")
		cli.PrintWhite("go mod init " + ctx.ModulePath)
		cli.PrintWhite("make install-tools")
		fmt.Println()
	}
}
