package morpheus

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
)

// RunStatus shows loop progress at a glance.
func RunStatus() {
	cwd, _ := os.Getwd()

	// Guard
	if !config.FileExists(filepath.Join(cwd, ".autonomous/loop.sh")) {
		cli.PrintDim("Not a morpheus service. Run morpheus status from a scaffolded service directory.")
		os.Exit(0)
	}

	serviceName := filepath.Base(cwd)

	// Tech detection
	techLabel := "Unknown"
	if config.FileExists(filepath.Join(cwd, ".claude/rules/go-service.md")) {
		techLabel = "Go"
	} else if config.FileExists(filepath.Join(cwd, ".claude/rules/node-service.md")) {
		techLabel = "Node.js"
	}

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan(fmt.Sprintf("morpheus status — %s  (%s)", serviceName, techLabel))))

	// Task progress
	done, remaining := countTasks(cwd)
	total := done + remaining
	pct := 0
	if total > 0 {
		pct = cli.Pct(float64(done), float64(total))
	}

	// Cycles
	cyclesRun, lastCycleNum, lastVerdict := parseCycles(cwd)

	// Max cycles from config
	maxCycles := 0
	if cfg, err := config.ReadConfigSh(filepath.Join(cwd, ".autonomous/config.sh")); err == nil {
		maxCycles = cfg.MaxCycles
	}

	// Loop state
	state := determineState(cwd, cyclesRun)

	// Decisions pending
	decisionsPending := countDecisionsPending(cwd)

	// ─── Print Results ────────────────────────────────────────────────────────
	stateDisplay := formatState(state)
	fmt.Printf("  %s\n", cli.Bold("State:    "+stateDisplay))

	cycleLabel := fmt.Sprintf("%d", cyclesRun)
	if maxCycles > 0 {
		cycleLabel = fmt.Sprintf("%d/%d", cyclesRun, maxCycles)
	}
	fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Cycles:   %s run", cycleLabel)))

	// Progress bar
	bar := cli.ProgressBar(done, total, 20)
	fmt.Printf("  %s  %d%%  (%d / %d tasks)\n", bar, pct, done, total)

	// Last review
	if lastCycleNum > 0 {
		verdictStr := cli.Dim("no verdict found")
		if lastVerdict == "APPROVED" {
			verdictStr = cli.Green("APPROVED")
		} else if lastVerdict == "NEEDS_FIX" {
			verdictStr = cli.Yellow("NEEDS_FIX")
		}
		fmt.Printf("  %s\n", cli.Bold(fmt.Sprintf("Review:   Cycle %d: %s", lastCycleNum, verdictStr)))
	}

	// Decisions pending
	if decisionsPending > 0 {
		fmt.Printf("  %s\n", cli.Yellow(fmt.Sprintf("⚠ %d decision(s) pending — check tasks/decisions-pending.md", decisionsPending)))
	}

	// Remaining tasks
	remainingTasks := getRemainingTasks(cwd, 5)
	if len(remainingTasks) > 0 {
		fmt.Printf("\n  %s\n", cli.Bold("Next tasks:"))
		for _, t := range remainingTasks {
			cli.PrintDim("  " + t)
		}
		if remaining > len(remainingTasks) {
			cli.PrintDim(fmt.Sprintf("  ... and %d more", remaining-len(remainingTasks)))
		}
	}

	fmt.Println()
}

func countTasks(cwd string) (done, remaining int) {
	content, err := config.ReadFileString(filepath.Join(cwd, "tasks/todo.md"))
	if err != nil {
		return 0, 0
	}
	doneRe := regexp.MustCompile(`(?im)^- \[x\]`)
	remainingRe := regexp.MustCompile(`(?m)^- \[ \]`)
	done = len(doneRe.FindAllString(content, -1))
	remaining = len(remainingRe.FindAllString(content, -1))
	return
}

func parseCycles(cwd string) (cyclesRun int, lastCycleNum int, lastVerdict string) {
	cyclesDir := filepath.Join(cwd, "tasks/cycles")
	entries, err := os.ReadDir(cyclesDir)
	if err != nil {
		return
	}

	reviewRe := regexp.MustCompile(`cycle-(\d+)-review`)
	var cycleNums []int

	for _, e := range entries {
		if m := reviewRe.FindStringSubmatch(e.Name()); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				cycleNums = append(cycleNums, n)
			}
		}
	}

	if len(cycleNums) == 0 {
		return
	}

	sort.Ints(cycleNums)
	cyclesRun = len(cycleNums)
	lastCycleNum = cycleNums[len(cycleNums)-1]

	// Parse last review for verdict
	for _, e := range entries {
		if strings.Contains(e.Name(), fmt.Sprintf("cycle-%d-review", lastCycleNum)) && strings.HasSuffix(e.Name(), ".md") {
			content, err := config.ReadFileString(filepath.Join(cyclesDir, e.Name()))
			if err == nil {
				if strings.Contains(content, "Verdict: APPROVED") {
					lastVerdict = "APPROVED"
				} else if strings.Contains(content, "Verdict: NEEDS_FIX") {
					lastVerdict = "NEEDS_FIX"
				}
			}
			break
		}
	}

	return
}

func determineState(cwd string, cyclesRun int) string {
	if cyclesRun == 0 {
		return "NOT STARTED"
	}

	// Check if the reviewer approved (loop completed successfully)
	cyclesDir := filepath.Join(cwd, "tasks/cycles")
	if hasReviewerApproval(cyclesDir) {
		return "REVIEWER_APPROVED"
	}

	return "IN PROGRESS"
}

func formatState(state string) string {
	switch state {
	case "NOT STARTED":
		return cli.Dim("NOT STARTED")
	case "IN PROGRESS":
		return cli.Cyan("IN PROGRESS")
	case "REVIEWER_APPROVED":
		return cli.Green("REVIEWER APPROVED")
	default:
		return cli.Dim(state)
	}
}

func countDecisionsPending(cwd string) int {
	content, err := config.ReadFileString(filepath.Join(cwd, "tasks/decisions-pending.md"))
	if err != nil {
		return 0
	}
	re := regexp.MustCompile(`(?m)^- \[ \]`)
	return len(re.FindAllString(content, -1))
}

func getRemainingTasks(cwd string, limit int) []string {
	content, err := config.ReadFileString(filepath.Join(cwd, "tasks/todo.md"))
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`(?m)^- \[ \] .+`)
	matches := re.FindAllString(content, -1)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}
