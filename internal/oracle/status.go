package oracle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/staleness"
)

// StatusOpts configures the oracle status command.
type StatusOpts struct {
	KnowledgeDir string
	MaxAgeMonths float64
}

// RunStatus shows loop progress or knowledge freshness.
func RunStatus(opts StatusOpts) {
	if opts.KnowledgeDir != "" {
		runKnowledgeFreshness(opts.KnowledgeDir, opts.MaxAgeMonths)
		return
	}

	cwd, _ := os.Getwd()
	theme := cli.GetTheme()

	// Check we're in an oracle workspace
	if !config.FileExists(filepath.Join(cwd, ".autonomous", "config.sh")) {
		fmt.Fprintln(os.Stderr, theme.ErrorBox("Not an oracle workspace",
			"Run oracle status from an output directory.",
			"To check knowledge freshness: oracle status --knowledge <dir>"))
		os.Exit(1)
	}

	fmt.Print(theme.Banner("oracle status", ""))

	// Status file
	statusFile := filepath.Join(cwd, "tasks", "status.txt")
	status := "UNKNOWN"
	if content, err := config.ReadFileString(statusFile); err == nil {
		status = strings.TrimSpace(content)
	}
	fmt.Println(theme.KeyValue([]cli.KV{{Key: "Status", Value: formatStatus(theme, status)}}))

	// Research cycle files
	researchDir := filepath.Join(cwd, "tasks", "research")
	var researchFiles []string
	if config.IsDir(researchDir) {
		if files, err := config.ListMDFiles(researchDir); err == nil {
			researchFiles = files
		}
	}

	reviewDir := filepath.Join(cwd, "tasks", "review")
	var reviewFiles []string
	if config.IsDir(reviewDir) {
		if files, err := config.ListMDFiles(reviewDir); err == nil {
			reviewFiles = files
		}
	}

	fmt.Println(theme.SectionHeader("Research cycles"))
	for _, cg := range CycleGroups {
		// Check if review file exists for this cycle
		hasReview := false
		for _, rf := range reviewFiles {
			if strings.Contains(rf, fmt.Sprintf("cycle-%d", cg.Num)) {
				hasReview = true
				break
			}
		}

		// Count topics written
		topicsWritten := 0
		for _, rf := range researchFiles {
			for _, t := range cg.Topics {
				if strings.HasPrefix(rf, t) {
					topicsWritten++
					break
				}
			}
		}

		var level cli.Level
		if hasReview {
			level = cli.Pass
		} else if topicsWritten > 0 {
			level = cli.Warn
		} else {
			level = cli.Fail
		}
		label := fmt.Sprintf("Cycle %d: %s", cg.Num, cg.Label)
		detail := fmt.Sprintf("(%d/%d topics written)", topicsWritten, len(cg.Topics))
		fmt.Println(theme.StatusLine(level, label, detail))
	}

	// Human gate
	gateFile := filepath.Join(cwd, "tasks", "SYNTHESIZE_APPROVED")
	gateApproved := config.FileExists(gateFile)
	gateValue := theme.Warning.Render("Waiting — touch tasks/SYNTHESIZE_APPROVED to continue")
	if gateApproved {
		gateValue = theme.Success.Render("APPROVED")
	}
	fmt.Println()
	fmt.Println(theme.KeyValue([]cli.KV{{Key: "Human gate", Value: gateValue}}))

	// Output docs
	outputDir := filepath.Join(cwd, "output")
	if config.IsDir(outputDir) {
		entries, err := os.ReadDir(outputDir)
		if err == nil {
			type outputInfo struct {
				Name  string
				Count int
			}
			var outputs []outputInfo
			for _, e := range entries {
				if e.IsDir() {
					entryPath := filepath.Join(outputDir, e.Name())
					files, _ := config.ListMDFiles(entryPath)
					outputs = append(outputs, outputInfo{Name: e.Name(), Count: len(files)})
				}
			}
			if len(outputs) > 0 {
				fmt.Println(theme.SectionHeader("Output"))
				for _, o := range outputs {
					var level cli.Level
					if o.Count >= 17 {
						level = cli.Pass
					} else if o.Count > 0 {
						level = cli.Warn
					} else {
						level = cli.Fail
					}
					label := fmt.Sprintf("output/%s/", o.Name)
					detail := fmt.Sprintf("(%d docs)", o.Count)
					fmt.Println(theme.StatusLine(level, label, detail))
				}
			}
		}
	}

	// Log tail
	logFile := filepath.Join(cwd, "tasks", "loop.log")
	if content, err := config.ReadFileString(logFile); err == nil {
		lines := strings.Split(strings.TrimSpace(content), "\n")
		start := 0
		if len(lines) > 5 {
			start = len(lines) - 5
		}
		fmt.Println(theme.SectionHeader("Last log entries"))
		for _, l := range lines[start:] {
			fmt.Printf("  %s\n", theme.Muted.Render(l))
		}
	}

	fmt.Println()
}

func formatStatus(theme *cli.Theme, status string) string {
	if entry, ok := StatusMap[status]; ok {
		switch entry.Color {
		case "dim":
			return theme.Muted.Render(entry.Text)
		case "yellow":
			return theme.Warning.Render(entry.Text)
		case "cyan", "magenta":
			return theme.Info.Render(entry.Text)
		case "green":
			return theme.Success.Render(entry.Text)
		case "red":
			return theme.Error.Render(entry.Text)
		}
		return entry.Text
	}
	return theme.Muted.Render(status)
}

// runKnowledgeFreshness scans a knowledge directory and reports doc freshness.
func runKnowledgeFreshness(knowledgeDir string, maxAgeMonths float64) {
	resolvedDir, _ := filepath.Abs(knowledgeDir)
	theme := cli.GetTheme()

	if !config.IsDir(resolvedDir) {
		fmt.Fprintln(os.Stderr, theme.ErrorBox(fmt.Sprintf("Knowledge directory not found: %s", resolvedDir)))
		os.Exit(1)
	}

	relDir := cli.PathToHome(resolvedDir)
	fmt.Print(theme.Banner("oracle freshness", relDir))
	fmt.Println()

	mdFiles, err := config.ListMDFiles(resolvedDir)
	if err != nil || len(mdFiles) == 0 {
		fmt.Println(theme.WarnBox(fmt.Sprintf("No .md files found in %s", relDir)))
		os.Exit(0)
	}

	type result struct {
		Filename string
		IsStale  *bool // nil = unknown
		DateStr  string
		AgeStr   string
	}

	var results []result
	maxFilename := 0

	for _, filename := range mdFiles {
		if len(filename) > maxFilename {
			maxFilename = len(filename)
		}

		filePath := filepath.Join(resolvedDir, filename)
		content, err := config.ReadFileString(filePath)
		if err != nil {
			results = append(results, result{Filename: filename, DateStr: "?", AgeStr: "read error"})
			continue
		}

		created, ok := staleness.ParseCreatedDate(content)
		if !ok {
			results = append(results, result{Filename: filename, DateStr: "?", AgeStr: "no Created date"})
			continue
		}

		age := staleness.MonthsAgo(created)
		isStale := age >= maxAgeMonths
		dateStr := created.Format("2006-01-02")
		ageStr := staleness.FormatAge(created)
		results = append(results, result{Filename: filename, IsStale: &isStale, DateStr: dateStr, AgeStr: ageStr})
	}

	// Display table
	freshCount := 0
	staleCount := 0
	unknownCount := 0
	var staleFiles []string

	for _, r := range results {
		padded := r.Filename + strings.Repeat(" ", maxFilename-len(r.Filename))

		if r.IsStale == nil {
			fmt.Println(theme.StatusLine(cli.Warn, padded, fmt.Sprintf("%s (%s)", r.DateStr, r.AgeStr)))
			unknownCount++
		} else if *r.IsStale {
			fmt.Println(theme.StatusLine(cli.Warn, padded, fmt.Sprintf("%s (%s — stale)", r.DateStr, r.AgeStr)))
			staleCount++
			staleFiles = append(staleFiles, strings.TrimSuffix(r.Filename, ".md"))
		} else {
			fmt.Println(theme.StatusLine(cli.Pass, padded, fmt.Sprintf("%s (%s)", r.DateStr, r.AgeStr)))
			freshCount++
		}
	}

	// Summary
	total := len(results)
	parts := []string{fmt.Sprintf("%d docs total", total)}
	if freshCount > 0 {
		parts = append(parts, theme.Success.Render(fmt.Sprintf("%d fresh", freshCount)))
	}
	if staleCount > 0 {
		parts = append(parts, theme.Warning.Render(fmt.Sprintf("%d stale", staleCount)))
	}
	if unknownCount > 0 {
		parts = append(parts, theme.Muted.Render(fmt.Sprintf("%d no date", unknownCount)))
	}

	fmt.Printf("\n  %s\n", strings.Join(parts, " — "))
	fmt.Printf("  %s\n", theme.Muted.Render(fmt.Sprintf("Threshold: %g months", maxAgeMonths)))

	if len(staleFiles) > 0 {
		fmt.Println()
		fmt.Println(theme.WarnBox("Stale topics: "+strings.Join(staleFiles, ", "),
			"Re-run oracle research to refresh these docs, then oracle inject."))
	}

	fmt.Println()
}
