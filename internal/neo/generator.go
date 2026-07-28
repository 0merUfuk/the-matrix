package neo

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/tmpl"
)

//go:embed templates/CLAUDE.md.tmpl
//go:embed templates/.claude/SERVICE_CONTEXT.md.tmpl
//go:embed templates/.claude/NEXT_STEPS.md.tmpl
//go:embed templates/.claude/DECISIONS.md.tmpl
//go:embed templates/.claude/KNOWN_ISSUES.md.tmpl
//go:embed templates/.claude/rules/session-protocol.md.tmpl
//go:embed templates/.claude/rules/stack-services.md.tmpl
//go:embed templates/.claude/agents/strategist.md.tmpl
//go:embed templates/.claude/agents/developer.md.tmpl
//go:embed templates/.claude/agents/manager.md.tmpl
//go:embed templates/.claude/agents/reviewer.md.tmpl
//go:embed templates/.claude/agents/tester.md.tmpl
//go:embed templates/.claude/agents/security-reviewer.md.tmpl
//go:embed templates/.claude/skills/audit/SKILL.md.tmpl
//go:embed templates/.claude/skills/commit/SKILL.md.tmpl
//go:embed templates/.claude/skills/continue/SKILL.md.tmpl
//go:embed templates/.claude/skills/doublecheck/SKILL.md.tmpl
//go:embed templates/.claude/skills/fix/SKILL.md.tmpl
//go:embed templates/.claude/skills/issue/SKILL.md.tmpl
//go:embed templates/.claude/skills/owasp-review/SKILL.md.tmpl
//go:embed templates/.claude/skills/secret-scan/SKILL.md.tmpl
//go:embed templates/.claude/skills/dep-audit/SKILL.md.tmpl
//go:embed templates/.claude/skills/security-scan/SKILL.md.tmpl
var neoFS embed.FS

// ManifestEntry describes a file to generate.
type ManifestEntry struct {
	TemplatePath string // relative to templates/ in embed.FS
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
func BuildNeoManifest(profile *ProjectProfile, outputDir string) []ManifestEntry {
	o := func(rel string) string { return filepath.Join(outputDir, rel) }

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
			TemplatePath: ".claude/agents/strategist.md.tmpl",
			OutputPath:   o(".claude/agents/strategist.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/agents/developer.md.tmpl",
			OutputPath:   o(".claude/agents/developer.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/agents/manager.md.tmpl",
			OutputPath:   o(".claude/agents/manager.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/agents/reviewer.md.tmpl",
			OutputPath:   o(".claude/agents/reviewer.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/agents/tester.md.tmpl",
			OutputPath:   o(".claude/agents/tester.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/agents/security-reviewer.md.tmpl",
			OutputPath:   o(".claude/agents/security-reviewer.md"),
		},
	)

	// ─── .claude/skills/ ───────────────────────────────────────────────────
	manifest = append(manifest,
		ManifestEntry{
			TemplatePath: ".claude/skills/audit/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/audit/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/commit/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/commit/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/continue/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/continue/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/doublecheck/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/doublecheck/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/fix/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/fix/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/issue/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/issue/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/owasp-review/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/owasp-review/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/secret-scan/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/secret-scan/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/dep-audit/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/dep-audit/SKILL.md"),
		},
		ManifestEntry{
			TemplatePath: ".claude/skills/security-scan/SKILL.md.tmpl",
			OutputPath:   o(".claude/skills/security-scan/SKILL.md"),
		},
	)

	return manifest
}

// GenerateEcosystem renders all templates, writes files, and produces
// .neo.json and .trinity.json configuration files. Returns the list of
// written file paths.
func GenerateEcosystem(profile *ProjectProfile, outputDir string) ([]string, error) {
	manifest := BuildNeoManifest(profile, outputDir)
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
				templateData = profile
			}

			rendered, err := tmpl.Render(entry.TemplatePath, string(data), templateData)
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
