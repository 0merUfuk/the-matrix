package oracle

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/charmbracelet/x/term"
)

// InjectOpts configures the oracle inject command.
type InjectOpts struct {
	From string
	To   string
}

// RunInject validates oracle output docs and writes them to a .claude/knowledge/ directory.
func RunInject(opts InjectOpts) {
	resolvedFrom, err := filepath.Abs(opts.From)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error resolving --from path: %v", err)))
		os.Exit(1)
	}
	resolvedTo, err := filepath.Abs(opts.To)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error resolving --to path: %v", err)))
		os.Exit(1)
	}

	// ─── 1. Verify source directory exists ────────────────────────────────────
	if !config.FileExists(resolvedFrom) {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error: Source directory not found: %s", resolvedFrom)))
		os.Exit(1)
	}

	if !config.IsDir(resolvedFrom) {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error: --from must be a directory, got a file: %s", resolvedFrom)))
		os.Exit(1)
	}

	// ─── 2. Read all .md files from source ────────────────────────────────────
	mdFiles, err := config.ListMDFiles(resolvedFrom)
	if err != nil || len(mdFiles) == 0 {
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", cli.Red(fmt.Sprintf("Error: No .md files found in %s", resolvedFrom)))
		os.Exit(1)
	}

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("oracle inject")))
	cli.PrintDim(fmt.Sprintf("From: %s", resolvedFrom))
	cli.PrintDim(fmt.Sprintf("To:   %s", resolvedTo))
	cli.PrintDim(fmt.Sprintf("Docs: %d found", len(mdFiles)))
	fmt.Println()

	// ─── 3. Read and validate each file ───────────────────────────────────────
	type validDoc struct {
		Filename string
		Content  string
	}
	type failedDoc struct {
		Filename string
		Errors   []string
	}

	var docs []validDoc
	var failures []failedDoc

	sort.Strings(mdFiles)
	for _, filename := range mdFiles {
		filePath := filepath.Join(resolvedFrom, filename)
		content, err := config.ReadFileString(filePath)
		if err != nil {
			failures = append(failures, failedDoc{Filename: filename, Errors: []string{fmt.Sprintf("read error: %v", err)}})
			continue
		}

		errs := ValidateInjectDoc(filename, content)
		if len(errs) > 0 {
			failures = append(failures, failedDoc{Filename: filename, Errors: errs})
		} else {
			docs = append(docs, validDoc{Filename: filename, Content: content})
		}
	}

	// ─── 4. Fail fast if any validation errors (all-or-nothing write) ──────────
	if len(failures) > 0 {
		label := "file"
		if len(failures) > 1 {
			label = "files"
		}
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Validation failed — nothing written (%d %s failed):", len(failures), label)))
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "  %s\n", cli.Red("✗  "+f.Filename))
			for _, e := range f.Errors {
				cli.PrintDim("     " + e)
			}
		}
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	fmt.Printf("  %s\n\n", cli.Green(fmt.Sprintf("✓  All %d docs passed validation", len(docs))))

	// ─── 5. Sequential write: ensure target exists, then write all files ──────
	// Note: writes are sequential, not atomic — an interrupted write may leave
	// partial results in the target directory.
	if err := config.EnsureDir(resolvedTo); err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Error creating target directory: %v", err)))
		os.Exit(1)
	}

	var newFiles, updatedFiles []string
	for _, doc := range docs {
		targetPath := filepath.Join(resolvedTo, doc.Filename)
		if config.FileExists(targetPath) {
			updatedFiles = append(updatedFiles, doc.Filename)
		} else {
			newFiles = append(newFiles, doc.Filename)
		}
	}

	// Write docs
	for _, doc := range docs {
		if err := os.WriteFile(filepath.Join(resolvedTo, doc.Filename), []byte(doc.Content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Error writing %s: %v", doc.Filename, err)))
			os.Exit(1)
		}
	}

	// ─── 6. Write _index.md ───────────────────────────────────────────────────
	docInfos := make([]DocInfo, len(docs))
	for i, d := range docs {
		docInfos[i] = DocInfo{Filename: d.Filename, Content: d.Content}
	}
	indexContent := BuildIndex(docInfos, resolvedFrom, resolvedTo)
	if err := os.WriteFile(filepath.Join(resolvedTo, "_index.md"), []byte(indexContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Error writing _index.md: %v", err)))
		os.Exit(1)
	}

	// ─── 7. Print summary ─────────────────────────────────────────────────────
	relTo := cli.PathToHome(resolvedTo)
	fmt.Printf("  %s\n", cli.Bold(cli.Green(fmt.Sprintf("Injected %d docs to %s", len(docs), relTo))))
	if len(newFiles) > 0 {
		fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("+ %d new", len(newFiles))))
	}
	if len(updatedFiles) > 0 {
		fmt.Printf("  %s\n", cli.Cyan(fmt.Sprintf("~ %d updated", len(updatedFiles))))
	}
	cli.PrintDim("\n_index.md written")
	fmt.Println()

	// ─── 8. Offer gold standard registration ─────────────────────────────────
	// Only prompt in interactive terminals
	if term.IsTerminal(os.Stdin.Fd()) {
		promptGoldStandardRegistration(resolvedFrom)
	}
}

// promptGoldStandardRegistration asks the user if they want to register injected
// docs as gold standard examples for future oracle research runs.
func promptGoldStandardRegistration(sourceDir string) {
	fmt.Printf("  %s ", cli.Cyan(fmt.Sprintf("Register as gold standard for future research? [y/N]:")))
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer != "y" && answer != "yes" {
		return
	}

	// Detect language from the source directory name
	// Oracle output dirs are typically named like "go-internal", "nodejs-express", etc.
	dirName := filepath.Base(sourceDir)
	lang := LanguageToConventionDir(dirName)
	if lang == "" {
		// Try parent directory name (e.g., output/go-internal/ → sourceDir is go-internal)
		parentName := filepath.Base(filepath.Dir(sourceDir))
		lang = LanguageToConventionDir(parentName)
	}
	if lang == "" {
		cli.PrintWarning("  Could not detect language from directory name.")
		cli.PrintDim("  To register manually, copy representative docs to:")
		cli.PrintDim("    ~/.the-matrix/gold-standards/<language>/")
		fmt.Println()
		return
	}

	// Select representative files
	selected := GoldStandardRegistrationFiles(sourceDir)
	if len(selected) == 0 {
		cli.PrintWarning("  No .md files suitable for gold standard registration.")
		return
	}

	// Create target directory
	home, err := os.UserHomeDir()
	if err != nil {
		cli.PrintWarning(fmt.Sprintf("  Error: could not determine home directory: %v", err))
		return
	}
	targetDir := filepath.Join(home, ".the-matrix", "gold-standards", lang)
	if err := config.EnsureDir(targetDir); err != nil {
		cli.PrintWarning(fmt.Sprintf("  Error creating %s: %v", targetDir, err))
		return
	}

	// Copy selected files
	copied := 0
	for _, filename := range selected {
		src := filepath.Join(sourceDir, filename)
		dst := filepath.Join(targetDir, filename)
		if err := config.CopyFile(src, dst, targetDir); err != nil {
			cli.PrintWarning(fmt.Sprintf("  Warning: could not copy %s: %v", filename, err))
			continue
		}
		copied++
	}

	relTarget := cli.PathToHome(targetDir)
	fmt.Printf("\n  %s\n", cli.Green(fmt.Sprintf("Registered %d gold standard docs at %s", copied, relTarget)))
	cli.PrintDim("  Future `oracle research` runs will auto-detect these for calibration.")
	fmt.Println()
}
