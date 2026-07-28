package neo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
)

// RunDoctor validates a neo-provisioned ecosystem.
// Runs trinity health checks + neo-specific validations.
func RunDoctor(projectDir string) {
	root, _ := filepath.Abs(projectDir)
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}
	relRoot := cli.PathToHome(root)

	fmt.Printf("\n  %s\n\n", cli.Bold(cli.Cyan("neo doctor — "+relRoot)))

	pass, fail, warn := 0, 0, 0

	// ─── 1. Trinity health checks ─────────────────────────────────────────────
	// Run core ecosystem checks (without calling os.Exit)
	claudeDir := filepath.Join(root, ".claude")
	if !config.IsDir(claudeDir) {
		cli.PrintLine(cli.Fail, ".claude/", "missing")
		fail++
	} else {
		cli.PrintLine(cli.Pass, ".claude/", "exists")
		pass++
	}

	claudeMd := filepath.Join(root, "CLAUDE.md")
	if config.FileExists(claudeMd) {
		cli.PrintLine(cli.Pass, "CLAUDE.md", "exists")
		pass++
	} else {
		cli.PrintLine(cli.Fail, "CLAUDE.md", "missing")
		fail++
	}

	// ─── 2. Neo-specific checks ───────────────────────────────────────────────
	neoJson := filepath.Join(root, ".neo.json")
	if config.FileExists(neoJson) {
		_, err := config.ReadJSON[ProjectProfile](neoJson)
		if err != nil {
			cli.PrintLine(cli.Fail, ".neo.json", "invalid JSON")
			fail++
		} else {
			cli.PrintLine(cli.Pass, ".neo.json", "valid")
			pass++
		}
	} else {
		cli.PrintLine(cli.Fail, ".neo.json", "missing — was this provisioned with neo init?")
		fail++
	}

	trinityJson := filepath.Join(root, ".trinity.json")
	if config.FileExists(trinityJson) {
		cli.PrintLine(cli.Pass, ".trinity.json", "exists")
		pass++
	} else {
		cli.PrintLine(cli.Warn, ".trinity.json", "missing — trinity sync not configured")
		warn++
	}

	// ─── 3. Knowledge directory checks ────────────────────────────────────────
	knowledgeDir := filepath.Join(claudeDir, "knowledge")
	if config.IsDir(knowledgeDir) {
		cli.PrintLine(cli.Pass, ".claude/knowledge/", "exists")
		pass++

		// Check each stack has a knowledge dir
		if config.FileExists(neoJson) {
			profile, err := config.ReadJSON[ProjectProfile](neoJson)
			if err == nil {
				for _, stack := range profile.Stacks {
					stackDir := filepath.Join(knowledgeDir, stack.Name)
					if config.IsDir(stackDir) {
						files, _ := config.ListMDFiles(stackDir)
						cli.PrintLine(cli.Pass, fmt.Sprintf("  %s/", stack.Name), fmt.Sprintf("%d docs", len(files)))
						pass++
					} else {
						cli.PrintLine(cli.Warn, fmt.Sprintf("  %s/", stack.Name), "missing — run neo update or oracle research")
						warn++
					}
				}
			}
		}
	} else {
		cli.PrintLine(cli.Warn, ".claude/knowledge/", "missing")
		warn++
	}

	// ─── 4. Context files ─────────────────────────────────────────────────────
	contextFiles := []string{
		".claude/SERVICE_CONTEXT.md",
		".claude/NEXT_STEPS.md",
		".claude/DECISIONS.md",
		".claude/KNOWN_ISSUES.md",
	}
	for _, rel := range contextFiles {
		p := filepath.Join(root, rel)
		label := filepath.Base(rel)
		if config.FileExists(p) {
			cli.PrintLine(cli.Pass, label, "exists")
			pass++
		} else {
			cli.PrintLine(cli.Warn, label, "missing — re-provision with neo init")
			warn++
		}
	}

	// ─── 5. Agent & skill ecosystem ───────────────────────────────────────────
	expectedAgents := []string{
		"manager.md", "developer.md", "tester.md", "reviewer.md",
		"strategist.md", "security-reviewer.md",
	}
	agentsDir := filepath.Join(claudeDir, "agents")
	if config.IsDir(agentsDir) {
		for _, agent := range expectedAgents {
			p := filepath.Join(agentsDir, agent)
			label := ".claude/agents/" + agent
			if config.FileExists(p) {
				cli.PrintLine(cli.Pass, label, "exists")
				pass++
			} else {
				cli.PrintLine(cli.Warn, label, "missing — re-provision with neo init")
				warn++
			}
		}
	} else {
		cli.PrintLine(cli.Warn, ".claude/agents/", "missing — re-provision with neo init")
		warn++
	}

	expectedSkills := []string{
		"audit", "commit", "continue", "dep-audit", "doublecheck",
		"fix", "issue", "owasp-review", "secret-scan", "security-scan",
	}
	skillsDir := filepath.Join(claudeDir, "skills")
	if config.IsDir(skillsDir) {
		for _, skill := range expectedSkills {
			dir := filepath.Join(skillsDir, skill)
			label := ".claude/skills/" + skill + "/"
			if config.IsDir(dir) {
				cli.PrintLine(cli.Pass, label, "exists")
				pass++
			} else {
				cli.PrintLine(cli.Warn, label, "missing — re-provision with neo init")
				warn++
			}
		}
	} else {
		cli.PrintLine(cli.Warn, ".claude/skills/", "missing — re-provision with neo init")
		warn++
	}

	// ─── 6. Trinity log ───────────────────────────────────────────────────────
	logPath := filepath.Join(claudeDir, "trinity-log.md")
	if config.FileExists(logPath) {
		cli.PrintLine(cli.Pass, "trinity-log.md", "exists")
		pass++
	} else {
		cli.PrintLine(cli.Warn, "trinity-log.md", "missing — trinity sync has not been run")
		warn++
	}

	// ─── Summary ──────────────────────────────────────────────────────────────
	fmt.Println()
	var parts []string
	if fail > 0 {
		parts = append(parts, cli.Red(fmt.Sprintf("%d critical", fail)))
	}
	if warn > 0 {
		label := "warning"
		if warn > 1 {
			label = "warnings"
		}
		parts = append(parts, cli.Yellow(fmt.Sprintf("%d %s", warn, label)))
	}
	if pass > 0 {
		parts = append(parts, cli.Green(fmt.Sprintf("%d passed", pass)))
	}
	fmt.Printf("  Status: %s\n\n", joinParts(parts))

	if fail > 0 {
		os.Exit(1)
	}
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
