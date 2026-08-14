package neo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/matrixassets"
	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
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
	rt := agentruntime.Registry.Get(agentruntime.DetectRuntime(root))
	if rt == nil {
		rt = agentruntime.Registry.Default()
	}
	contextSpec := rt.ContextFiles()
	capabilities := rt.Capabilities()

	// ─── 1. Trinity health checks ─────────────────────────────────────────────
	// Run core ecosystem checks (without calling os.Exit)
	if contextSpec.ContextDir != "" {
		contextDir := filepath.Join(root, contextSpec.ContextDir)
		contextDirLabel := filepath.ToSlash(contextSpec.ContextDir) + "/"
		if !config.IsDir(contextDir) {
			cli.PrintLine(cli.Fail, contextDirLabel, "missing")
			fail++
		} else {
			cli.PrintLine(cli.Pass, contextDirLabel, "exists")
			pass++
		}
	}

	primaryContext := filepath.Join(root, contextSpec.PrimaryContext)
	if config.FileExists(primaryContext) {
		cli.PrintLine(cli.Pass, contextSpec.PrimaryContext, "exists")
		pass++
	} else {
		cli.PrintLine(cli.Fail, contextSpec.PrimaryContext, "missing")
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
	knowledgeDir := filepath.Join(root, contextSpec.KnowledgeDir)
	knowledgeLabel := filepath.ToSlash(contextSpec.KnowledgeDir) + "/"
	if contextSpec.KnowledgeDir != "" && config.IsDir(knowledgeDir) {
		cli.PrintLine(cli.Pass, knowledgeLabel, "exists")
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
	} else if contextSpec.KnowledgeDir != "" {
		cli.PrintLine(cli.Warn, knowledgeLabel, "missing")
		warn++
	}

	// ─── 4. Context files ─────────────────────────────────────────────────────
	if contextSpec.ContextDir != "" {
		contextFiles := []string{
			"SERVICE_CONTEXT.md",
			"NEXT_STEPS.md",
			"DECISIONS.md",
			"KNOWN_ISSUES.md",
		}
		for _, name := range contextFiles {
			p := filepath.Join(root, contextSpec.ContextDir, name)
			if config.FileExists(p) {
				cli.PrintLine(cli.Pass, name, "exists")
				pass++
			} else {
				cli.PrintLine(cli.Warn, name, "missing — re-provision with neo init")
				warn++
			}
		}
	}

	// ─── 5. Agent & skill ecosystem ───────────────────────────────────────────
	if capabilities.AgentDefinitions && contextSpec.AgentDir != "" {
		expectedAgents := matrixassets.ExpectedAgents
		agentsDir := filepath.Join(root, contextSpec.AgentDir)
		agentDirLabel := filepath.ToSlash(contextSpec.AgentDir) + "/"
		if config.IsDir(agentsDir) {
			for _, agent := range expectedAgents {
				p := filepath.Join(agentsDir, agent)
				label := filepath.ToSlash(filepath.Join(contextSpec.AgentDir, agent))
				if config.FileExists(p) {
					cli.PrintLine(cli.Pass, label, "exists")
					pass++
				} else {
					cli.PrintLine(cli.Warn, label, "missing — re-provision with neo init")
					warn++
				}
			}
		} else {
			cli.PrintLine(cli.Warn, agentDirLabel, "missing — re-provision with neo init")
			warn++
		}
	}

	if capabilities.SkillDefinitions && contextSpec.SkillDir != "" {
		expectedSkills := matrixassets.ExpectedSkills
		skillsDir := filepath.Join(root, contextSpec.SkillDir)
		skillDirLabel := filepath.ToSlash(contextSpec.SkillDir) + "/"
		if config.IsDir(skillsDir) {
			for _, skill := range expectedSkills {
				dir := filepath.Join(skillsDir, skill)
				label := filepath.ToSlash(filepath.Join(contextSpec.SkillDir, skill)) + "/"
				if config.IsDir(dir) {
					cli.PrintLine(cli.Pass, label, "exists")
					pass++
				} else {
					cli.PrintLine(cli.Warn, label, "missing — re-provision with neo init")
					warn++
				}
			}
		} else {
			cli.PrintLine(cli.Warn, skillDirLabel, "missing — re-provision with neo init")
			warn++
		}
	}

	// ─── 6. Trinity log ───────────────────────────────────────────────────────
	logDir := contextSpec.ContextDir
	if logDir == "" && contextSpec.KnowledgeDir != "" {
		logDir = filepath.Dir(contextSpec.KnowledgeDir)
	}
	if logDir != "" && logDir != "." {
		logPath := filepath.Join(root, logDir, "trinity-log.md")
		if config.FileExists(logPath) {
			cli.PrintLine(cli.Pass, "trinity-log.md", "exists")
			pass++
		} else {
			cli.PrintLine(cli.Warn, "trinity-log.md", "missing — trinity sync has not been run")
			warn++
		}
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
