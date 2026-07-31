package neo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	huh "charm.land/huh/v2"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/registry"
	"github.com/0merUfuk/the-matrix/internal/validation"
	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// InitOpts configures the neo init command.
type InitOpts struct {
	DryRun bool
	Path   string
	Preset string // Named preset — skips wizard if non-empty
}

// runPresetFlow applies the preset template and derives the project name from
// the --path flag (or cwd) so the flow is fully non-interactive — suitable for
// CI pipelines with no TTY. A previous version called wizard.AskInput here,
// which blocked on TTY-less stdin and broke the advertised CI example.
func runPresetFlow(presetName, path string) (ProjectProfile, error) {
	profile, ok := GetPreset(presetName)
	if !ok {
		return ProjectProfile{}, fmt.Errorf("unknown preset %q — valid presets: %s",
			presetName, strings.Join(ListPresets(), ", "))
	}

	// Resolve output directory (default to cwd when --path is empty).
	var absPath string
	if path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return ProjectProfile{}, fmt.Errorf("resolving path: %w", err)
		}
		absPath = abs
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return ProjectProfile{}, fmt.Errorf("getting working directory: %w", err)
		}
		absPath = cwd
	}

	// Derive project name from the path basename. filepath.Base returns "/"
	// for the root path and "." for empty input — neither is usable as a
	// project name, so reject them with a clear error.
	name := filepath.Base(absPath)
	if name == "" || name == "." || name == "/" || name == string(filepath.Separator) {
		return ProjectProfile{}, fmt.Errorf(
			"cannot derive project name from path %q; supply a non-root --path (e.g. --path ./my-project)",
			absPath)
	}
	if err := wizard.ValidateKebabCase(name); err != nil {
		return ProjectProfile{}, fmt.Errorf(
			"derived project name %q from --path is not a valid kebab-case name (%v); rename the directory or pass a --path whose basename is kebab-case",
			name, err)
	}

	profile.ProjectName = name
	profile.OutputPath = absPath

	return profile, nil
}

// RunInit provisions a complete Claude Code agent ecosystem.
func RunInit(opts InitOpts) {
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("neo — ecosystem provisioner")))

	// Step 1: Collect profile — via preset or full wizard
	var profile ProjectProfile
	var err error
	if opts.Preset != "" {
		fmt.Printf("  %s %s\n\n", cli.Dim("Preset:"), cli.Bold(opts.Preset))
		profile, err = runPresetFlow(opts.Preset, opts.Path)
	} else {
		profile, err = RunProjectWizard()
	}
	if err != nil {
		// huh.ErrUserAborted is returned when the user presses Ctrl+C or Esc.
		// That is a clean cancellation — exit 0 with a brief message.
		// All other errors (unknown preset, OS errors) are failures — exit 1.
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("\nAborted.")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "\n  %s\n", cli.Red(err.Error()))
		os.Exit(1)
	}

	// When --path is provided in non-preset mode, override OutputPath.
	// In preset mode, runPresetFlow already applied --path.
	outputDir := profile.OutputPath
	if opts.Preset == "" && opts.Path != "" {
		abs, err := filepath.Abs(opts.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Cannot resolve path: %v", err)))
			os.Exit(1)
		}
		outputDir = abs
		// Keep profile.OutputPath in sync so .neo.json records the real
		// project root. Without this, `neo status` / `neo update` read
		// .neo.json and walk into the wrong directory.
		profile.OutputPath = abs
	}

	// Step 2: Print summary and confirm
	printProfileSummary(&profile)

	// Step 3: Dry-run — must come before registry check to avoid writing files
	if opts.DryRun {
		manifest := BuildNeoManifest(&profile, outputDir)
		fmt.Printf("  %s\n\n", cli.Bold(cli.Cyan(fmt.Sprintf("── Dry Run: %d files would be generated ──", len(manifest)))))
		cwd, _ := os.Getwd()
		for _, entry := range manifest {
			rel, _ := filepath.Rel(cwd, entry.OutputPath)
			fmt.Printf("  %s %s\n", cli.Green("[tmpl]"), rel)
		}
		cli.PrintDim("\n  Would check knowledge registry for matching packs.")
		cli.PrintDim("\nNo files written (--dry-run mode).")
		return
	}

	// Step 4: Registry check — look for pre-researched knowledge packs.
	// Skipped when --preset is set: preset mode is non-interactive, but
	// checkRegistryForStacks calls wizard.AskConfirm ("Use <pack> from
	// registry?") which requires a TTY. Prompting here contradicts the
	// "--preset is fully non-interactive" contract. The preset already
	// defines the stack list, so the registry pack is a wizard-flow
	// convenience we intentionally forgo in preset mode.
	var registryPulled map[string]bool
	if opts.Preset != "" {
		registryPulled = make(map[string]bool)
	} else {
		registryPulled = checkRegistryForStacks(&profile, outputDir)
	}

	// Step 5: Generate ecosystem
	fmt.Printf("  Generating .claude/ ecosystem...\n")

	written, err := GenerateEcosystem(&profile, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Generation failed: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("✓ Generated %d files", len(written))))

	// Step 6: Print summary
	printInitSummary(&profile, outputDir, len(written), registryPulled)

	// Step 7: Optionally scaffold services with morp
	RunMorpheusIntegration(&profile, outputDir)
}

func printProfileSummary(p *ProjectProfile) {
	fmt.Printf("  %s\n\n", cli.Bold("Project Profile:"))
	fmt.Printf("    Name:    %s\n", p.ProjectName)
	fmt.Printf("    Type:    %s\n", p.ProjectType)
	fmt.Printf("    Team:    %s\n", p.TeamSize)
	fmt.Printf("    originating project:  %v\n", p.IsInternal)
	fmt.Printf("    Stacks:  %d\n", len(p.Stacks))
	for i, s := range p.Stacks {
		fmt.Printf("      %d. %s (%s / %s)\n", i+1, s.Name, s.Language, s.Framework)
	}
	if len(p.Services) > 0 {
		fmt.Printf("    Services: %d\n", len(p.Services))
		for _, svc := range p.Services {
			fmt.Printf("      - %s (stack: %s, port: %d)\n", svc.Name, svc.Stack, svc.Port)
		}
	}
	fmt.Println()
}

func printInitSummary(p *ProjectProfile, outputDir string, fileCount int, registryPulled map[string]bool) {
	cwd, _ := os.Getwd()
	rel, _ := filepath.Rel(cwd, outputDir)
	if rel == "" {
		rel = outputDir
	}

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Green("✓ neo init complete")))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Project:  %s (%s, %s)", p.ProjectName, p.ProjectType, p.TeamSize)))
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Location: %s/", rel)))
	fmt.Printf("  %s\n\n", cli.Bold(fmt.Sprintf("Files:    %d generated", fileCount)))

	fmt.Printf("  %s\n\n", cli.Cyan("Next steps:"))
	cli.PrintWhite("neo doctor                       # Validate ecosystem")
	cli.PrintWhite("neo status                       # Dashboard view")

	// Show next steps for stacks not covered by registry.
	var uncovered []string
	for _, s := range p.Stacks {
		if !registryPulled[s.Name] {
			uncovered = append(uncovered, s.Name)
		}
	}

	if len(uncovered) > 0 {
		cli.PrintDim(fmt.Sprintf("\n  To research knowledge for %s:", uncovered[0]))
		cli.PrintWhite("oracle research                  # Generate knowledge docs")
		cli.PrintWhite("trinity sync                     # Inject into .claude/knowledge/")
	} else if len(registryPulled) > 0 {
		cli.PrintDim("\n  Knowledge packs were pulled from the registry.")
		cli.PrintWhite("neo registry list                # Browse available packs")
	}

	fmt.Println()
}

// checkRegistryForStacks checks the knowledge registry for matching packs
// for each stack in the profile. If matches are found, prompts the user
// and pulls the packs. Returns a map of stack names that were successfully
// pulled from the registry.
//
// This function handles all errors internally and never fails fatally —
// on any error (network timeout, registry unreachable, user interrupt),
// it returns what it has and lets the caller proceed normally.
func checkRegistryForStacks(profile *ProjectProfile, outputDir string) map[string]bool {
	pulled := make(map[string]bool)

	if len(profile.Stacks) == 0 {
		return pulled
	}

	fmt.Printf("  %s\n", cli.Dim("Checking knowledge registry..."))

	var index *registry.RegistryIndex
	err := cli.WithSpinner("Connecting to registry...", func() error {
		var fetchErr error
		index, fetchErr = registry.Load()
		return fetchErr
	})
	if err != nil {
		cli.PrintDim("  Registry unreachable, falling back to oracle research.")
		fmt.Println()
		return pulled
	}

	if len(index.Packs) == 0 {
		cli.PrintDim("  Registry is empty.")
		fmt.Println()
		return pulled
	}

	// Check each stack for a matching pack.
	for _, stack := range profile.Stacks {
		match := registry.FindMatch(index, stack.Language, stack.Framework)
		if match == nil {
			continue
		}

		// Found a match — prompt.
		fmt.Printf("\n  %s %s in registry (%s, score %d).\n",
			cli.Green("Found"),
			cli.Bold(match.DisplayName),
			match.RefreshedDate,
			match.QualityScore,
		)

		useIt, err := wizard.AskConfirm(fmt.Sprintf("Use %s from registry?", match.Name), true)
		if err != nil {
			// User interrupted — stop checking, return what we have.
			return pulled
		}
		if !useIt {
			continue
		}

		// Pull the pack.
		targetDir := filepath.Join(outputDir, ".claude", "knowledge", stack.Name)

		validateFn := func(filename, content string) error {
			errs := validation.ValidateInjectDoc(filename, content)
			if len(errs) > 0 {
				return fmt.Errorf("%s", strings.Join(errs, "; "))
			}
			return nil
		}

		pullErr := cli.WithSpinner(fmt.Sprintf("Pulling %s...", match.Name), func() error {
			return registry.Pull(match.Name, targetDir, validateFn)
		})
		if pullErr != nil {
			cli.PrintWarning(fmt.Sprintf("Failed to pull %s: %v", match.Name, pullErr))
			cli.PrintDim("  Will fall back to oracle research for this stack.")
			continue
		}

		// Count files.
		entries, _ := os.ReadDir(targetDir)
		docCount := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				docCount++
			}
		}

		cli.PrintSuccess(fmt.Sprintf("Pulled %d docs for %s from registry", docCount, stack.Name))
		pulled[stack.Name] = true
	}

	if len(pulled) > 0 {
		fmt.Println()
	}

	return pulled
}
