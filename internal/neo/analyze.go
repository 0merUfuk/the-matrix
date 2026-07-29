package neo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
)

// AnalyzeOpts configures the neo analyze command.
type AnalyzeOpts struct {
	Path   string
	DryRun bool
	Force  bool
}

// RunAnalyze scans an existing codebase, detects its stack, and generates
// a tailored Claude Code agent ecosystem without requiring a wizard.
func RunAnalyze(opts AnalyzeOpts) {
	cli.PrintHeader("neo", "analyze — codebase detection")

	// ─── 1. Resolve project path ─────────────────────────────────────────────
	projectPath := opts.Path
	if projectPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			cli.PrintError("cannot determine working directory: " + err.Error())
			os.Exit(1)
		}
		projectPath = wd
	}
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		cli.PrintError("cannot resolve absolute path: " + err.Error())
		os.Exit(1)
	}
	projectPath = absPath

	if !config.IsDir(projectPath) {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Directory not found: %s", projectPath)))
		os.Exit(1)
	}

	theme := cli.GetTheme()
	fmt.Println(theme.KeyValue([]cli.KV{
		{Key: "Scanning", Value: cli.PathToHome(projectPath)},
	}))
	fmt.Println()

	// ─── 2. Run detection ────────────────────────────────────────────────────
	detection := RunDetection(projectPath)

	// ─── 3. Run extraction ───────────────────────────────────────────────────
	extraction := RunExtraction(projectPath)

	// ─── 4. Print detection results ──────────────────────────────────────────
	printDetectionResults(detection, extraction)

	// ─── 5. Validate detection ───────────────────────────────────────────────
	if detection.Language == "unknown" {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red("No supported language detected. Is this a project directory?"))
		os.Exit(1)
	}

	// ─── 6. Build ProjectProfile from detection ──────────────────────────────
	profile := buildProfileFromDetection(projectPath, detection, extraction)

	// ─── 7. Dry-run mode ─────────────────────────────────────────────────────
	if opts.DryRun {
		manifest := BuildNeoManifest(&profile, projectPath)
		fmt.Printf("\n  %s\n", cli.Bold(cli.Cyan("Dry run — no files will be written.")))
		fmt.Printf("  Would generate: .claude/ (CLAUDE.md, rules/, agents/, skills/, SERVICE_CONTEXT.md, ...)\n\n")
		for _, entry := range manifest {
			rel, _ := filepath.Rel(projectPath, entry.OutputPath)
			fmt.Printf("  %s %s\n", cli.Green("[tmpl]"), rel)
		}
		fmt.Printf("\n  %s\n", cli.Dim(fmt.Sprintf("%d files total.", len(manifest))))
		return
	}

	// ─── 8. Check if .claude/ exists ─────────────────────────────────────────
	claudeDir := filepath.Join(projectPath, ".claude")
	if config.IsDir(claudeDir) && !opts.Force {
		fmt.Printf("\n  %s", cli.Yellow(".claude/ already exists. Manifest files will be overwritten;\n  existing custom files will be preserved. Continue? [y/N]: "))
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			cli.PrintDim("Aborted.")
			return
		}
	}

	// ─── 9. Generate ecosystem ───────────────────────────────────────────────
	fmt.Printf("\n  Generating .claude/ ecosystem...\n")

	written, err := GenerateEcosystem(&profile, projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Generation failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("Generated %d files", len(written))))

	// ─── 10. Print success summary ───────────────────────────────────────────
	printAnalyzeSummary(&profile, projectPath, len(written))
}

// printDetectionResults displays what was detected in a formatted table.
func printDetectionResults(d DetectionResult, e ProjectExtract) {
	theme := cli.GetTheme()

	fmt.Println(theme.SectionHeader("Detection Results"))
	fmt.Println()

	// Language
	langLevel := cli.Pass
	if d.Language == "unknown" {
		langLevel = cli.Fail
	}
	cli.PrintLine(langLevel, padRight("Language", 18), d.Language)

	// Framework
	fwLevel := cli.Pass
	if d.Framework == "unknown" {
		fwLevel = cli.Warn
	}
	cli.PrintLine(fwLevel, padRight("Framework", 18), d.Framework)

	// Architecture
	cli.PrintLine(cli.Pass, padRight("Architecture", 18), d.ArchStyle)

	// Testing
	testLevel := cli.Pass
	if d.TestingFramework == "unknown" {
		testLevel = cli.Warn
	}
	cli.PrintLine(testLevel, padRight("Testing", 18), d.TestingFramework)

	// Data layer
	cli.PrintLine(cli.Pass, padRight("Data layer", 18), d.DataLayer)

	// Queue
	queueDetail := "no"
	if d.HasQueue {
		queueDetail = "yes"
	}
	cli.PrintLine(cli.Pass, padRight("Queue", 18), queueDetail)

	// Observability
	if len(d.Observability) > 0 {
		cli.PrintLine(cli.Pass, padRight("Observability", 18), strings.Join(d.Observability, ", "))
	}

	// Project type
	cli.PrintLine(cli.Pass, padRight("Project type", 18), d.ProjectType)

	// originating project
	internalProjectDetail := "no"
	if d.IsInternal {
		internalProjectDetail = "yes"
	}
	cli.PrintLine(cli.Pass, padRight("originating project", 18), internalProjectDetail)

	// Build commands
	if len(e.BuildCommands) > 0 {
		// Sort command names for deterministic output
		var names []string
		for name := range e.BuildCommands {
			names = append(names, name)
		}
		sort.Strings(names)

		var cmds []string
		for _, name := range names {
			cmds = append(cmds, e.BuildCommands[name])
		}
		cli.PrintLine(cli.Pass, padRight("Build commands", 18), strings.Join(cmds, ", "))
	}

	// CI/CD
	var cicd []string
	if e.HasGitHub {
		cicd = append(cicd, "GitHub Actions")
	}
	if e.HasGitLab {
		cicd = append(cicd, "GitLab CI")
	}
	if e.HasCircleCI {
		cicd = append(cicd, "CircleCI")
	}
	if len(cicd) > 0 {
		cli.PrintLine(cli.Pass, padRight("CI/CD", 18), strings.Join(cicd, ", "))
	} else {
		cli.PrintLine(cli.Warn, padRight("CI/CD", 18), "none detected")
	}

	// Existing AI context
	if e.ExistingAI != "" {
		cli.PrintLine(cli.Pass, padRight("Existing AI ctx", 18), "found")
	}
}

// buildProfileFromDetection maps detection and extraction results to a
// ProjectProfile suitable for GenerateEcosystem.
func buildProfileFromDetection(projectPath string, d DetectionResult, e ProjectExtract) ProjectProfile {
	stackName := d.Language
	if d.Framework != "" && d.Framework != "unknown" {
		// Use just the framework base name for the stack name.
		// Replace "/" with "-" so the name is safe as a path component.
		fields := strings.Fields(d.Framework)
		fwName := d.Framework
		if len(fields) > 0 {
			fwName = fields[0]
		}
		stackName = d.Language + "-" + strings.ToLower(strings.ReplaceAll(fwName, "/", "-"))
	}

	return ProjectProfile{
		ProjectName: filepath.Base(projectPath),
		ProjectType: d.ProjectType,
		TeamSize:    "solo", // Cannot detect; sensible default
		IsInternal:  d.IsInternal,
		CreatedDate: time.Now().Format("2006-01-02"),
		OutputPath:  projectPath,
		Stacks: []StackProfile{{
			Name:             stackName,
			Language:         d.Language,
			Framework:        d.Framework,
			ArchStyle:        d.ArchStyle,
			TestingFramework: d.TestingFramework,
			HasQueue:         d.HasQueue,
			DataLayer:        d.DataLayer,
		}},
	}
}

// printAnalyzeSummary prints the post-generation success summary.
func printAnalyzeSummary(p *ProjectProfile, outputDir string, fileCount int) {
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Green("neo analyze complete")))

	theme := cli.GetTheme()
	kvs := []cli.KV{
		{Key: "Project", Value: fmt.Sprintf("%s (%s)", p.ProjectName, p.ProjectType)},
	}
	if len(p.Stacks) > 0 {
		kvs = append(kvs, cli.KV{Key: "Stack", Value: fmt.Sprintf("%s (%s / %s)", p.Stacks[0].Name, p.Stacks[0].Language, p.Stacks[0].Framework)})
	}
	kvs = append(kvs, cli.KV{Key: "Files", Value: fmt.Sprintf("%d generated", fileCount)})
	fmt.Println(theme.KeyValue(kvs))

	fmt.Printf("\n  %s\n\n", cli.Cyan("Next steps:"))
	cli.PrintWhite("neo doctor                       # Validate ecosystem")
	cli.PrintWhite("neo status                       # Dashboard view")

	if len(p.Stacks) > 0 {
		cli.PrintDim(fmt.Sprintf("\n  To research deeper knowledge for %s:", p.Stacks[0].Name))
		cli.PrintWhite("oracle research                  # Generate knowledge docs")
		cli.PrintWhite("trinity sync                     # Inject into .claude/knowledge/")
	}

	fmt.Println()
}

// padRight pads a string with spaces to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
