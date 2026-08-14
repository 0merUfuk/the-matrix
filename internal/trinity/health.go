package trinity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/matrixassets"
	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
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
	rt := agentruntime.Registry.Get(agentruntime.DetectRuntime(root))
	if rt == nil {
		rt = agentruntime.Registry.Default()
	}
	contextSpec := rt.ContextFiles()

	// 1. Runtime context directory exists when the runtime requires one.
	if contextSpec.ContextDir != "" {
		contextDir := filepath.Join(root, contextSpec.ContextDir)
		contextDirLabel := filepath.ToSlash(contextSpec.ContextDir) + "/"
		if !config.IsDir(contextDir) {
			fmt.Println(theme.StatusLine(cli.Fail, contextDirLabel, "missing"))
			results.Fail++
			printSummaryForRuntime(theme, results, rt)
			os.Exit(1)
		}
		fmt.Println(theme.StatusLine(cli.Pass, contextDirLabel, "exists"))
		results.Pass++
	}

	// 2. Primary runtime context exists at project root.
	primaryContext := filepath.Join(root, contextSpec.PrimaryContext)
	if config.FileExists(primaryContext) {
		fmt.Println(theme.StatusLine(cli.Pass, contextSpec.PrimaryContext, "exists"))
		results.Pass++
	} else {
		fmt.Println(theme.StatusLine(cli.Fail, contextSpec.PrimaryContext, "missing"))
		results.Fail++
	}

	// 3. Runtime knowledge directory exists.
	// Not all projects use oracle-driven knowledge — only warn, do not fail.
	knowledgeDir := filepath.Join(root, contextSpec.KnowledgeDir)
	knowledgeLabel := filepath.ToSlash(contextSpec.KnowledgeDir) + "/"
	if contextSpec.KnowledgeDir != "" && !config.IsDir(knowledgeDir) {
		fmt.Println(theme.StatusLine(cli.Warn, knowledgeLabel, "missing — run oracle research + trinity sync if you want stack knowledge docs"))
		results.Warn++
		checkContextFilesForRuntime(theme, root, contextSpec, results)
		checkAgentSkillEcosystemForRuntime(theme, root, rt, results)
		checkTrinityLogForRuntime(theme, root, contextSpec, results)
		printSummaryForRuntime(theme, results, rt)
		os.Exit(exitCode(results))
	}
	if contextSpec.KnowledgeDir != "" {
		fmt.Println(theme.StatusLine(cli.Pass, knowledgeLabel, "exists"))
		results.Pass++
	}

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
			checkKnowledgeDirAt(theme, filepath.Join(knowledgeDir, name), knowledgeLabel, name, opts.MaxAgeMonths, results)
		}
	}

	// 5. Context files
	checkContextFilesForRuntime(theme, root, contextSpec, results)

	// 6. Agent & skill ecosystem
	checkAgentSkillEcosystemForRuntime(theme, root, rt, results)

	// 7. Trinity log
	checkTrinityLogForRuntime(theme, root, contextSpec, results)

	// Summary
	printSummaryForRuntime(theme, results, rt)
	os.Exit(exitCode(results))
}

// checkKnowledgeDir validates a single knowledge subdirectory.
func checkKnowledgeDir(theme *cli.Theme, dirPath, name string, maxAgeMonths float64, results *Results) {
	checkKnowledgeDirAt(theme, dirPath, ".claude/knowledge/", name, maxAgeMonths, results)
}

func checkKnowledgeDirAt(theme *cli.Theme, dirPath, knowledgeLabel, name string, maxAgeMonths float64, results *Results) {
	relPath := fmt.Sprintf("%s%s/", knowledgeLabel, name)

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
	rt := agentruntime.Registry.Get("claude")
	checkContextFilesForRuntime(theme, root, rt.ContextFiles(), results)
}

func checkContextFilesForRuntime(theme *cli.Theme, root string, spec agentruntime.ContextFileSpec, results *Results) {
	if spec.ContextDir == "" {
		return
	}

	contextFiles := []struct {
		label string
	}{
		{"SERVICE_CONTEXT.md"},
		{"NEXT_STEPS.md"},
	}

	for _, cf := range contextFiles {
		filePath := filepath.Join(root, spec.ContextDir, cf.label)
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
	rt := agentruntime.Registry.Get("claude")
	checkTrinityLogForRuntime(theme, root, rt.ContextFiles(), results)
}

func checkTrinityLogForRuntime(theme *cli.Theme, root string, spec agentruntime.ContextFileSpec, results *Results) {
	logDir := spec.ContextDir
	if logDir == "" && spec.KnowledgeDir != "" {
		logDir = filepath.Dir(spec.KnowledgeDir)
	}
	if logDir == "" || logDir == "." {
		return
	}

	logPath := filepath.Join(root, logDir, "trinity-log.md")
	if config.FileExists(logPath) {
		fmt.Println(theme.StatusLine(cli.Pass, "trinity-log.md", "exists"))
		results.Pass++
	} else {
		fmt.Println(theme.StatusLine(cli.Warn, "trinity-log.md", "missing — trinity sync has not been run"))
		results.Warn++
	}
}

var ExpectedAgents = matrixassets.ExpectedAgents
var ExpectedSkills = matrixassets.ExpectedSkills

// checkAgentSkillEcosystem verifies expected agents and skills are present.
func checkAgentSkillEcosystem(theme *cli.Theme, root string, results *Results) {
	rt := agentruntime.Registry.Get("claude")
	checkAgentSkillEcosystemForRuntime(theme, root, rt, results)
}

func checkAgentSkillEcosystemForRuntime(theme *cli.Theme, root string, rt agentruntime.Runtime, results *Results) {
	spec := rt.ContextFiles()
	capabilities := rt.Capabilities()

	if !capabilities.AgentDefinitions || spec.AgentDir == "" {
		fmt.Println("  " + cli.Dim("Agent definitions: not provisioned for "+rt.DisplayName()))
	} else {
		agentsDir := filepath.Join(root, spec.AgentDir)
		agentDirLabel := filepath.ToSlash(spec.AgentDir) + "/"
		if config.IsDir(agentsDir) {
			for _, agent := range ExpectedAgents {
				p := filepath.Join(agentsDir, agent)
				label := filepath.ToSlash(filepath.Join(spec.AgentDir, agent))
				if config.FileExists(p) {
					fmt.Println(theme.StatusLine(cli.Pass, label, "exists"))
					results.Pass++
				} else {
					fmt.Println(theme.StatusLine(cli.Warn, label, "missing — re-provision with neo init"))
					results.Warn++
				}
			}
		} else {
			fmt.Println(theme.StatusLine(cli.Warn, agentDirLabel, "missing — re-provision with neo init"))
			results.Warn++
		}
	}

	if !capabilities.SkillDefinitions || spec.SkillDir == "" {
		fmt.Println("  " + cli.Dim("Skill definitions: not provisioned for "+rt.DisplayName()))
	} else {
		skillsDir := filepath.Join(root, spec.SkillDir)
		skillDirLabel := filepath.ToSlash(spec.SkillDir) + "/"
		if config.IsDir(skillsDir) {
			for _, skill := range ExpectedSkills {
				dir := filepath.Join(skillsDir, skill)
				label := filepath.ToSlash(filepath.Join(spec.SkillDir, skill)) + "/"
				if config.IsDir(dir) {
					fmt.Println(theme.StatusLine(cli.Pass, label, "exists"))
					results.Pass++
				} else {
					fmt.Println(theme.StatusLine(cli.Warn, label, "missing — re-provision with neo init"))
					results.Warn++
				}
			}
		} else {
			fmt.Println(theme.StatusLine(cli.Warn, skillDirLabel, "missing — re-provision with neo init"))
			results.Warn++
		}
	}
}

// printSummary renders the final pass/fail/warn summary.
func printSummary(theme *cli.Theme, results *Results) {
	rt := agentruntime.Registry.Get("claude")
	printSummaryForRuntime(theme, results, rt)
}

func printSummaryForRuntime(theme *cli.Theme, results *Results, rt agentruntime.Runtime) {
	fmt.Println()
	fmt.Println(theme.Summary(results.Pass, results.Fail, results.Warn))

	if results.Warn > 0 {
		details := []string{
			"Run: trinity refresh --stale-only to update stale docs",
			"Run: trinity sync to regenerate missing _index.md",
		}
		capabilities := rt.Capabilities()
		if capabilities.AgentDefinitions || capabilities.SkillDefinitions {
			details = append(details, "Run: neo init to re-provision missing agents/skills")
		}
		fmt.Println()
		fmt.Println(theme.WarnBox("Some checks need attention", details...))
	}
	fmt.Println()
}

func exitCode(results *Results) int {
	if results.Fail > 0 {
		return 1
	}
	return 0
}
