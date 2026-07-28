package morpheus

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/tmpl"
)

//go:embed templates/shared/*
//go:embed templates/shared/tasks/*
//go:embed templates/go/*
//go:embed templates/go/.autonomous/*
//go:embed templates/go/.autonomous/prompts/*
//go:embed templates/go/.claude/*
//go:embed templates/go/.claude/agents/*
//go:embed templates/go/.claude/skills/continue/*
//go:embed templates/go/.claude/skills/commit/*
//go:embed templates/go/.claude/skills/owasp-review/*
//go:embed templates/go/.claude/skills/secret-scan/*
//go:embed templates/go/.claude/skills/dep-audit/*
//go:embed templates/go/.claude/skills/security-scan/*
//go:embed templates/go/.claude/rules/*
//go:embed templates/go/tasks/*
//go:embed templates/go/external-configs/*
//go:embed templates/node/*
//go:embed templates/node/.autonomous/*
//go:embed templates/node/.autonomous/prompts/*
//go:embed templates/node/.claude/*
//go:embed templates/node/.claude/agents/*
//go:embed templates/node/.claude/skills/continue/*
//go:embed templates/node/.claude/skills/commit/*
//go:embed templates/node/.claude/skills/owasp-review/*
//go:embed templates/node/.claude/skills/secret-scan/*
//go:embed templates/node/.claude/skills/dep-audit/*
//go:embed templates/node/.claude/skills/security-scan/*
//go:embed templates/node/.claude/rules/*
//go:embed templates/node/tasks/*
//go:embed templates/loop/*
//go:embed templates/loop/agents/*
//go:embed templates/loop/skills/continue/*
//go:embed templates/loop/skills/commit/*
//go:embed templates/loop/skills/owasp-review/*
//go:embed templates/loop/skills/secret-scan/*
//go:embed templates/loop/skills/dep-audit/*
//go:embed templates/loop/skills/security-scan/*
//go:embed templates/loop/prompts/*
//go:embed templates/loop/tasks/*
var morpheusFS embed.FS

// ManifestEntry describes a file to generate.
type ManifestEntry struct {
	TemplatePath string // relative to templates/ in embed.FS
	OutputPath   string // absolute output path
	IsStatic     bool
}

// BuildMorpheusManifest returns the file manifest for a morpheus scaffolded service.
func BuildMorpheusManifest(ctx *ProjectContext, outputDir string) []ManifestEntry {
	o := func(rel string) string { return filepath.Join(outputDir, rel) }
	techDir := "go"
	if ctx.IsNode {
		techDir = "node"
	}
	t := func(rel string) string { return filepath.Join(techDir, rel) }

	var manifest []ManifestEntry

	// Shared files (always generated)
	manifest = append(manifest,
		ManifestEntry{TemplatePath: "shared/loop.sh", OutputPath: o(".autonomous/loop.sh"), IsStatic: true},
		ManifestEntry{TemplatePath: "shared/finalizer.sh.tmpl", OutputPath: o(".autonomous/finalizer.sh"), IsStatic: false},
		ManifestEntry{TemplatePath: "shared/gitignore.tmpl", OutputPath: o(".gitignore"), IsStatic: false},
		ManifestEntry{TemplatePath: "shared/tasks/current-state.tmpl", OutputPath: o("tasks/current-state.md"), IsStatic: false},
		ManifestEntry{TemplatePath: "shared/tasks/lessons.tmpl", OutputPath: o("tasks/lessons.md"), IsStatic: false},
		ManifestEntry{TemplatePath: "shared/tasks/decisions-pending.tmpl", OutputPath: o("tasks/decisions-pending.md"), IsStatic: false},
	)

	// Tech-specific files
	manifest = append(manifest,
		ManifestEntry{TemplatePath: t("CLAUDE.md.tmpl"), OutputPath: o("CLAUDE.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/config.sh.tmpl"), OutputPath: o(".autonomous/config.sh"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/manager.md.tmpl"), OutputPath: o(".autonomous/prompts/manager.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/strategist.md.tmpl"), OutputPath: o(".autonomous/prompts/strategist.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/developer.md.tmpl"), OutputPath: o(".autonomous/prompts/developer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/tester.md.tmpl"), OutputPath: o(".autonomous/prompts/tester.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/reviewer.md.tmpl"), OutputPath: o(".autonomous/prompts/reviewer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/security.md.tmpl"), OutputPath: o(".autonomous/prompts/security.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/finalizer.md.tmpl"), OutputPath: o(".autonomous/prompts/finalizer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/settings.json.tmpl"), OutputPath: o(".claude/settings.json"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/SERVICE_CONTEXT.md.tmpl"), OutputPath: o(".claude/SERVICE_CONTEXT.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/NEXT_STEPS.md.tmpl"), OutputPath: o(".claude/NEXT_STEPS.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/DECISIONS.md.tmpl"), OutputPath: o(".claude/DECISIONS.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/KNOWN_ISSUES.md.tmpl"), OutputPath: o(".claude/KNOWN_ISSUES.md"), IsStatic: false},
	)

	// Tech-specific agents
	manifest = append(manifest,
		ManifestEntry{TemplatePath: t(".claude/agents/developer.md.tmpl"), OutputPath: o(".claude/agents/developer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/agents/tester.md.tmpl"), OutputPath: o(".claude/agents/tester.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/agents/reviewer.md.tmpl"), OutputPath: o(".claude/agents/reviewer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/agents/strategist.md.tmpl"), OutputPath: o(".claude/agents/strategist.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/agents/manager.md.tmpl"), OutputPath: o(".claude/agents/manager.md"), IsStatic: false},
	)

	// Tech-specific skills
	manifest = append(manifest,
		ManifestEntry{TemplatePath: t(".claude/skills/continue/SKILL.md.tmpl"), OutputPath: o(".claude/skills/continue/SKILL.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/skills/commit/SKILL.md.tmpl"), OutputPath: o(".claude/skills/commit/SKILL.md"), IsStatic: false},
	)

	// Security agent + skills
	manifest = append(manifest,
		ManifestEntry{TemplatePath: t(".claude/agents/security-reviewer.md.tmpl"), OutputPath: o(".claude/agents/security-reviewer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/skills/owasp-review/SKILL.md.tmpl"), OutputPath: o(".claude/skills/owasp-review/SKILL.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/skills/secret-scan/SKILL.md.tmpl"), OutputPath: o(".claude/skills/secret-scan/SKILL.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/skills/dep-audit/SKILL.md.tmpl"), OutputPath: o(".claude/skills/dep-audit/SKILL.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".claude/skills/security-scan/SKILL.md.tmpl"), OutputPath: o(".claude/skills/security-scan/SKILL.md"), IsStatic: false},
	)

	// Tech-specific rules
	if ctx.IsGo {
		manifest = append(manifest,
			ManifestEntry{TemplatePath: t(".claude/rules/go-service.md.tmpl"), OutputPath: o(".claude/rules/go-service.md"), IsStatic: false},
		)
	} else {
		manifest = append(manifest,
			ManifestEntry{TemplatePath: t(".claude/rules/node-service.md.tmpl"), OutputPath: o(".claude/rules/node-service.md"), IsStatic: false},
		)
	}

	// Tasks
	manifest = append(manifest,
		ManifestEntry{TemplatePath: t("tasks/DECISIONS.md.tmpl"), OutputPath: o("tasks/DECISIONS.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t("tasks/TARGET_STATE.md.tmpl"), OutputPath: o("tasks/TARGET_STATE.md"), IsStatic: false},
	)

	// Go-specific infrastructure
	if ctx.IsGo {
		manifest = append(manifest,
			ManifestEntry{TemplatePath: t("Makefile.tmpl"), OutputPath: o("Makefile"), IsStatic: false},
			ManifestEntry{TemplatePath: t("go.mod.tmpl"), OutputPath: o("go.mod"), IsStatic: false},
			ManifestEntry{TemplatePath: t("docker-compose.yml.tmpl"), OutputPath: o("docker-compose.yml"), IsStatic: false},
			ManifestEntry{TemplatePath: t("docker-compose.infra.yml.tmpl"), OutputPath: o("docker-compose.infra.yml"), IsStatic: false},
			ManifestEntry{TemplatePath: t("Dockerfile.api.tmpl"), OutputPath: o("Dockerfile.api"), IsStatic: false},
			ManifestEntry{TemplatePath: t("external-configs/api-local.yaml.tmpl"), OutputPath: o("external-configs/api/api-local.yaml"), IsStatic: false},
		)

		if ctx.HasWorker {
			manifest = append(manifest,
				ManifestEntry{TemplatePath: t("Dockerfile.worker.tmpl"), OutputPath: o("Dockerfile.worker"), IsStatic: false},
				ManifestEntry{TemplatePath: t("external-configs/worker-local.yaml.tmpl"), OutputPath: o("external-configs/worker/worker-local.yaml"), IsStatic: false},
			)
		}
	}

	return manifest
}

// GenerateMorpheusFiles renders all templates and writes files to disk.
func GenerateMorpheusFiles(ctx *ProjectContext, outputDir string) ([]string, error) {
	manifest := BuildMorpheusManifest(ctx, outputDir)
	var written []string

	for _, entry := range manifest {
		if err := config.EnsureDir(filepath.Dir(entry.OutputPath)); err != nil {
			return nil, fmt.Errorf("creating directory: %w", err)
		}

		if entry.IsStatic {
			data, err := morpheusFS.ReadFile("templates/" + entry.TemplatePath)
			if err != nil {
				return nil, fmt.Errorf("reading embedded file %s: %w", entry.TemplatePath, err)
			}
			if err := os.WriteFile(entry.OutputPath, data, 0755); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}
		} else {
			data, err := morpheusFS.ReadFile("templates/" + entry.TemplatePath)
			if err != nil {
				return nil, fmt.Errorf("reading template %s: %w", entry.TemplatePath, err)
			}

			rendered, err := tmpl.Render(entry.TemplatePath, string(data), ctx)
			if err != nil {
				return nil, fmt.Errorf("template render failed for %s: %w", entry.TemplatePath, err)
			}

			if err := os.WriteFile(entry.OutputPath, []byte(rendered), 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}

			if strings.HasSuffix(entry.OutputPath, ".sh") {
				os.Chmod(entry.OutputPath, 0755)
			}
		}

		written = append(written, entry.OutputPath)
	}

	return written, nil
}
