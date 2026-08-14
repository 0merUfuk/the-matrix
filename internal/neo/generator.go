package neo

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
	"github.com/0merUfuk/the-matrix/internal/tmpl"
)

//go:embed templates/CLAUDE.md.tmpl
//go:embed templates/.claude/SERVICE_CONTEXT.md.tmpl
//go:embed templates/.claude/NEXT_STEPS.md.tmpl
//go:embed templates/.claude/DECISIONS.md.tmpl
//go:embed templates/.claude/KNOWN_ISSUES.md.tmpl
//go:embed templates/.claude/rules/session-protocol.md.tmpl
//go:embed templates/.claude/rules/stack-services.md.tmpl
//go:embed templates/shared/prompts/*.tmpl
//go:embed templates/shared/prompts/skills/*.tmpl
//go:embed templates/runtimes/claude/agents/*.tmpl
//go:embed templates/runtimes/claude/skills/*/*.tmpl
//go:embed templates/runtimes/codex/*.tmpl
var neoFS embed.FS

// ManifestEntry describes a file to generate.
type ManifestEntry struct {
	TemplatePath string // relative to templates/ in embed.FS
	PartialPath  string // optional named partial relative to templates/
	PartialPaths []string
	OutputPath   string // absolute output path
	IsStatic     bool   // if true, copy as-is without template rendering
	Data         any    // template data (defaults to ProjectProfile if nil)
}

// stackRuleData holds data for the per-stack rule template.
type stackRuleData struct {
	StackName        string
	Language         string
	Framework        string
	ArchStyle        string
	TestingFramework string
	DataLayer        string
	FileExtension    string
}

// trinityConfigJSON is the serialized form of .trinity.json.
type trinityConfigJSON struct {
	Stacks []trinityStackJSON `json:"stacks"`
}

// trinityStackJSON describes a single source-to-target sync mapping.
type trinityStackJSON struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

// languageExtension returns the primary file extension for a language.
func languageExtension(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "go"
	case "nodejs", "typescript", "javascript":
		return "ts"
	case "python":
		return "py"
	case "flutter", "dart":
		return "dart"
	case "rust":
		return "rs"
	case "java":
		return "java"
	case "ruby":
		return "rb"
	case "php":
		return "php"
	case "swift":
		return "swift"
	case "kotlin":
		return "kt"
	default:
		return "*"
	}
}

// BuildNeoManifest returns the file manifest for a neo-provisioned ecosystem.
func BuildNeoManifest(profile *ProjectProfile, outputDir string, runtimeNames ...string) []ManifestEntry {
	o := func(rel string) string { return filepath.Join(outputDir, rel) }
	rt, _ := resolveRuntime(runtimeNames)

	if rt.Name() == "codex" {
		return []ManifestEntry{
			{
				TemplatePath: "runtimes/codex/AGENTS.md.tmpl",
				PartialPaths: []string{
					"shared/prompts/manager.md.tmpl",
					"shared/prompts/strategist.md.tmpl",
					"shared/prompts/developer.md.tmpl",
					"shared/prompts/tester.md.tmpl",
					"shared/prompts/reviewer.md.tmpl",
					"shared/prompts/security-reviewer.md.tmpl",
				},
				OutputPath: o(rt.ContextFiles().PrimaryContext),
			},
		}
	}

	var manifest []ManifestEntry

	// ─── Root file ───────────────────────────────────────────────────────────
	manifest = append(manifest,
		ManifestEntry{
			TemplatePath: "CLAUDE.md.tmpl",
			OutputPath:   o("CLAUDE.md"),
		},
	)

	// ─── .claude/ context files ──────────────────────────────────────────────
	manifest = append(manifest,
		ManifestEntry{
			TemplatePath: ".claude/SERVICE_CONTEXT.md.tmpl",
			OutputPath:   o(".claude/SERVICE_CONTEXT.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/NEXT_STEPS.md.tmpl",
			OutputPath:   o(".claude/NEXT_STEPS.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/DECISIONS.md.tmpl",
			OutputPath:   o(".claude/DECISIONS.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/KNOWN_ISSUES.md.tmpl",
			OutputPath:   o(".claude/KNOWN_ISSUES.md"),
		},
	)

	// ─── .claude/rules/ ─────────────────────────────────────────────────────
	manifest = append(manifest,
		ManifestEntry{
			TemplatePath: ".claude/rules/session-protocol.md.tmpl",
			OutputPath:   o(".claude/rules/session-protocol.md"),
		},
	)

	// Per-stack rule files
	for _, stack := range profile.Stacks {
		manifest = append(manifest,
			ManifestEntry{
				TemplatePath: ".claude/rules/stack-services.md.tmpl",
				OutputPath:   o(fmt.Sprintf(".claude/rules/%s-services.md", stack.Name)),
				Data: stackRuleData{
					StackName:        stack.Name,
					Language:         stack.Language,
					Framework:        stack.Framework,
					ArchStyle:        stack.ArchStyle,
					TestingFramework: stack.TestingFramework,
					DataLayer:        stack.DataLayer,
					FileExtension:    languageExtension(stack.Language),
				},
			},
		)
	}

	// ─── .claude/agents/ ────────────────────────────────────────────────────
	manifest = append(manifest,
		ManifestEntry{
			TemplatePath: "runtimes/claude/agents/strategist.md.tmpl",
			PartialPath:  "shared/prompts/strategist.md.tmpl",
			OutputPath:   o(".claude/agents/strategist.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/agents/developer.md.tmpl",
			PartialPath:  "shared/prompts/developer.md.tmpl",
			OutputPath:   o(".claude/agents/developer.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/agents/manager.md.tmpl",
			PartialPath:  "shared/prompts/manager.md.tmpl",
			OutputPath:   o(".claude/agents/manager.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/agents/reviewer.md.tmpl",
			PartialPath:  "shared/prompts/reviewer.md.tmpl",
			OutputPath:   o(".claude/agents/reviewer.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/agents/tester.md.tmpl",
			PartialPath:  "shared/prompts/tester.md.tmpl",
			OutputPath:   o(".claude/agents/tester.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/agents/security-reviewer.md.tmpl",
			PartialPath:  "shared/prompts/security-reviewer.md.tmpl",
			OutputPath:   o(".claude/agents/security-reviewer.md"),
		},
	)

	// ─── .claude/skills/ ───────────────────────────────────────────────────
	manifest = append(manifest,
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/audit/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/audit.md.tmpl",
			OutputPath:   o(".claude/skills/audit/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/commit/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/commit.md.tmpl",
			OutputPath:   o(".claude/skills/commit/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/continue/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/continue.md.tmpl",
			OutputPath:   o(".claude/skills/continue/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/doublecheck/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/doublecheck.md.tmpl",
			OutputPath:   o(".claude/skills/doublecheck/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/fix/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/fix.md.tmpl",
			OutputPath:   o(".claude/skills/fix/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/issue/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/issue.md.tmpl",
			OutputPath:   o(".claude/skills/issue/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/owasp-review/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/owasp-review.md.tmpl",
			OutputPath:   o(".claude/skills/owasp-review/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/secret-scan/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/secret-scan.md.tmpl",
			OutputPath:   o(".claude/skills/secret-scan/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/dep-audit/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/dep-audit.md.tmpl",
			OutputPath:   o(".claude/skills/dep-audit/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: "runtimes/claude/skills/security-scan/SKILL.md.tmpl",
			PartialPath:  "shared/prompts/skills/security-scan.md.tmpl",
			OutputPath:   o(".claude/skills/security-scan/SKILL.md"),
		},
	)

	return manifest
}

// GenerateEcosystem renders all templates, writes files, and produces
// .neo.json and .trinity.json configuration files. Returns the list of
// written file paths.
func GenerateEcosystem(profile *ProjectProfile, outputDir string, runtimeNames ...string) ([]string, error) {
	rt, _ := resolveRuntime(runtimeNames)
	manifest := BuildNeoManifest(profile, outputDir, rt.Name())
	renderProfile := *profile
	renderProfile.Runtime = rt.Name()
	spec := rt.ContextFiles()
	renderProfile.ContextFile = spec.PrimaryContext
	renderProfile.KnowledgeDir = spec.KnowledgeDir
	if spec.ContextDir == "" {
		renderProfile.ServiceContextFile = spec.PrimaryContext
		renderProfile.DecisionsFile = spec.PrimaryContext
		renderProfile.KnownIssuesFile = spec.PrimaryContext
		renderProfile.NextStepsFile = spec.PrimaryContext
		renderProfile.ContextFilesLabel = spec.PrimaryContext
	} else {
		renderProfile.ServiceContextFile = filepath.Join(spec.ContextDir, "SERVICE_CONTEXT.md")
		renderProfile.DecisionsFile = filepath.Join(spec.ContextDir, "DECISIONS.md")
		renderProfile.KnownIssuesFile = filepath.Join(spec.ContextDir, "KNOWN_ISSUES.md")
		renderProfile.NextStepsFile = filepath.Join(spec.ContextDir, "NEXT_STEPS.md")
		renderProfile.ContextFilesLabel = spec.ContextDir + "/"
	}
	var written []string

	for _, entry := range manifest {
		// Ensure parent directory exists
		if err := config.EnsureDir(filepath.Dir(entry.OutputPath)); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", entry.OutputPath, err)
		}

		// Read embedded template
		data, err := neoFS.ReadFile("templates/" + entry.TemplatePath)
		if err != nil {
			return nil, fmt.Errorf("reading embedded file %s: %w", entry.TemplatePath, err)
		}

		if entry.IsStatic {
			// Copy as-is
			if err := config.WriteFileString(entry.OutputPath, string(data)); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}
		} else {
			// Determine template data
			templateData := entry.Data
			if templateData == nil {
				templateData = &renderProfile
			}

			var rendered string
			partialPaths := append([]string(nil), entry.PartialPaths...)
			if entry.PartialPath != "" {
				partialPaths = append(partialPaths, entry.PartialPath)
			}
			if len(partialPaths) == 0 {
				rendered, err = tmpl.Render(entry.TemplatePath, string(data), templateData)
			} else {
				partials := make(map[string]string, len(partialPaths))
				for _, partialPath := range partialPaths {
					partial, readErr := neoFS.ReadFile("templates/" + partialPath)
					if readErr != nil {
						return nil, fmt.Errorf("reading embedded partial %s: %w", partialPath, readErr)
					}
					partialName := strings.TrimSuffix(partialPath, ".tmpl")
					partials[partialName] = string(partial)
				}
				rendered, err = tmpl.RenderWithPartials(
					entry.TemplatePath,
					string(data),
					partials,
					templateData,
				)
			}
			if err != nil {
				return nil, fmt.Errorf("template render failed for %s: %w", entry.TemplatePath, err)
			}

			if err := config.WriteFileString(entry.OutputPath, rendered); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}
		}

		written = append(written, entry.OutputPath)
	}

	// ─── .neo.json ──────────────────────────────────────────────────────────
	neoJsonPath := filepath.Join(outputDir, ".neo.json")
	if err := config.WriteJSON(neoJsonPath, profile); err != nil {
		return nil, fmt.Errorf("writing .neo.json: %w", err)
	}
	written = append(written, neoJsonPath)

	// ─── .trinity.json ──────────────────────────────────────────────────────
	trinityCfg := newTrinityConfig(profile)
	trinityJsonPath := filepath.Join(outputDir, ".trinity.json")
	if err := config.WriteJSON(trinityJsonPath, trinityCfg); err != nil {
		return nil, fmt.Errorf("writing .trinity.json: %w", err)
	}
	written = append(written, trinityJsonPath)

	return written, nil
}

func resolveRuntime(runtimeNames []string) (agentruntime.Runtime, bool) {
	name := ""
	if len(runtimeNames) > 0 {
		name = strings.TrimSpace(runtimeNames[0])
	}
	explicit := name != ""
	if name == "" {
		name = "claude"
	}
	rt := agentruntime.Registry.Get(name)
	if rt == nil {
		rt = agentruntime.Registry.Default()
	}
	return rt, explicit
}

// newTrinityConfig creates a .trinity.json config from the project profile.
// Each stack gets a sync pair mapping oracle output to .claude/knowledge/.
func newTrinityConfig(profile *ProjectProfile) trinityConfigJSON {
	var stacks []trinityStackJSON
	for _, s := range profile.Stacks {
		stacks = append(stacks, trinityStackJSON{
			Name: s.Name,
			From: fmt.Sprintf("oracle-output/%s/", s.Name),
			To:   fmt.Sprintf(".claude/knowledge/%s/", s.Name),
		})
	}
	return trinityConfigJSON{Stacks: stacks}
}
