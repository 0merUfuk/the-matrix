package morpheus

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
)

// RunReport generates a post-loop report from cycle data.
func RunReport() {
	cwd, _ := os.Getwd()

	// Guards
	if !config.FileExists(filepath.Join(cwd, ".autonomous/loop.sh")) {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red("Not a morpheus service."))
		os.Exit(1)
	}
	cyclesDir := filepath.Join(cwd, "tasks/cycles")
	if !config.IsDir(cyclesDir) {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red("No cycle data found. Run the autonomous loop first."))
		os.Exit(1)
	}

	report, err := GenerateLoopReport(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Report generation failed: %v", err)))
		os.Exit(1)
	}

	reportPath := filepath.Join(cwd, "tasks/loop-report.md")
	if err := config.WriteFileString(reportPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "  %s\n", cli.Red(fmt.Sprintf("Failed to write report: %v", err)))
		os.Exit(1)
	}

	fmt.Printf("  %s\n\n", cli.Green("✓ Report written to tasks/loop-report.md"))
}

// ─── Knowledge Gap Detection (Loop A) ────────────────────────────────────────

// topicMapping maps an oracle topic slug to the keywords that signal it needs refreshing.
type topicMapping struct {
	Topic    string
	Keywords []string
}

// defaultTopicMappings covers standard Go service oracle topics.
// Case-insensitive keyword matching against lessons.md content.
var defaultTopicMappings = []topicMapping{
	{"02-layered-architecture", []string{"layer", "architecture", "repository", "domain", "service layer"}},
	{"03-dependency-injection", []string{"wire", "inject", "dependency", "di ", "provider"}},
	{"04-error-handling", []string{"error", "wrap", "sentinel", "errors.as", "errors.is", "panic", "recover"}},
	{"05-testing", []string{"test", "ginkgo", "gomega", "mock", "stub", "fixture", "coverage"}},
	{"06-database", []string{"database", " db ", "postgres", "sql", "bun ", "migration", "query", "orm", "repo "}},
	{"07-configuration", []string{"config", "env ", "yaml", "environment", "secret", "dotenv"}},
	{"08-api-design", []string{"api", "endpoint", "handler", "route", "http", "rest", "request", "response", "status code"}},
	{"09-security", []string{"security", "auth ", "token", "jwt", "password", "permission", "authorization"}},
	{"10-deployment", []string{"docker", "kubernetes", "k8s", "container", "deploy", "build image"}},
	{"11-redis-caching", []string{"redis", "cache", "lua", "expiry", "ttl", "pubsub"}},
	{"12-queue-integration", []string{"queue", "rabbitmq", "amqp", "worker", "consumer", "publish", "message queue"}},
	{"13-observability", []string{"logging", "metric", "trace", "zerolog", "newrelic", "sentry", "observ"}},
}

// knowledgeGap holds a matched topic and the keywords that triggered it.
type knowledgeGap struct {
	Topic   string
	Matched []string // keywords found in lessons
	Lessons int      // number of lesson sections that matched
}

// detectKnowledgeGaps reads tasks/lessons.md and maps lesson content to oracle topics.
// Returns gaps sorted by lesson count (descending), then topic name (ascending).
func detectKnowledgeGaps(cwd string) []knowledgeGap {
	content, err := config.ReadFileString(filepath.Join(cwd, "tasks/lessons.md"))
	if err != nil || strings.TrimSpace(content) == "" {
		return nil
	}

	// Split lessons.md into sections by heading
	sections := splitLessonSections(content)

	lower := strings.ToLower(content)

	type topicScore struct {
		gap   knowledgeGap
		score int
	}
	var scores []topicScore

	for _, tm := range defaultTopicMappings {
		var matched []string
		for _, kw := range tm.Keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				matched = append(matched, strings.TrimSpace(kw))
			}
		}
		if len(matched) == 0 {
			continue
		}
		// Count sections that mention any of the matched keywords
		lessonCount := 0
		for _, sec := range sections {
			secLower := strings.ToLower(sec)
			for _, kw := range matched {
				if strings.Contains(secLower, strings.ToLower(kw)) {
					lessonCount++
					break
				}
			}
		}
		if lessonCount == 0 {
			lessonCount = 1 // at least 1 since content matched
		}
		scores = append(scores, topicScore{
			gap:   knowledgeGap{Topic: tm.Topic, Matched: matched, Lessons: lessonCount},
			score: lessonCount,
		})
	}

	// Sort: lesson count desc, then topic asc
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		return scores[i].gap.Topic < scores[j].gap.Topic
	})

	result := make([]knowledgeGap, len(scores))
	for i, s := range scores {
		result[i] = s.gap
	}
	return result
}

// splitLessonSections splits lessons.md content into sections by ## or ### headings.
func splitLessonSections(content string) []string {
	re := regexp.MustCompile(`(?m)^#{2,3} .+`)
	indices := re.FindAllStringIndex(content, -1)
	if len(indices) == 0 {
		return []string{content}
	}
	var sections []string
	for i, idx := range indices {
		start := idx[0]
		end := len(content)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		sections = append(sections, content[start:end])
	}
	return sections
}

// ─── Report Generation ───────────────────────────────────────────────────────

type cycleOutput struct {
	NumTurns           int                `json:"num_turns"`
	TotalCostUSD       float64            `json:"total_cost_usd"`
	DurationMs         int64              `json:"duration_ms"`
	Subtype            string             `json:"subtype"`
	PermissionDenials  []permissionDenial `json:"permission_denials"`
}

type permissionDenial struct {
	Tool string `json:"tool"`
}

type cycleData struct {
	Num      int
	Dev      *cycleOutput
	Tester   *cycleOutput
	Reviewer *cycleOutput
}

// GenerateLoopReport creates a structured markdown report from cycle JSON files.
func GenerateLoopReport(cwd string) (string, error) {
	serviceName := filepath.Base(cwd)

	// Read loop status
	state := determineState(cwd, 1)

	// Read config
	var maxCycles, devBudget, testBudget, reviewBudget int
	if cfg, err := config.ReadConfigSh(filepath.Join(cwd, ".autonomous/config.sh")); err == nil {
		maxCycles = cfg.MaxCycles
		devBudget = cfg.DeveloperMaxTurns
		testBudget = cfg.TesterMaxTurns
		reviewBudget = cfg.ReviewerMaxTurns
	}

	// Read tasks
	done, remaining := countTasks(cwd)
	total := done + remaining

	// Parse cycle data
	cycles := parseCycleData(filepath.Join(cwd, "tasks/cycles"))

	// Count lessons
	lessonCount := countLessons(cwd)

	// Aggregate stats
	var totalCost float64
	var totalDuration int64
	totalDenials := 0
	devHits, testHits, reviewHits := 0, 0, 0
	var devCost, testCost, reviewCost float64
	denialCounts := map[string]int{}

	for _, c := range cycles {
		if c.Dev != nil {
			totalCost += c.Dev.TotalCostUSD
			totalDuration += c.Dev.DurationMs
			devCost += c.Dev.TotalCostUSD
			if c.Dev.Subtype == "error_max_turns" {
				devHits++
			}
			for _, d := range c.Dev.PermissionDenials {
				denialCounts[d.Tool]++
				totalDenials++
			}
		}
		if c.Tester != nil {
			totalCost += c.Tester.TotalCostUSD
			totalDuration += c.Tester.DurationMs
			testCost += c.Tester.TotalCostUSD
			if c.Tester.Subtype == "error_max_turns" {
				testHits++
			}
			for _, d := range c.Tester.PermissionDenials {
				denialCounts[d.Tool]++
				totalDenials++
			}
		}
		if c.Reviewer != nil {
			totalCost += c.Reviewer.TotalCostUSD
			totalDuration += c.Reviewer.DurationMs
			reviewCost += c.Reviewer.TotalCostUSD
			if c.Reviewer.Subtype == "error_max_turns" {
				reviewHits++
			}
			for _, d := range c.Reviewer.PermissionDenials {
				denialCounts[d.Tool]++
				totalDenials++
			}
		}
	}

	// Build report
	var sb strings.Builder
	now := time.Now().Format("2006-01-02 15:04")

	sb.WriteString(fmt.Sprintf("# Loop Report: %s\n\n", serviceName))
	sb.WriteString(fmt.Sprintf("> Generated by `morpheus report` on %s\n\n", now))
	sb.WriteString("---\n\n")

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Status | %s |\n", state))
	sb.WriteString(fmt.Sprintf("| Cycles completed | %d / %d |\n", len(cycles), maxCycles))
	pct := 0
	if total > 0 {
		pct = int(math.Round(float64(done) / float64(total) * 100))
	}
	sb.WriteString(fmt.Sprintf("| Tasks completed | %d / %d (%d%%) |\n", done, total, pct))
	sb.WriteString(fmt.Sprintf("| Total cost | $%.2f |\n", totalCost))
	sb.WriteString(fmt.Sprintf("| Total duration | %s |\n", cli.FormatDuration(totalDuration)))
	sb.WriteString(fmt.Sprintf("| Permission denials | %d |\n", totalDenials))
	sb.WriteString(fmt.Sprintf("| Lessons captured | %d |\n", lessonCount))
	sb.WriteString("\n---\n\n")

	// Per-cycle breakdown
	if len(cycles) > 0 {
		sb.WriteString("## Per-Cycle Breakdown\n\n")
		sb.WriteString("| Cycle | Dev turns | Test turns | Review turns | Cost | Duration |\n")
		sb.WriteString("|-------|-----------|------------|--------------|------|----------|\n")
		for _, c := range cycles {
			devT := formatTurns(c.Dev)
			testT := formatTurns(c.Tester)
			reviewT := formatTurns(c.Reviewer)
			cost := cycleCost(c)
			dur := cycleDuration(c)
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | $%.2f | %s |\n", c.Num, devT, testT, reviewT, cost, cli.FormatDuration(dur)))
		}
		sb.WriteString("\n---\n\n")
	}

	// Turn budget efficiency
	sb.WriteString("## Turn Budget Efficiency\n\n")
	sb.WriteString("| Role | Budget | Hit count | Hit rate |\n")
	sb.WriteString("|------|--------|-----------|----------|\n")
	writeEfficiency := func(role string, budget, hits, total int) {
		rate := 0
		if total > 0 {
			rate = int(math.Round(float64(hits) / float64(total) * 100))
		}
		warning := ""
		if rate > 50 {
			warning = " ⚠"
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %d / %d | %d%%%s |\n", role, budget, hits, total, rate, warning))
	}
	writeEfficiency("Developer", devBudget, devHits, len(cycles))
	writeEfficiency("Tester", testBudget, testHits, len(cycles))
	writeEfficiency("Reviewer", reviewBudget, reviewHits, len(cycles))
	sb.WriteString("\n---\n\n")

	// Permission denials
	if len(denialCounts) > 0 {
		sb.WriteString("## Permission Denials\n\n")
		sb.WriteString("| Tool | Count |\n")
		sb.WriteString("|------|-------|\n")
		// Sort by count desc
		type toolCount struct {
			Tool  string
			Count int
		}
		var sorted []toolCount
		for tool, count := range denialCounts {
			sorted = append(sorted, toolCount{tool, count})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Count > sorted[j].Count })
		for _, tc := range sorted {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", tc.Tool, tc.Count))
		}
		sb.WriteString("\n---\n\n")
	}

	// Cost by role
	sb.WriteString("## Cost by Role\n\n")
	sb.WriteString("| Role | Cost | Share |\n")
	sb.WriteString("|------|------|-------|\n")
	writeCostRow := func(role string, cost float64) {
		share := 0
		if totalCost > 0 {
			share = int(math.Round(cost / totalCost * 100))
		}
		sb.WriteString(fmt.Sprintf("| %s | $%.2f | %d%% |\n", role, cost, share))
	}
	writeCostRow("Developer", devCost)
	writeCostRow("Tester", testCost)
	writeCostRow("Reviewer", reviewCost)
	sb.WriteString(fmt.Sprintf("| **Total** | **$%.2f** | **100%%** |\n", totalCost))

	// Knowledge gap suggestions (Loop A)
	gaps := detectKnowledgeGaps(cwd)
	sb.WriteString("\n---\n\n")
	sb.WriteString("## Knowledge Gap Suggestions (Loop A)\n\n")
	if len(gaps) == 0 {
		sb.WriteString("No knowledge gaps detected — lessons.md is empty or not found.\n")
	} else {
		sb.WriteString("Lessons from this run hint at the following oracle topics.\n")
		sb.WriteString("Review and run `oracle update` on any that apply.\n\n")
		sb.WriteString("| Oracle Topic | Matched Keywords | Lessons |\n")
		sb.WriteString("|-------------|-----------------|--------|\n")
		var topicSlugs []string
		for _, g := range gaps {
			kws := strings.Join(g.Matched, ", ")
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %d |\n", g.Topic, kws, g.Lessons))
			topicSlugs = append(topicSlugs, g.Topic)
		}
		sb.WriteString("\n**Suggested command:**\n")
		sb.WriteString("```\n")
		sb.WriteString("oracle update --workspace <oracle-workspace> --knowledge .claude/knowledge/go-rules/ \\\n")
		sb.WriteString(fmt.Sprintf("  --topics %s\n", strings.Join(topicSlugs, ",")))
		sb.WriteString("```\n")
		sb.WriteString("\n> **Note**: This mapping uses keyword matching against `tasks/lessons.md` — review before running.\n")
	}

	return sb.String(), nil
}

func parseCycleData(cyclesDir string) []cycleData {
	entries, err := os.ReadDir(cyclesDir)
	if err != nil {
		return nil
	}

	cycleMap := map[int]*cycleData{}
	re := regexp.MustCompile(`cycle-(\d+)-(developer|tester|reviewer)-output\.json`)

	for _, e := range entries {
		m := re.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		num, _ := strconv.Atoi(m[1])
		role := m[2]

		if _, ok := cycleMap[num]; !ok {
			cycleMap[num] = &cycleData{Num: num}
		}

		data, err := os.ReadFile(filepath.Join(cyclesDir, e.Name()))
		if err != nil {
			continue
		}
		var output cycleOutput
		if err := json.Unmarshal(data, &output); err != nil {
			continue
		}

		switch role {
		case "developer":
			cycleMap[num].Dev = &output
		case "tester":
			cycleMap[num].Tester = &output
		case "reviewer":
			cycleMap[num].Reviewer = &output
		}
	}

	// Sort by cycle number
	var result []cycleData
	for _, cd := range cycleMap {
		result = append(result, *cd)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Num < result[j].Num })
	return result
}

func formatTurns(output *cycleOutput) string {
	if output == nil {
		return "—"
	}
	s := strconv.Itoa(output.NumTurns)
	if output.Subtype == "error_max_turns" {
		s += "*"
	}
	return s
}

func cycleCost(c cycleData) float64 {
	cost := 0.0
	if c.Dev != nil {
		cost += c.Dev.TotalCostUSD
	}
	if c.Tester != nil {
		cost += c.Tester.TotalCostUSD
	}
	if c.Reviewer != nil {
		cost += c.Reviewer.TotalCostUSD
	}
	return cost
}

func cycleDuration(c cycleData) int64 {
	var dur int64
	if c.Dev != nil {
		dur += c.Dev.DurationMs
	}
	if c.Tester != nil {
		dur += c.Tester.DurationMs
	}
	if c.Reviewer != nil {
		dur += c.Reviewer.DurationMs
	}
	return dur
}

func countLessons(cwd string) int {
	content, err := config.ReadFileString(filepath.Join(cwd, "tasks/lessons.md"))
	if err != nil {
		return 0
	}
	re := regexp.MustCompile(`(?m)^#{2,3} `)
	return len(re.FindAllString(content, -1))
}
