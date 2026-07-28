package trinity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
)

// SyncOpts configures the trinity sync command.
type SyncOpts struct {
	From       string
	To         string
	ProjectDir string
	DryRun     bool
}

// RunSync pushes oracle output to .claude/knowledge/.
// Supports explicit --from/--to or .trinity.json config mode.
func RunSync(opts SyncOpts) {
	root, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		root = opts.ProjectDir
	}
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}

	// ─── Resolve sync pairs ──────────────────────────────────────────────────
	var syncPairs []SyncPair

	if opts.From != "" && opts.To != "" {
		from, _ := filepath.Abs(opts.From)
		to, _ := filepath.Abs(opts.To)
		syncPairs = []SyncPair{{
			Name: filepath.Base(from),
			From: from,
			To:   to,
		}}
	} else {
		syncPairs = LoadConfig(root)
	}

	header := "trinity sync"
	if opts.DryRun {
		header += cli.Dim(" (dry run)")
	}
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan(header)))

	// ─── Process each sync pair ───────────────────────────────────────────────
	totalSynced := 0

	for _, pair := range syncPairs {
		if syncOnePair(pair, root, opts.DryRun) {
			totalSynced++
		}
	}

	// ─── Final summary ───────────────────────────────────────────────────────
	if len(syncPairs) > 1 {
		fmt.Printf("  %s\n\n", cli.Bold(fmt.Sprintf("%d/%d stacks synced", totalSynced, len(syncPairs))))
	}

	if totalSynced < len(syncPairs) && !opts.DryRun {
		os.Exit(1)
	}
}

// ─── Single pair sync ────────────────────────────────────────────────────────

func syncOnePair(pair SyncPair, projectRoot string, dryRun bool) bool {
	relFrom := cli.PathToHome(pair.From)
	relTo := cli.PathToHome(pair.To)

	fmt.Printf("  %s\n", cli.Bold(pair.Name))
	cli.PrintDim(fmt.Sprintf("From: %s", relFrom))
	cli.PrintDim(fmt.Sprintf("To:   %s", relTo))

	// ─── 1. Validate source exists ──────────────────────────────────────────
	if !config.IsDir(pair.From) {
		fmt.Printf("  %s\n\n", cli.Red(fmt.Sprintf("✗  Source not found: %s", relFrom)))
		return false
	}

	// ─── 2. Pre-compute stats ──────────────────────────────────────────────
	sourceFiles, err := config.ListMDFiles(pair.From)
	if err != nil || len(sourceFiles) == 0 {
		fmt.Printf("  %s\n\n", cli.Red("✗  No .md files in source"))
		return false
	}

	// Count new vs existing
	newCount := 0
	updateCount := 0
	for _, filename := range sourceFiles {
		targetPath := filepath.Join(pair.To, filename)
		if config.FileExists(targetPath) {
			updateCount++
		} else {
			newCount++
		}
	}

	cli.PrintDim(fmt.Sprintf("Docs: %d found (%d new, %d existing)", len(sourceFiles), newCount, updateCount))

	// ─── 3. Dry run: print and exit ────────────────────────────────────────
	if dryRun {
		fmt.Println()
		fmt.Printf("  %s\n", cli.Yellow("Dry run — no files written"))
		cli.PrintDim(fmt.Sprintf("Would call: oracle inject --from %s --to %s", relFrom, relTo))
		if newCount > 0 {
			cli.PrintDim("New files:")
			for _, f := range sourceFiles {
				if !config.FileExists(filepath.Join(pair.To, f)) {
					cli.PrintDim(fmt.Sprintf("  + %s", f))
				}
			}
		}
		cli.PrintDim("Log entry would be appended to .claude/trinity-log.md")
		fmt.Println()
		return true
	}

	// ─── 4. Call oracle inject ─────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "oracle", "inject", "--from", pair.From, "--to", pair.To)
	env := os.Environ()
	env = append(env, "FORCE_COLOR=1")
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if oracle is not found
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			fmt.Printf("\n  %s\n", cli.Red("✗  oracle not found — trinity sync requires oracle to be installed"))
			fmt.Printf("     Install: %s\n", cli.Dim("brew tap 0merUfuk/thematrix && brew install oracle"))
			fmt.Printf("     Or add the the-matrix bin/ directory to your PATH\n\n")
			return false
		}

		fmt.Printf("\n  %s\n", cli.Red("✗  oracle inject failed"))
		if len(output) > 0 {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			// Show last 5 lines
			start := 0
			if len(lines) > 5 {
				start = len(lines) - 5
			}
			for _, line := range lines[start:] {
				cli.PrintDim("  " + line)
			}
		}
		fmt.Println()
		return false
	}

	fmt.Printf("  %s\n", cli.Green(fmt.Sprintf("✓  Injected %d docs", len(sourceFiles))))

	// ─── 5. Read _index.md for confirmation ────────────────────────────────
	indexPath := filepath.Join(pair.To, "_index.md")
	if config.FileExists(indexPath) {
		cli.PrintDim("_index.md written")
	}

	// ─── 6. Append to trinity-log.md ──────────────────────────────────────
	logEntry := BuildLogEntry(pair.Name, relTo, len(sourceFiles), newCount, updateCount)
	logPath := filepath.Join(projectRoot, ".claude", "trinity-log.md")
	if err := AppendLog(logPath, logEntry); err != nil {
		cli.PrintDim(fmt.Sprintf("Warning: could not write log: %v", err))
	} else {
		cli.PrintDim("Logged to .claude/trinity-log.md")
	}
	fmt.Println()

	return true
}
