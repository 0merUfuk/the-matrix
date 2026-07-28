package oracle

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/staleness"
)

// updateScriptTimeout bounds runUpdateScript so a hung Claude subprocess
// cannot block `oracle update` forever. 4 hours is generous — full loops
// typically complete in under 60 minutes.
const updateScriptTimeout = 4 * time.Hour

// validTopicPattern matches safe topic names: alphanumeric start, then
// alphanumeric, hyphens, or underscores. Rejects path traversal attempts.
var validTopicPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// UpdateOpts configures the oracle update command.
type UpdateOpts struct {
	WorkspaceDir string
	KnowledgeDir string
	Topics       []string // parsed from --topics flag (split on comma)
	MaxAgeMonths int
	DryRun       bool
	AutoApprove  bool
}

// RunUpdate performs an incremental knowledge refresh — re-researching only
// specific stale topics instead of the full research loop.
func RunUpdate(opts UpdateOpts) {
	theme := cli.GetTheme()

	fmt.Print(theme.Banner("oracle update", "incremental knowledge refresh"))
	fmt.Println()

	// Step 1: Validate inputs
	resolvedWorkspace, err := filepath.Abs(opts.WorkspaceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Error resolving --workspace path: %v", err)))
		os.Exit(1)
	}
	if err := validateWorkspace(resolvedWorkspace); err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Invalid workspace: %v", err)))
		os.Exit(1)
	}

	resolvedKnowledge, err := filepath.Abs(opts.KnowledgeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Error resolving --knowledge path: %v", err)))
		os.Exit(1)
	}
	if !config.IsDir(resolvedKnowledge) {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Knowledge directory not found: %s", resolvedKnowledge)))
		os.Exit(1)
	}

	cli.PrintDim(fmt.Sprintf("Workspace: %s", cli.PathToHome(resolvedWorkspace)))
	cli.PrintDim(fmt.Sprintf("Knowledge: %s", cli.PathToHome(resolvedKnowledge)))
	fmt.Println()

	// Step 2: Discover topics to update
	var topics []string
	if len(opts.Topics) > 0 {
		topics = opts.Topics
		cli.PrintDim(fmt.Sprintf("Topics (explicit): %s", strings.Join(topics, ", ")))
	} else {
		topics = discoverStaleTopics(resolvedKnowledge, float64(opts.MaxAgeMonths))
		if len(topics) == 0 {
			fmt.Printf("  %s\n\n", cli.Green("All docs are fresh — nothing to update."))
			return
		}
		cli.PrintDim(fmt.Sprintf("Topics (auto-discovered, >%d months): %s", opts.MaxAgeMonths, strings.Join(topics, ", ")))
	}
	fmt.Println()

	// Step 3: Dry run
	if opts.DryRun {
		printDryRunTable(resolvedKnowledge, topics)
		fmt.Printf("\n  %s\n\n", cli.Dim("Run without --dry-run to update."))
		return
	}

	// Step 4: Confirm
	if !opts.AutoApprove {
		fmt.Printf("  Topics to update (%d):\n", len(topics))
		for _, t := range topics {
			fmt.Printf("    %s\n", theme.Info.Render(t))
		}
		fmt.Println()
		fmt.Printf("  %s ", cli.Cyan(fmt.Sprintf("Update %d topic(s)? This will run research cycles and re-synthesize. [y/N]:", len(topics))))
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Println("\n  Aborted.")
			return
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("  Aborted.")
			return
		}
		fmt.Println()
	}

	// Step 5: Scaffold update workspace
	dateStr := time.Now().Format("2006-01-02")
	updateDir := findAvailableUpdateDir(filepath.Join(resolvedWorkspace, "updates"), dateStr)

	fmt.Printf("  Scaffolding update workspace: %s\n", cli.PathToHome(updateDir))

	if err := scaffoldUpdateWorkspace(resolvedWorkspace, updateDir, topics); err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("Failed to scaffold update workspace: %v", err)))
		os.Exit(1)
	}
	fmt.Printf("  %s\n\n", cli.Green("Update workspace ready."))

	// Step 6: Run update.sh
	fmt.Printf("  %s\n\n", cli.Bold(cli.Cyan("Running update loop...")))

	if err := runUpdateScript(updateDir); err != nil {
		fmt.Fprintf(os.Stderr, "\n  %s\n", cli.Red(fmt.Sprintf("Update loop failed: %v", err)))
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Dim(fmt.Sprintf("Check logs: %s", filepath.Join(updateDir, "tasks", "loop.log"))))
		os.Exit(1)
	}

	// Step 7: Inject results
	stackName := readStackNameFromConfig(filepath.Join(updateDir, ".autonomous", "config.sh"))
	if stackName == "" {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red("Could not read STACK_NAME from config.sh"))
		os.Exit(1)
	}

	outputDir := filepath.Join(updateDir, "output", stackName)
	if !config.IsDir(outputDir) {
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Red(fmt.Sprintf("No output produced at %s", outputDir)))
		os.Exit(1)
	}

	injected, err := injectUpdatedDocs(outputDir, resolvedKnowledge, topics)
	if err != nil {
		// injectUpdatedDocs rolls back any partial writes on failure, so the
		// knowledge directory is always either fully updated or unchanged.
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Injection failed: %v", err)))
		fmt.Fprintf(os.Stderr, "  %s\n\n", cli.Dim("Knowledge directory rolled back to prior state."))
		os.Exit(1)
	}

	// Print summary
	fmt.Println()
	fmt.Printf("  %s\n", cli.Bold(cli.Green(fmt.Sprintf("Updated %d topics in %s:", len(injected), cli.PathToHome(resolvedKnowledge)))))
	for _, name := range injected {
		fmt.Printf("    %s\n", theme.Info.Render(name))
	}
	fmt.Println()
}

// discoverStaleTopics reads a knowledge directory and returns topic prefixes
// (filename without .md) for files older than maxAgeMonths.
func discoverStaleTopics(knowledgeDir string, maxAgeMonths float64) []string {
	mdFiles, err := config.ListMDFiles(knowledgeDir)
	if err != nil || len(mdFiles) == 0 {
		return nil
	}

	var stale []string
	for _, filename := range mdFiles {
		filePath := filepath.Join(knowledgeDir, filename)
		content, err := config.ReadFileString(filePath)
		if err != nil {
			continue
		}

		created, ok := staleness.ParseCreatedDate(content)
		if !ok {
			// No date means we can't determine staleness; skip
			continue
		}

		if staleness.IsStale(created, maxAgeMonths) {
			topic := strings.TrimSuffix(filename, ".md")
			stale = append(stale, topic)
		}
	}

	sort.Strings(stale)
	return stale
}

// ParseTopicsFlag splits a comma-separated topic string into a trimmed slice.
// Returns nil for empty input. Validates each topic name against a safe pattern
// to prevent path traversal. Exported for use by cmd/oracle/main.go.
func ParseTopicsFlag(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if !validTopicPattern.MatchString(trimmed) {
			return nil, fmt.Errorf("invalid topic name %q: must match %s", trimmed, validTopicPattern.String())
		}
		result = append(result, trimmed)
	}
	return result, nil
}

// validateWorkspace checks that a directory is a valid oracle research workspace.
// A valid workspace contains .autonomous/config.sh.
func validateWorkspace(workspaceDir string) error {
	if !config.IsDir(workspaceDir) {
		return fmt.Errorf("directory not found: %s", workspaceDir)
	}
	configPath := filepath.Join(workspaceDir, ".autonomous", "config.sh")
	if !config.FileExists(configPath) {
		return fmt.Errorf("config.sh not found at %s — not an oracle workspace", configPath)
	}
	return nil
}

// printDryRunTable displays a table of topics to refresh with their current age.
func printDryRunTable(knowledgeDir string, topics []string) {
	theme := cli.GetTheme()

	fmt.Printf("  %s\n\n", cli.Bold(cli.Cyan(fmt.Sprintf("Dry run: %d topic(s) would be updated", len(topics)))))

	maxLen := 0
	for _, t := range topics {
		if len(t) > maxLen {
			maxLen = len(t)
		}
	}

	for _, topic := range topics {
		padded := topic + strings.Repeat(" ", maxLen-len(topic))
		filename := topic + ".md"
		filePath := filepath.Join(knowledgeDir, filename)

		ageStr := "unknown age"
		if content, err := config.ReadFileString(filePath); err == nil {
			if created, ok := staleness.ParseCreatedDate(content); ok {
				ageStr = staleness.FormatAge(created)
			}
		}

		fmt.Println(theme.StatusLine(cli.Warn, padded, fmt.Sprintf("(%s)", ageStr)))
	}
}

// scaffoldUpdateWorkspace creates the update workspace directory with all
// necessary files copied from the original workspace.
func scaffoldUpdateWorkspace(originalWorkspace, updateDir string, topics []string) error {
	// Create directory structure matching the original workspace layout
	dirs := []string{
		filepath.Join(updateDir, ".autonomous", "prompts"),
		filepath.Join(updateDir, "tasks", "research"),
		filepath.Join(updateDir, "tasks", "review"),
		filepath.Join(updateDir, "tasks", "cycles"),
	}
	for _, d := range dirs {
		if err := config.EnsureDir(d); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Copy files from original workspace
	copyFiles := []struct {
		src string
		dst string
	}{
		{filepath.Join(originalWorkspace, ".autonomous", "config.sh"), filepath.Join(updateDir, ".autonomous", "config.sh")},
		{filepath.Join(originalWorkspace, ".autonomous", "prompts", "researcher.md"), filepath.Join(updateDir, ".autonomous", "prompts", "researcher.md")},
		{filepath.Join(originalWorkspace, ".autonomous", "prompts", "reviewer.md"), filepath.Join(updateDir, ".autonomous", "prompts", "reviewer.md")},
		{filepath.Join(originalWorkspace, ".autonomous", "prompts", "synthesizer.md"), filepath.Join(updateDir, ".autonomous", "prompts", "synthesizer.md")},
		{filepath.Join(originalWorkspace, "CLAUDE.md"), filepath.Join(updateDir, "CLAUDE.md")},
	}

	for _, cf := range copyFiles {
		if config.FileExists(cf.src) {
			if err := config.CopyFile(cf.src, cf.dst, updateDir); err != nil {
				return fmt.Errorf("copying %s: %w", filepath.Base(cf.src), err)
			}
		}
	}

	// Write topics.txt
	topicsContent := strings.Join(topics, "\n") + "\n"
	topicsPath := filepath.Join(updateDir, "topics.txt")
	if err := os.WriteFile(topicsPath, []byte(topicsContent), 0644); err != nil {
		return fmt.Errorf("writing topics.txt: %w", err)
	}

	// Write update.sh from embedded template
	updateShData, err := templateFS.ReadFile("templates/update.sh")
	if err != nil {
		return fmt.Errorf("reading embedded update.sh: %w", err)
	}
	updateShPath := filepath.Join(updateDir, ".autonomous", "update.sh")
	if err := os.WriteFile(updateShPath, updateShData, 0755); err != nil {
		return fmt.Errorf("writing update.sh: %w", err)
	}

	// Write status.txt
	statusPath := filepath.Join(updateDir, "tasks", "status.txt")
	if err := os.WriteFile(statusPath, []byte("INITIALIZED\n"), 0644); err != nil {
		return fmt.Errorf("writing status.txt: %w", err)
	}

	// Create output directory for the stack
	stackName := readStackNameFromConfig(filepath.Join(updateDir, ".autonomous", "config.sh"))
	if stackName != "" {
		outputDir := filepath.Join(updateDir, "output", stackName)
		if err := config.EnsureDir(outputDir); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	return nil
}

// findAvailableUpdateDir returns a non-existing directory path for the update
// workspace. If update-{date} already exists, it appends an incrementing counter:
// update-{date}-2, update-{date}-3, etc.
func findAvailableUpdateDir(updatesDir, dateStr string) string {
	base := filepath.Join(updatesDir, "update-"+dateStr)
	if !config.IsDir(base) {
		return base
	}
	for i := 2; i < 1000; i++ {
		candidate := filepath.Join(updatesDir, fmt.Sprintf("update-%s-%d", dateStr, i))
		if !config.IsDir(candidate) {
			return candidate
		}
	}
	// Fallback: return a unique path using timestamp nanos
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

// runUpdateScript executes the update.sh script in the update workspace.
// A context timeout is applied so a hung subprocess cannot block the caller
// forever; the bash process (and its descendants) are killed when the
// deadline fires. See trinity/sync.go for the pattern this mirrors.
func runUpdateScript(updateDir string) error {
	return runUpdateScriptWithTimeout(updateDir, updateScriptTimeout)
}

// runUpdateScriptWithTimeout is the parameterized variant of runUpdateScript.
// Kept unexported and used by tests to exercise the kill-on-deadline path
// without waiting 4 hours.
func runUpdateScriptWithTimeout(updateDir string, timeout time.Duration) error {
	scriptPath := filepath.Join(updateDir, ".autonomous", "update.sh")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = updateDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Put the bash process (and any children it spawns) into its own process
	// group so we can SIGKILL the entire group on context cancellation. Without
	// this, the Go test runtime's WaitDelay fires because bash's children
	// (e.g. sleep) inherit our Stdout/Stderr fds and keep them open past the
	// parent's death, blocking io.Copy goroutines.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return os.ErrProcessDone
	}
	// Belt-and-suspenders for interpreters that ignore the process-group kill:
	// force the I/O goroutines to unblock after a grace period so cmd.Run
	// always returns cleanly instead of leaving the test binary exit 1 with
	// "WaitDelay expired before I/O complete".
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("update.sh timed out after %s (killed)", timeout)
		}
		return err
	}
	return nil
}

// readStackNameFromConfig extracts STACK_NAME from a config.sh file.
func readStackNameFromConfig(configPath string) string {
	content, err := config.ReadFileString(configPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STACK_NAME=") {
			value := strings.TrimPrefix(line, "STACK_NAME=")
			value = strings.Trim(value, "\"'")
			return value
		}
	}
	return ""
}

// injectUpdatedDocs copies updated docs from the update output directory to the
// knowledge directory. Only copies files matching the specified topics.
// Returns the list of successfully injected topic names.
//
// The function is two-phase: first collect and validate every candidate, then
// write. If any write fails mid-way, the knowledge directory is rolled back to
// its prior contents (pre-existing files restored from their snapshot; newly
// created files removed). This prevents a kill mid-write from leaving the
// knowledge directory in a mixed state.
func injectUpdatedDocs(outputDir, knowledgeDir string, topics []string) ([]string, error) {
	mdFiles, err := config.ListMDFiles(outputDir)
	if err != nil {
		return nil, fmt.Errorf("reading output directory: %w", err)
	}

	// Build a set of topic prefixes for matching
	topicSet := make(map[string]bool, len(topics))
	for _, t := range topics {
		topicSet[t] = true
	}

	// Phase 1: collect and validate every candidate before any write happens.
	type pendingWrite struct {
		filename string
		topic    string
		content  []byte
	}
	var pending []pendingWrite

	for _, filename := range mdFiles {
		topic := strings.TrimSuffix(filename, ".md")
		if !topicSet[topic] {
			continue
		}

		srcPath := filepath.Join(outputDir, filename)
		content, err := config.ReadFileString(srcPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filename, err)
		}

		errs := ValidateInjectDoc(filename, content)
		if len(errs) > 0 {
			cli.PrintWarning(fmt.Sprintf("Skipping %s (validation failed: %s)", filename, strings.Join(errs, "; ")))
			continue
		}

		pending = append(pending, pendingWrite{
			filename: filename,
			topic:    topic,
			content:  []byte(content),
		})
	}

	// Phase 2: snapshot each destination, then write. On any failure, roll
	// back every write we've made so far so the knowledge directory is never
	// left in a mixed state.
	type snapshot struct {
		path     string
		existed  bool
		contents []byte
	}
	var snapshots []snapshot
	var injected []string

	rollback := func() {
		for _, s := range snapshots {
			if s.existed {
				_ = os.WriteFile(s.path, s.contents, 0644)
			} else {
				_ = os.Remove(s.path)
			}
		}
	}

	for _, pw := range pending {
		dstPath := filepath.Join(knowledgeDir, pw.filename)
		snap := snapshot{path: dstPath}
		if existing, rerr := os.ReadFile(dstPath); rerr == nil {
			snap.existed = true
			snap.contents = existing
		} else if !errors.Is(rerr, os.ErrNotExist) {
			rollback()
			return nil, fmt.Errorf("snapshotting %s: %w", pw.filename, rerr)
		}
		snapshots = append(snapshots, snap)

		if err := os.WriteFile(dstPath, pw.content, 0644); err != nil {
			rollback()
			return nil, fmt.Errorf("writing %s to knowledge dir: %w", pw.topic, err)
		}
		injected = append(injected, pw.topic)
	}

	sort.Strings(injected)
	return injected, nil
}
