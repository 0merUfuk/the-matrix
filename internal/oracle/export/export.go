package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
)

// RunExportOpts configures the oracle export command.
type RunExportOpts struct {
	From     string
	Format   string // format name or "all"
	To       string
	Language string // explicit override, or "" for auto-detect
}

// RunExport is the main entry point called from the cobra command.
// Reads docs from From, dispatches to the requested format adapter(s).
// If format is "all", runs all registered formats.
func RunExport(opts RunExportOpts) error {
	// ── 1. Resolve absolute paths ─────────────────────────────────────────────
	resolvedFrom, err := filepath.Abs(opts.From)
	if err != nil {
		return fmt.Errorf("resolving --from path: %w", err)
	}
	resolvedTo, err := filepath.Abs(opts.To)
	if err != nil {
		return fmt.Errorf("resolving --to path: %w", err)
	}

	// ── 2. Verify source directory ────────────────────────────────────────────
	if !config.FileExists(resolvedFrom) {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error: Source directory not found: %s", resolvedFrom)))
		os.Exit(1)
	}
	if !config.IsDir(resolvedFrom) {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error: --from must be a directory, got a file: %s", resolvedFrom)))
		os.Exit(1)
	}

	// ── 3. Read docs ──────────────────────────────────────────────────────────
	result, err := ReadDocs(resolvedFrom)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error reading docs: %v", err)))
		os.Exit(1)
	}
	if len(result.Docs) == 0 {
		if result.Total > 0 {
			msg := fmt.Sprintf("All %d docs in %s failed validation (skipped: %s). Check minimum line count (50 lines).",
				result.Total, resolvedFrom, strings.Join(result.Skipped, ", "))
			fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red("Error: "+msg))
		} else {
			fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error: No .md files found in %s", resolvedFrom)))
		}
		os.Exit(1)
	}
	docs := result.Docs

	// ── 3b. Report skipped docs ───────────────────────────────────────────────
	if len(result.Skipped) > 0 {
		cli.PrintWarning(fmt.Sprintf("%d of %d docs skipped (failed validation)", len(result.Skipped), result.Total))
		for _, name := range result.Skipped {
			cli.PrintDim("  skipped: " + name)
		}
		fmt.Println()
	}

	// ── 4. Auto-detect language ───────────────────────────────────────────────
	stackName := filepath.Base(resolvedFrom)
	language := opts.Language
	if language == "" {
		language = DetectLanguage(stackName)
	}

	// Best-effort framework detection from dir name (e.g., "go-chi" -> "chi")
	framework := ""
	parts := strings.SplitN(stackName, "-", 2)
	if len(parts) == 2 {
		framework = parts[1]
	}

	exportOpts := ExportOpts{
		StackName: stackName,
		Language:  language,
		Framework: framework,
	}

	// ── 5. Validate format ────────────────────────────────────────────────────
	formatName := opts.Format
	if formatName != "all" {
		if _, ok := Registry[formatName]; !ok {
			available := strings.Join(FormatNames(), ", ")
			fmt.Fprintf(os.Stderr, "\n  %s\n", cli.Red(fmt.Sprintf("Error: Unknown format %q", formatName)))
			fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Dim(fmt.Sprintf("Available: %s, all", available)))
			os.Exit(1)
		}
	}

	// ── 6. Print header ───────────────────────────────────────────────────────
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("oracle export")))
	cli.PrintDim(fmt.Sprintf("From:     %s", cli.PathToHome(resolvedFrom)))
	cli.PrintDim(fmt.Sprintf("Format:   %s", formatName))
	cli.PrintDim(fmt.Sprintf("To:       %s", cli.PathToHome(resolvedTo)))
	cli.PrintDim(fmt.Sprintf("Docs:     %d", len(docs)))
	if language != "" {
		cli.PrintDim(fmt.Sprintf("Language: %s", language))
	}
	fmt.Println()

	// ── 7. Dispatch to format adapter(s) ──────────────────────────────────────
	if formatName == "all" {
		var failures int
		for _, name := range FormatNames() {
			f := Registry[name]
			if err := f.Export(docs, resolvedTo, exportOpts); err != nil {
				fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Error exporting %s: %v", name, err)))
				failures++
				continue
			}
			fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("%s: %s", f.Name(), f.Description())))
		}
		if failures > 0 {
			fmt.Println()
			return fmt.Errorf("%d format(s) failed to export", failures)
		}
	} else {
		f := Registry[formatName]
		if err := f.Export(docs, resolvedTo, exportOpts); err != nil {
			fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error: %v", err)))
			os.Exit(1)
		}
		fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("%s: %s", f.Name(), f.Description())))
	}

	fmt.Println()
	return nil
}
