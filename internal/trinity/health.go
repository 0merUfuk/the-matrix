package trinity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

// Health check sections, in the order RunHealth evaluates them.
const (
	SectionRuntime   = "runtime"
	SectionKnowledge = "knowledge"
	SectionContext   = "context"
	SectionEcosystem = "ecosystem"
	SectionLog       = "log"
)

// HealthCheck is one structured health result. RunHealth renders it as a
// terminal status line; the dashboard serves it as JSON.
type HealthCheck struct {
	Section string `json:"section"`
	Level   string `json:"level"` // pass | warn | fail | info
	Label   string `json:"label"`
	Detail  string `json:"detail"`
}

// KnowledgeDoc is the freshness verdict for one knowledge document.
type KnowledgeDoc struct {
	File    string `json:"file"`
	Status  string `json:"status"` // fresh | stale | undated | invalid
	Created string `json:"created,omitempty"`
	Age     string `json:"age,omitempty"`
	AgeDays int    `json:"age_days"` // -1 when the age is unknown
	Detail  string `json:"detail,omitempty"`
}

// KnowledgeStack groups the documents of one knowledge subdirectory.
type KnowledgeStack struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	HasIndex bool           `json:"has_index"`
	Error    string         `json:"error,omitempty"`
	Docs     []KnowledgeDoc `json:"docs"`
}

// EcosystemItem is one expected agent or skill and whether it is provisioned.
type EcosystemItem struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
}

// EcosystemSummary counts the provisioned agents and skills a runtime expects.
type EcosystemSummary struct {
	AgentsSupported bool            `json:"agents_supported"`
	AgentsExpected  int             `json:"agents_expected"`
	AgentsPresent   int             `json:"agents_present"`
	Agents          []EcosystemItem `json:"agents"`
	SkillsSupported bool            `json:"skills_supported"`
	SkillsExpected  int             `json:"skills_expected"`
	SkillsPresent   int             `json:"skills_present"`
	Skills          []EcosystemItem `json:"skills"`
}

// Results tracks pass/fail/warn counts across all health checks, plus the
// structured checks and knowledge inventory behind those counts.
type Results struct {
	Pass int
	Fail int
	Warn int

	Checks    []HealthCheck
	Knowledge []KnowledgeStack
	Ecosystem EcosystemSummary
}

// record counts one check, keeps it as structured data, and prints it as a
// status line when a theme is given. A nil theme collects silently.
func (r *Results) record(theme *cli.Theme, section string, level cli.Level, label, detail string) {
	name := "pass"
	switch level {
	case cli.Fail:
		r.Fail++
		name = "fail"
	case cli.Warn:
		r.Warn++
		name = "warn"
	default:
		r.Pass++
	}
	r.Checks = append(r.Checks, HealthCheck{Section: section, Level: name, Label: label, Detail: detail})
	if theme != nil {
		fmt.Println(theme.StatusLine(level, label, detail))
	}
}

// note records an informational line that does not affect the counts.
func (r *Results) note(theme *cli.Theme, section, msg string) {
	r.Checks = append(r.Checks, HealthCheck{Section: section, Level: "info", Detail: msg})
	if theme != nil {
		fmt.Println("  " + cli.Dim(msg))
	}
}

// HealthReport is the non-printing result of CollectHealth.
type HealthReport struct {
	Root         string           `json:"root"`
	Runtime      string           `json:"runtime"`
	RuntimeName  string           `json:"runtime_name"`
	MaxAgeMonths float64          `json:"max_age_months"`
	Pass         int              `json:"pass"`
	Warn         int              `json:"warn"`
	Fail         int              `json:"fail"`
	Checks       []HealthCheck    `json:"checks"`
	Knowledge    []KnowledgeStack `json:"knowledge"`
	Ecosystem    EcosystemSummary `json:"ecosystem"`
	Hints        []string         `json:"hints"`
}

// RunHealth runs the trinity health command: read-only ecosystem validation.
// Exit codes: 0 = healthy (or warnings only), 1 = any critical failure.
func RunHealth(opts HealthOpts) {
	root := resolveRoot(opts.ProjectDir)
	theme := cli.GetTheme()

	fmt.Print(theme.Banner("trinity health", cli.PathToHome(root)))
	fmt.Println()

	results := &Results{}
	rt := runHealthChecks(theme, root, opts.MaxAgeMonths, results)

	printSummaryForRuntime(theme, results, rt)
	os.Exit(exitCode(results))
}

// CollectHealth runs the same checks as RunHealth without printing or exiting.
func CollectHealth(opts HealthOpts) HealthReport {
	root := resolveRoot(opts.ProjectDir)
	results := &Results{}
	rt := runHealthChecks(nil, root, opts.MaxAgeMonths, results)

	report := HealthReport{
		Root:         root,
		Runtime:      rt.Name(),
		RuntimeName:  rt.DisplayName(),
		MaxAgeMonths: opts.MaxAgeMonths,
		Pass:         results.Pass,
		Warn:         results.Warn,
		Fail:         results.Fail,
		Checks:       results.Checks,
		Knowledge:    results.Knowledge,
		Ecosystem:    results.Ecosystem,
		Hints:        healthHints(results, rt),
	}
	if report.Checks == nil {
		report.Checks = []HealthCheck{}
	}
	if report.Knowledge == nil {
		report.Knowledge = []KnowledgeStack{}
	}
	return report
}

func resolveRoot(projectDir string) string {
	root, err := filepath.Abs(projectDir)
	if err != nil {
		root = projectDir
	}
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}
	return root
}

// runHealthChecks evaluates every check for root and returns the detected
// runtime. A nil theme collects silently.
func runHealthChecks(theme *cli.Theme, root string, maxAgeMonths float64, results *Results) agentruntime.Runtime {
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
			results.record(theme, SectionRuntime, cli.Fail, contextDirLabel, "missing")
			return rt
		}
		results.record(theme, SectionRuntime, cli.Pass, contextDirLabel, "exists")
	}

	// 2. Primary runtime context exists at project root.
	primaryContext := filepath.Join(root, contextSpec.PrimaryContext)
	if config.FileExists(primaryContext) {
		results.record(theme, SectionRuntime, cli.Pass, contextSpec.PrimaryContext, "exists")
	} else {
		results.record(theme, SectionRuntime, cli.Fail, contextSpec.PrimaryContext, "missing")
	}

	// 3. Runtime knowledge directory exists.
	// Not all projects use oracle-driven knowledge — only warn, do not fail.
	knowledgeDir := filepath.Join(root, contextSpec.KnowledgeDir)
	knowledgeLabel := filepath.ToSlash(contextSpec.KnowledgeDir) + "/"
	if contextSpec.KnowledgeDir != "" && !config.IsDir(knowledgeDir) {
		results.record(theme, SectionKnowledge, cli.Warn, knowledgeLabel, "missing — run oracle research + trinity sync if you want stack knowledge docs")
		checkContextFilesForRuntime(theme, root, contextSpec, results)
		checkAgentSkillEcosystemForRuntime(theme, root, rt, results)
		checkTrinityLogForRuntime(theme, root, contextSpec, results)
		return rt
	}
	if contextSpec.KnowledgeDir != "" {
		results.record(theme, SectionKnowledge, cli.Pass, knowledgeLabel, "exists")
	}

	// 4. Scan knowledge subdirectories
	entries, err := os.ReadDir(knowledgeDir)
	if err != nil {
		results.record(theme, SectionKnowledge, cli.Fail, knowledgeLabel, fmt.Sprintf("read error: %v", err))
	} else {
		var subdirs []string
		for _, e := range entries {
			if e.IsDir() {
				subdirs = append(subdirs, e.Name())
			}
		}
		sort.Strings(subdirs)

		if len(subdirs) == 0 {
			results.record(theme, SectionKnowledge, cli.Warn, knowledgeLabel, "no subdirectories found")
		}

		for _, name := range subdirs {
			checkKnowledgeDirAt(theme, filepath.Join(knowledgeDir, name), knowledgeLabel, name, maxAgeMonths, results)
		}
	}

	// 5. Context files
	checkContextFilesForRuntime(theme, root, contextSpec, results)

	// 6. Agent & skill ecosystem
	checkAgentSkillEcosystemForRuntime(theme, root, rt, results)

	// 7. Trinity log
	checkTrinityLogForRuntime(theme, root, contextSpec, results)

	return rt
}

// checkKnowledgeDir validates a single knowledge subdirectory.
func checkKnowledgeDir(theme *cli.Theme, dirPath, name string, maxAgeMonths float64, results *Results) {
	checkKnowledgeDirAt(theme, dirPath, ".claude/knowledge/", name, maxAgeMonths, results)
}

func checkKnowledgeDirAt(theme *cli.Theme, dirPath, knowledgeLabel, name string, maxAgeMonths float64, results *Results) {
	relPath := fmt.Sprintf("%s%s/", knowledgeLabel, name)
	stack := KnowledgeStack{Name: name, Path: relPath, Docs: []KnowledgeDoc{}}
	defer func() { results.Knowledge = append(results.Knowledge, stack) }()

	mdFiles, err := config.ListMDFiles(dirPath)
	if err != nil {
		stack.Error = fmt.Sprintf("read error: %v", err)
		results.record(theme, SectionKnowledge, cli.Fail, relPath, stack.Error)
		return
	}

	if len(mdFiles) == 0 {
		stack.Error = "empty — no .md files"
		results.record(theme, SectionKnowledge, cli.Warn, relPath, stack.Error)
		return
	}

	results.record(theme, SectionKnowledge, cli.Pass, relPath, fmt.Sprintf("%d docs", len(mdFiles)))

	// Check _index.md
	indexPath := filepath.Join(dirPath, "_index.md")
	if config.FileExists(indexPath) {
		stack.HasIndex = true
		results.record(theme, SectionKnowledge, cli.Pass, relPath+"_index.md", "exists")
	} else {
		results.record(theme, SectionKnowledge, cli.Warn, relPath+"_index.md", "missing")
	}

	// Validate each doc
	for _, filename := range mdFiles {
		doc := KnowledgeDoc{File: filename, AgeDays: -1}
		content, err := config.ReadFileString(filepath.Join(dirPath, filename))
		if err != nil {
			doc.Status, doc.Detail = "invalid", fmt.Sprintf("read error: %v", err)
			stack.Docs = append(stack.Docs, doc)
			results.record(theme, SectionKnowledge, cli.Fail, "  "+filename, doc.Detail)
			continue
		}

		// Frontmatter + EJS validation
		errors := ValidateDoc(filename, content)
		if len(errors) > 0 {
			doc.Status, doc.Detail = "invalid", errors[0]
			stack.Docs = append(stack.Docs, doc)
			results.record(theme, SectionKnowledge, cli.Fail, "  "+filename, doc.Detail)
			continue
		}

		// Staleness check
		created, ok := staleness.ParseCreatedDate(content)
		if !ok {
			doc.Status, doc.Detail = "undated", "no Created date"
			stack.Docs = append(stack.Docs, doc)
			results.record(theme, SectionKnowledge, cli.Warn, "  "+filename, doc.Detail)
			continue
		}

		doc.Created = created.Format("2006-01-02")
		doc.Age = staleness.FormatAge(created)
		doc.AgeDays = int(time.Since(created).Hours() / 24)

		if staleness.IsStale(created, maxAgeMonths) {
			doc.Status = "stale"
			stack.Docs = append(stack.Docs, doc)
			results.record(theme, SectionKnowledge, cli.Warn, "  "+filename, fmt.Sprintf("stale (%s, %s)", doc.Created, doc.Age))
		} else {
			doc.Status = "fresh"
			stack.Docs = append(stack.Docs, doc)
			results.record(theme, SectionKnowledge, cli.Pass, "  "+filename, fmt.Sprintf("fresh (%s, %s)", doc.Created, doc.Age))
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

	for _, label := range []string{"SERVICE_CONTEXT.md", "NEXT_STEPS.md"} {
		if config.FileExists(filepath.Join(root, spec.ContextDir, label)) {
			results.record(theme, SectionContext, cli.Pass, label, "exists")
		} else {
			results.record(theme, SectionContext, cli.Warn, label, "missing")
		}
	}
}

// checkTrinityLog verifies the trinity sync log exists.
func checkTrinityLog(theme *cli.Theme, root string, results *Results) {
	rt := agentruntime.Registry.Get("claude")
	checkTrinityLogForRuntime(theme, root, rt.ContextFiles(), results)
}

// trinityLogPath returns where trinity sync writes its log for spec, or ""
// when the runtime has no context location for it.
func trinityLogPath(root string, spec agentruntime.ContextFileSpec) string {
	logDir := spec.ContextDir
	if logDir == "" && spec.KnowledgeDir != "" {
		logDir = filepath.Dir(spec.KnowledgeDir)
	}
	if logDir == "" || logDir == "." {
		return ""
	}
	return filepath.Join(root, logDir, "trinity-log.md")
}

func checkTrinityLogForRuntime(theme *cli.Theme, root string, spec agentruntime.ContextFileSpec, results *Results) {
	logPath := trinityLogPath(root, spec)
	if logPath == "" {
		return
	}

	if config.FileExists(logPath) {
		results.record(theme, SectionLog, cli.Pass, "trinity-log.md", "exists")
	} else {
		results.record(theme, SectionLog, cli.Warn, "trinity-log.md", "missing — trinity sync has not been run")
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
		results.note(theme, SectionEcosystem, "Agent definitions: not provisioned for "+rt.DisplayName())
	} else {
		results.Ecosystem.AgentsSupported = true
		results.Ecosystem.AgentsExpected = len(ExpectedAgents)
		agentsDir := filepath.Join(root, spec.AgentDir)
		agentDirLabel := filepath.ToSlash(spec.AgentDir) + "/"
		hasAgentsDir := config.IsDir(agentsDir)
		for _, agent := range ExpectedAgents {
			present := hasAgentsDir && config.FileExists(filepath.Join(agentsDir, agent))
			results.Ecosystem.Agents = append(results.Ecosystem.Agents, EcosystemItem{Name: strings.TrimSuffix(agent, ".md"), Present: present})
		}
		if hasAgentsDir {
			for _, agent := range ExpectedAgents {
				label := filepath.ToSlash(filepath.Join(spec.AgentDir, agent))
				if config.FileExists(filepath.Join(agentsDir, agent)) {
					results.Ecosystem.AgentsPresent++
					results.record(theme, SectionEcosystem, cli.Pass, label, "exists")
				} else {
					results.record(theme, SectionEcosystem, cli.Warn, label, "missing — re-provision with neo init")
				}
			}
		} else {
			results.record(theme, SectionEcosystem, cli.Warn, agentDirLabel, "missing — re-provision with neo init")
		}
	}

	if !capabilities.SkillDefinitions || spec.SkillDir == "" {
		results.note(theme, SectionEcosystem, "Skill definitions: not provisioned for "+rt.DisplayName())
	} else {
		results.Ecosystem.SkillsSupported = true
		results.Ecosystem.SkillsExpected = len(ExpectedSkills)
		skillsDir := filepath.Join(root, spec.SkillDir)
		skillDirLabel := filepath.ToSlash(spec.SkillDir) + "/"
		hasSkillsDir := config.IsDir(skillsDir)
		for _, skill := range ExpectedSkills {
			present := hasSkillsDir && config.IsDir(filepath.Join(skillsDir, skill))
			results.Ecosystem.Skills = append(results.Ecosystem.Skills, EcosystemItem{Name: skill, Present: present})
		}
		if hasSkillsDir {
			for _, skill := range ExpectedSkills {
				label := filepath.ToSlash(filepath.Join(spec.SkillDir, skill)) + "/"
				if config.IsDir(filepath.Join(skillsDir, skill)) {
					results.Ecosystem.SkillsPresent++
					results.record(theme, SectionEcosystem, cli.Pass, label, "exists")
				} else {
					results.record(theme, SectionEcosystem, cli.Warn, label, "missing — re-provision with neo init")
				}
			}
		} else {
			results.record(theme, SectionEcosystem, cli.Warn, skillDirLabel, "missing — re-provision with neo init")
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

	if hints := healthHints(results, rt); len(hints) > 0 {
		fmt.Println()
		fmt.Println(theme.WarnBox("Some checks need attention", hints...))
	}
	fmt.Println()
}

// healthHints returns the remediation commands shown when warnings exist.
func healthHints(results *Results, rt agentruntime.Runtime) []string {
	if results.Warn == 0 {
		return []string{}
	}
	hints := []string{
		"Run: trinity refresh --stale-only to update stale docs",
		"Run: trinity sync to regenerate missing _index.md",
	}
	capabilities := rt.Capabilities()
	if capabilities.AgentDefinitions || capabilities.SkillDefinitions {
		hints = append(hints, "Run: neo init to re-provision missing agents/skills")
	}
	return hints
}

func exitCode(results *Results) int {
	if results.Fail > 0 {
		return 1
	}
	return 0
}
