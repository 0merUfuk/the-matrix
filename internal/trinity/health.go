package trinity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/staleness"
)

// HealthOpts configures the trinity health command.
type HealthOpts struct {
	ProjectDir   string
	MaxAgeMonths float64
}

// Results tracks pass/fail/warn counts across all health checks.
type Results struct {
	Pass int
	Fail int
	Warn int
}

// RunHealth runs the trinity health command: read-only ecosystem validation.
// Exit codes: 0 = healthy (or warnings only), 1 = any critical failure.
func RunHealth(opts HealthOpts) {
	root, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		root = opts.ProjectDir
	}
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}
	relRoot := cli.PathToHome(root)

	theme := cli.GetTheme()

	fmt.Print(theme.Banner("trinity health", relRoot))
	fmt.Println()

	results := &Results{}

	// 1. .claude/ directory exists
	claudeDir := filepath.Join(root, ".claude")
	if !config.IsDir(claudeDir) {
		fmt.Println(theme.StatusLine(cli.Fail, ".claude/", "missing"))
		results.Fail++
		printSummary(theme, results)
		os.Exit(1)
	}
	fmt.Println(theme.StatusLine(cli.Pass, ".claude/", "exists"))
	results.Pass++

	// 2. CLAUDE.md exists at project root
	claudeMd := filepath.Join(root, "CLAUDE.md")
	if config.FileExists(claudeMd) {
		fmt.Println(theme.StatusLine(cli.Pass, "CLAUDE.md", "exists"))
		results.Pass++
	} else {
		fmt.Println(theme.StatusLine(cli.Fail, "CLAUDE.md", "missing"))
		results.Fail++
	}

	// 3. .claude/knowledge/ directory exists
	// Not all projects use oracle-driven knowledge — only warn, do not fail.
	knowledgeDir := filepath.Join(claudeDir, "knowledge")
	if !config.IsDir(knowledgeDir) {
		fmt.Println(theme.StatusLine(cli.Warn, ".claude/knowledge/", "missing — run oracle research + trinity sync if you want stack knowledge docs"))
		results.Warn++
		checkContextFiles(theme, root, results)
		checkAgentSkillEcosystem(theme, root, results)
		checkTrinityLog(theme, root, results)
		printSummary(theme, results)
		os.Exit(exitCode(results))
	}
	fmt.Println(theme.StatusLine(cli.Pass, ".claude/knowledge/", "exists"))
	results.Pass++

	// 4. Scan knowledge subdirectories
	entries, err := os.ReadDir(knowledgeDir)
	if err != nil {
		fmt.Println(theme.StatusLine(cli.Fail, ".claude/knowledge/", fmt.Sprintf("read error: %v", err)))
		results.Fail++
	} else {
		var subdirs []string
		for _, e := range entries {
			if e.IsDir() {
				subdirs = append(subdirs, e.Name())
			}
		}
		sort.Strings(subdirs)

		if len(subdirs) == 0 {
			fmt.Println(theme.StatusLine(cli.Warn, ".claude/knowledge/", "no subdirectories found"))
			results.Warn++
		}

		for _, name := range subdirs {
			checkKnowledgeDir(theme, filepath.Join(knowledgeDir, name), name, opts.MaxAgeMonths, results)
		}
	}

	// 5. Context files
	checkContextFiles(theme, root, results)

	// 6. Agent & skill ecosystem
	checkAgentSkillEcosystem(theme, root, results)

	// 7. Trinity log
	checkTrinityLog(theme, root, results)

	// Summary
	printSummary(theme, results)
	os.Exit(exitCode(results))
}

// checkKnowledgeDir validates a single knowledge subdirectory.
func checkKnowledgeDir(theme *cli.Theme, dirPath, name string, maxAgeMonths float64, results *Results) {
	relPath := fmt.Sprintf(".claude/knowledge/%s/", name)

	mdFiles, err := config.ListMDFiles(dirPath)
	if err != nil {
		fmt.Println(theme.StatusLine(cli.Fail, relPath, fmt.Sprintf("read error: %v", err)))
		results.Fail++
		return
	}

	if len(mdFiles) == 0 {
		fmt.Println(theme.StatusLine(cli.Warn, relPath, "empty — no .md files"))
		results.Warn++
		return
	}

	fmt.Println(theme.StatusLine(cli.Pass, relPath, fmt.Sprintf("%d docs", len(mdFiles))))
	results.Pass++

	// Check _index.md
	indexPath := filepath.Join(dirPath, "_index.md")
	if config.FileExists(indexPath) {
		fmt.Println(theme.StatusLine(cli.Pass, relPath+"_index.md", "exists"))
		results.Pass++
	} else {
		fmt.Println(theme.StatusLine(cli.Warn, relPath+"_index.md", "missing"))
		results.Warn++
	}

	// Validate each doc
	for _, filename := range mdFiles {
		filePath := filepath.Join(dirPath, filename)
		content, err := config.ReadFileString(filePath)
		if err != nil {
			fmt.Println(theme.StatusLine(cli.Fail, "  "+filename, fmt.Sprintf("read error: %v", err)))
			results.Fail++
			continue
		}

		// Frontmatter + EJS validation
		errors := ValidateDoc(filename, content)
		if len(errors) > 0 {
			fmt.Println(theme.StatusLine(cli.Fail, "  "+filename, errors[0]))
			results.Fail++
			continue
		}

		// Staleness check
		created, ok := staleness.ParseCreatedDate(content)
		if !ok {
			fmt.Println(theme.StatusLine(cli.Warn, "  "+filename, "no Created date"))
			results.Warn++
			continue
		}

		ageStr := staleness.FormatAge(created)
		dateStr := created.Format("2006-01-02")

		if staleness.IsStale(created, maxAgeMonths) {
			fmt.Println(theme.StatusLine(cli.Warn, "  "+filename, fmt.Sprintf("stale (%s, %s)", dateStr, ageStr)))
			results.Warn++
		} else {
			fmt.Println(theme.StatusLine(cli.Pass, "  "+filename, fmt.Sprintf("fresh (%s, %s)", dateStr, ageStr)))
			results.Pass++
		}
	}
}

// checkContextFiles verifies that key .claude/ context files exist.
func checkContextFiles(theme *cli.Theme, root string, results *Results) {
	contextFiles := []struct {
		rel   string
		label string
	}{
		{".claude/SERVICE_CONTEXT.md", "SERVICE_CONTEXT.md"},
		{".claude/NEXT_STEPS.md", "NEXT_STEPS.md"},
	}

	for _, cf := range contextFiles {
		filePath := filepath.Join(root, cf.rel)
		if config.FileExists(filePath) {
			fmt.Println(theme.StatusLine(cli.Pass, cf.label, "exists"))
			results.Pass++
		} else {
			fmt.Println(theme.StatusLine(cli.Warn, cf.label, "missing"))
			results.Warn++
		}
	}
}

// checkTrinityLog verifies the trinity sync log exists.
func checkTrinityLog(theme *cli.Theme, root string, results *Results) {
	logPath := filepath.Join(root, ".claude", "trinity-log.md")
	if config.FileExists(logPath) {
		fmt.Println(theme.StatusLine(cli.Pass, "trinity-log.md", "exists"))
		results.Pass++
	} else {
		fmt.Println(theme.StatusLine(cli.Warn, "trinity-log.md", "missing — trinity sync has not been run"))
		results.Warn++
	}
}

// ExpectedAgents is the canonical list of agents the-matrix ecosystem ships.
// Keep aligned with `.claude/agents/` and CLAUDE.md "Agent Roster" table.
var ExpectedAgents = []string{
	"manager.md", "developer.md", "tester.md", "reviewer.md",
	"strategist.md", "security-reviewer.md",
	"architect.md", "product-lead.md", "tech-lead.md", "growth-lead.md",
	"self-improver.md",
}

// ExpectedSkills is the canonical list of skills the-matrix ecosystem ships.
// Keep aligned with `.claude/skills/` and CLAUDE.md skill list.
var ExpectedSkills = []string{
	"audit", "commit", "dep-audit", "doublecheck", "fix",
	"issue", "owasp-review", "provision", "release", "secret-scan",
	"security-scan", "self-improve", "self-state", "session-learn",
	"strategy-monthly", "strategy-weekly", "taste-calibration",
}

// checkAgentSkillEcosystem verifies expected agents and skills are present.
func checkAgentSkillEcosystem(theme *cli.Theme, root string, results *Results) {
	claudeDir := filepath.Join(root, ".claude")

	agentsDir := filepath.Join(claudeDir, "agents")
	if config.IsDir(agentsDir) {
		for _, agent := range ExpectedAgents {
			p := filepath.Join(agentsDir, agent)
			label := ".claude/agents/" + agent
			if config.FileExists(p) {
				fmt.Println(theme.StatusLine(cli.Pass, label, "exists"))
				results.Pass++
			} else {
				fmt.Println(theme.StatusLine(cli.Warn, label, "missing — re-provision with neo init"))
				results.Warn++
			}
		}
	} else {
		fmt.Println(theme.StatusLine(cli.Warn, ".claude/agents/", "missing — re-provision with neo init"))
		results.Warn++
	}

	skillsDir := filepath.Join(claudeDir, "skills")
	if config.IsDir(skillsDir) {
		for _, skill := range ExpectedSkills {
			dir := filepath.Join(skillsDir, skill)
			label := ".claude/skills/" + skill + "/"
			if config.IsDir(dir) {
				fmt.Println(theme.StatusLine(cli.Pass, label, "exists"))
				results.Pass++
			} else {
				fmt.Println(theme.StatusLine(cli.Warn, label, "missing — re-provision with neo init"))
				results.Warn++
			}
		}
	} else {
		fmt.Println(theme.StatusLine(cli.Warn, ".claude/skills/", "missing — re-provision with neo init"))
		results.Warn++
	}
}

// printSummary renders the final pass/fail/warn summary.
func printSummary(theme *cli.Theme, results *Results) {
	fmt.Println()
	fmt.Println(theme.Summary(results.Pass, results.Fail, results.Warn))

	if results.Warn > 0 {
		fmt.Println()
		fmt.Println(theme.WarnBox("Some checks need attention",
			"Run: trinity refresh --stale-only to update stale docs",
			"Run: trinity sync to regenerate missing _index.md",
			"Run: neo init to re-provision missing agents/skills"))
	}
	fmt.Println()
}

func exitCode(results *Results) int {
	if results.Fail > 0 {
		return 1
	}
	return 0
}
