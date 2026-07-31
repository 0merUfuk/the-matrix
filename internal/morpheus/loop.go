package morpheus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// LoopOpts configures the morp loop command.
type LoopOpts struct {
	DryRun         bool
	Path           string
	NonInteractive bool

	// Non-interactive inputs (used when NonInteractive=true, ignored otherwise).
	Goal              string
	MaxCycles         int
	DeveloperMaxTurns int
	TesterMaxTurns    int
	ReviewerMaxTurns  int
}

// LoopContext holds the data collected by the loop wizard, consumed by loop templates.
type LoopContext struct {
	ProjectName       string
	ProjectNamePascal string
	Goal              string
	MaxCycles         int
	DeveloperMaxTurns int
	TesterMaxTurns    int
	ReviewerMaxTurns  int
	CreatedDate       string

	// Aliases for shared templates that expect .ServiceName / .ServiceNamePascal
	ServiceName       string
	ServiceNamePascal string
}

// defaultCycleConfig is the canonical default cycle configuration.
// Both interactive (wizard) and non-interactive modes derive from this.
var defaultCycleConfig = CycleConfig{
	MaxCycles:         12,
	DeveloperMaxTurns: 30,
	TesterMaxTurns:    20,
	ReviewerMaxTurns:  12,
}

// NewLoopContext creates a LoopContext with shared template aliases populated.
func NewLoopContext(projectName, goal string, cc CycleConfig) *LoopContext {
	pascal := wizard.ToPascalCase(projectName)
	return &LoopContext{
		ProjectName:       projectName,
		ProjectNamePascal: pascal,
		Goal:              goal,
		MaxCycles:         cc.MaxCycles,
		DeveloperMaxTurns: cc.DeveloperMaxTurns,
		TesterMaxTurns:    cc.TesterMaxTurns,
		ReviewerMaxTurns:  cc.ReviewerMaxTurns,
		CreatedDate:       wizard.TodayISO(),
		ServiceName:       projectName,
		ServiceNamePascal: pascal,
	}
}

// RunLoop sets up an autonomous development loop on an existing project.
func RunLoop(opts LoopOpts) {
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("morp loop — autonomous development loop for any project")))

	// 1. Resolve project path
	projectDir := opts.Path
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}
	projectDir, _ = filepath.Abs(projectDir)

	// Warn if non-interactive-only flags are passed without --non-interactive
	if !opts.NonInteractive && (opts.Goal != "" || opts.MaxCycles > 0 || opts.DeveloperMaxTurns > 0 || opts.TesterMaxTurns > 0 || opts.ReviewerMaxTurns > 0) {
		cli.PrintWarning("--goal, --max-cycles, --dev-turns, --test-turns, --review-turns are only used with --non-interactive and will be ignored.")
		fmt.Println()
	}

	// 2. Validate CLAUDE.md exists
	claudemd := filepath.Join(projectDir, "CLAUDE.md")
	if !config.FileExists(claudemd) {
		cli.PrintError("No CLAUDE.md found in " + projectDir)
		fmt.Println()
		cli.PrintDim("  morp loop requires a CLAUDE.md — it's how agents learn the project.")
		cli.PrintDim("  Create one with project rules, tech stack, and build commands.")
		fmt.Println()
		os.Exit(1)
	}

	// 3. Check .autonomous/ exists → prompt overwrite (skip in non-interactive mode)
	autonomousDir := filepath.Join(projectDir, ".autonomous")
	if config.IsDir(autonomousDir) && !opts.NonInteractive {
		overwrite, err := wizard.AskConfirm(".autonomous/ already exists. Overwrite?", false)
		if err != nil || !overwrite {
			fmt.Println("\n  Aborted.")
			return
		}
	}

	// 4. Derive project name from directory basename
	projectName := filepath.Base(projectDir)

	// 5. Run wizard or use non-interactive defaults
	var goal string
	var cycleConfig CycleConfig

	if opts.NonInteractive {
		goal = opts.Goal
		cycleConfig = buildCycleConfig(opts)
	} else {
		var err error
		goal, cycleConfig, err = RunLoopWizard()
		if err != nil {
			fmt.Println("\n  Aborted.")
			os.Exit(0)
		}
	}

	ctx := NewLoopContext(projectName, goal, cycleConfig)

	// 6. Dry-run
	if opts.DryRun {
		manifest := BuildLoopManifest(ctx, projectDir)
		fmt.Printf("  %s\n\n", cli.Bold(cli.Cyan(fmt.Sprintf("── Dry Run: %d files would be generated ──", len(manifest)))))
		for _, entry := range manifest {
			rel, _ := filepath.Rel(projectDir, entry.OutputPath)
			tag := cli.Green("[tmpl]   ")
			if entry.IsStatic {
				tag = cli.Cyan("[static]")
			}
			fmt.Printf("  %s %s\n", tag, rel)
		}
		fmt.Printf("\n  + tasks/todo.md (generated separately)\n")
		cli.PrintDim("\n  No files written (--dry-run mode).")
		return
	}

	// 7. Generate files
	fmt.Printf("  Generating loop infrastructure...\n")
	writtenFiles, err := GenerateLoopFiles(ctx, projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("File generation failed: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("✓ Generated %d files", len(writtenFiles))))

	// 8. Generate tasks/todo.md
	todoPath := filepath.Join(projectDir, "tasks/todo.md")
	if ctx.Goal != "" {
		fmt.Printf("  Decomposing goal into tasks/todo.md...\n")
		todoContent, err := DecomposeGoalWithClaude(projectDir, ctx.Goal)
		if err != nil {
			cli.PrintDim(fmt.Sprintf("  Claude unavailable — using empty skeleton (%v)", err))
			todoContent = BuildEmptyTodo(ctx.ProjectNamePascal)
		}
		if err := config.WriteFileString(todoPath, todoContent); err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Failed to write tasks/todo.md: %v", err)))
			os.Exit(1)
		}
	} else {
		if err := config.WriteFileString(todoPath, BuildEmptyTodo(ctx.ProjectNamePascal)); err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Failed to write tasks/todo.md: %v", err)))
			os.Exit(1)
		}
	}
	fmt.Printf("  %s\n", cli.Green("✓ tasks/todo.md generated"))

	// 9. Ensure .claude/ directory exists (so loop.sh preflight passes)
	if err := config.EnsureDir(filepath.Join(projectDir, ".claude")); err != nil {
		cli.PrintWarning(fmt.Sprintf("Could not create .claude/ directory: %v", err))
	}

	// 10. Update .gitignore
	updateGitignore(projectDir)

	// 11. Summary
	printLoopSummary(ctx, projectDir, len(writtenFiles))
}

// updateGitignore appends .autonomous/ and tasks/ to .gitignore if not already present.
func updateGitignore(projectDir string) {
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	content := ""
	if config.FileExists(gitignorePath) {
		content, _ = config.ReadFileString(gitignorePath)
	}

	var toAdd []string
	if !strings.Contains(content, ".autonomous/") {
		toAdd = append(toAdd, ".autonomous/")
	}
	if !strings.Contains(content, "tasks/") {
		toAdd = append(toAdd, "tasks/")
	}

	if len(toAdd) > 0 {
		addition := "\n# morp loop (autonomous development infrastructure)\n" + strings.Join(toAdd, "\n") + "\n"
		if err := config.AppendFileString(gitignorePath, addition); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: failed to update .gitignore: %v\n", err)
		}
		cli.PrintDim("  Updated .gitignore")
	}
}

// buildCycleConfig constructs a CycleConfig from LoopOpts flags,
// falling back to defaultCycleConfig for any unset values.
func buildCycleConfig(opts LoopOpts) CycleConfig {
	cc := defaultCycleConfig // copy — not a pointer
	if opts.MaxCycles > 0 {
		cc.MaxCycles = opts.MaxCycles
	}
	if opts.DeveloperMaxTurns > 0 {
		cc.DeveloperMaxTurns = opts.DeveloperMaxTurns
	}
	if opts.TesterMaxTurns > 0 {
		cc.TesterMaxTurns = opts.TesterMaxTurns
	}
	if opts.ReviewerMaxTurns > 0 {
		cc.ReviewerMaxTurns = opts.ReviewerMaxTurns
	}
	return cc
}

func printLoopSummary(ctx *LoopContext, projectDir string, fileCount int) {
	cwd, _ := os.Getwd()
	rel, _ := filepath.Rel(cwd, projectDir)
	if rel == "" || rel == "." {
		rel = projectDir
	}

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Green("✓ morp loop complete")))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Project:  %s", ctx.ProjectName)))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Location: %s/", rel)))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Files:    %d + tasks/todo.md", fileCount)))
	if ctx.Goal != "" {
		fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Goal:     %s", ctx.Goal)))
	}
	fmt.Printf("  %s\n\n", cli.Bold(fmt.Sprintf("Cycles:   max=%d, dev=%d, test=%d, review=%d",
		ctx.MaxCycles, ctx.DeveloperMaxTurns, ctx.TesterMaxTurns, ctx.ReviewerMaxTurns)))

	fmt.Printf("  %s\n\n", cli.Cyan("Next steps:"))
	if rel != "." {
		cli.PrintWhite(fmt.Sprintf("  cd %s", rel))
	}
	cli.PrintWhite("  cat tasks/todo.md               # Review the task list")
	cli.PrintWhite("  bash .autonomous/loop.sh         # Start the autonomous dev loop")
	fmt.Println()
	cli.PrintDim("  To stop the loop: touch tasks/STOP")
	cli.PrintDim("  To pause for decisions: write to tasks/decisions-pending.md")
	fmt.Println()
}
