package morpheus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
)

// hasReviewerApproval checks if the latest review file in the cycles directory
// contains an APPROVED verdict. Only the most recent cycle is checked.
func hasReviewerApproval(cyclesDir string) bool {
	matches, err := filepath.Glob(filepath.Join(cyclesDir, "cycle-*-review.md"))
	if err != nil || len(matches) == 0 {
		return false
	}

	// Sort descending — check the latest review first
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	// Check only the latest (most recent) cycle's review file
	content, err := config.ReadFileString(matches[0])
	if err != nil {
		return false
	}
	return hasVerdict(content, "APPROVED")
}

// hasDualApproval confirms that reviewer and security approvals come from the
// same latest review cycle.
func hasDualApproval(cyclesDir string) bool {
	reviewMatches, err := filepath.Glob(filepath.Join(cyclesDir, "cycle-*-review.md"))
	if err != nil || len(reviewMatches) == 0 {
		return false
	}

	sort.Sort(sort.Reverse(sort.StringSlice(reviewMatches)))
	latestReview := reviewMatches[0]
	reviewContent, err := config.ReadFileString(latestReview)
	if err != nil || !hasVerdict(reviewContent, "APPROVED") {
		return false
	}

	cycle, ok := cycleNumber(filepath.Base(latestReview), "review")
	if !ok {
		return false
	}
	securityPath := filepath.Join(cyclesDir, fmt.Sprintf("cycle-%s-security.md", cycle))
	securityContent, err := config.ReadFileString(securityPath)
	if err != nil {
		return false
	}
	return hasVerdict(securityContent, "SECURITY_APPROVED")
}

func hasVerdict(content, verdict string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## Verdict:") && strings.Contains(line, verdict) {
			return true
		}
	}
	return false
}

func cycleNumber(filename, kind string) (string, bool) {
	prefix := "cycle-"
	suffix := "-" + kind + ".md"
	if !strings.HasPrefix(filename, prefix) || !strings.HasSuffix(filename, suffix) {
		return "", false
	}
	cycle := strings.TrimSuffix(strings.TrimPrefix(filename, prefix), suffix)
	if cycle == "" || strings.ContainsAny(cycle, `/\\`) {
		return "", false
	}
	return cycle, true
}

// RunFinalize populates runtime context files after the loop completes.
func RunFinalize() {
	cwd, _ := os.Getwd()

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("morp finalize")))

	// Guard 1: latest cycle must have both reviewer and security approval
	if !hasDualApproval(filepath.Join(cwd, "tasks/cycles")) {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red("Loop has not completed. The latest cycle requires both reviewer APPROVED and security SECURITY_APPROVED verdicts."))
		cli.PrintDim("Run the autonomous loop first: bash .autonomous/loop.sh")
		fmt.Println()
		os.Exit(1)
	}

	// Guard 2: finalizer.sh must exist
	finalizerPath := filepath.Join(cwd, ".autonomous/finalizer.sh")
	if !config.FileExists(finalizerPath) {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red("Missing .autonomous/finalizer.sh — was this project scaffolded with morp init?"))
		fmt.Println()
		os.Exit(1)
	}

	// Run finalizer.sh with inherited stdio
	cli.PrintDim("Running .autonomous/finalizer.sh...")
	fmt.Println()

	cmd := exec.Command("bash", finalizerPath) // #nosec G204 -- argv-only exec of bash on the project's .autonomous/finalizer.sh, no shell string
	cmd.Dir = cwd
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("\n  %s\n\n", cli.Red(fmt.Sprintf("Finalizer failed: %v", err)))
		os.Exit(1)
	}

	// Best-effort: generate loop report
	cyclesDir := filepath.Join(cwd, "tasks/cycles")
	if config.IsDir(cyclesDir) {
		report, err := GenerateLoopReport(cwd)
		if err == nil && report != "" {
			reportPath := filepath.Join(cwd, "tasks/loop-report.md")
			if writeErr := config.WriteFileString(reportPath, report); writeErr == nil {
				cli.PrintDim("Loop report written to tasks/loop-report.md")
			}
		} else if err != nil {
			cli.PrintDim(fmt.Sprintf("Warning: loop report generation failed: %v", err))
		}
	}

	// Summary
	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Green("✓ Finalize complete")))
	cli.PrintDim("Review these files:")
	for _, reviewFile := range finalizeReviewFiles(cwd) {
		cli.PrintDim(fmt.Sprintf("  %-27s — %s", reviewFile.path, reviewFile.description))
	}
	fmt.Println()
}

type finalizeReviewFile struct {
	path        string
	description string
}

func finalizeReviewFiles(cwd string) []finalizeReviewFile {
	rt := agentruntime.Registry.Get(agentruntime.DetectRuntime(cwd))
	if rt == nil {
		return nil
	}
	contextFiles := rt.ContextFiles()
	if contextFiles.ContextDir == "" {
		if contextFiles.PrimaryContext == "" {
			return nil
		}
		return []finalizeReviewFile{{path: contextFiles.PrimaryContext, description: "project context"}}
	}
	return []finalizeReviewFile{
		{path: filepath.Join(contextFiles.ContextDir, "SERVICE_CONTEXT.md"), description: "current state"},
		{path: filepath.Join(contextFiles.ContextDir, "DECISIONS.md"), description: "architectural decisions"},
		{path: filepath.Join(contextFiles.ContextDir, "KNOWN_ISSUES.md"), description: "open issues"},
		{path: filepath.Join(contextFiles.ContextDir, "NEXT_STEPS.md"), description: "remaining work"},
	}
}
