package morpheus

import (
	"embed"
	"path/filepath"
	"strings"

	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
)

//go:embed templates/shared/*
//go:embed templates/shared/tasks/*
//go:embed templates/shared/prompts/*.tmpl
//go:embed templates/shared/prompts/skills/*.tmpl
//go:embed templates/shared/prompts/loop/*.tmpl
//go:embed templates/shared/prompts/loop/skills/*.tmpl
//go:embed templates/runtimes/claude/*.tmpl
//go:embed templates/runtimes/claude/agents/*.tmpl
//go:embed templates/runtimes/claude/skills/*/*.tmpl
//go:embed templates/runtimes/claude/loop/agents/*.tmpl
//go:embed templates/runtimes/claude/loop/skills/*/*.tmpl
//go:embed templates/runtimes/codex/*.tmpl
//go:embed templates/go/*
//go:embed templates/go/.autonomous/*
//go:embed templates/go/.autonomous/prompts/*
//go:embed templates/go/.claude/*.tmpl
//go:embed templates/go/.claude/rules/*
//go:embed templates/go/tasks/*
//go:embed templates/go/external-configs/*
//go:embed templates/node/*
//go:embed templates/node/.autonomous/*
//go:embed templates/node/.autonomous/prompts/*
//go:embed templates/node/.claude/*.tmpl
//go:embed templates/node/.claude/rules/*
//go:embed templates/node/tasks/*
//go:embed templates/loop/prompts/*
//go:embed templates/loop/tasks/*
var morpheusFS embed.FS

// ManifestEntry describes a file to generate.
type ManifestEntry struct {
	TemplatePath string // relative to templates/ in embed.FS
	PartialPath  string // optional named partial relative to templates/
	PartialPaths []string
	OutputPath   string // absolute output path
	IsStatic     bool
}

// BuildMorpheusManifest returns the file manifest for a morp scaffolded service.
func BuildMorpheusManifest(ctx *ProjectContext, outputDir string, runtimeNames ...string) []ManifestEntry {
	o := func(rel string) string { return filepath.Join(outputDir, rel) }
	rt, _ := resolveMorpheusRuntime(runtimeNames)
	runtimeName := rt.Name()
	contextSpec := rt.ContextFiles()
	capabilities := rt.Capabilities()
	techDir := "go"
	if ctx.IsNode {
		techDir = "node"
	}
	t := func(rel string) string { return filepath.Join(techDir, rel) }
	finalizerTemplate := t(".autonomous/prompts/finalizer.md.tmpl")
	if runtimeName == "codex" {
		finalizerTemplate = "runtimes/codex/finalizer.md.tmpl"
	}
	p := func(templatePath, partialPath, outputPath string) ManifestEntry {
		return ManifestEntry{
			TemplatePath: templatePath,
			PartialPath:  partialPath,
			OutputPath:   o(outputPath),
		}
	}

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

	// Runtime context and config
	if runtimeName == "codex" {
		manifest = append(manifest, ManifestEntry{
			TemplatePath: "runtimes/codex/AGENTS.md.tmpl",
			PartialPaths: []string{
				"shared/prompts/manager.md.tmpl",
				"shared/prompts/strategist.md.tmpl",
				"shared/prompts/developer.md.tmpl",
				"shared/prompts/tester.md.tmpl",
				"shared/prompts/reviewer.md.tmpl",
				"shared/prompts/security-reviewer.md.tmpl",
			},
			OutputPath: o(contextSpec.PrimaryContext),
		})
	} else {
		manifest = append(manifest, ManifestEntry{
			TemplatePath: t("CLAUDE.md.tmpl"),
			OutputPath:   o(contextSpec.PrimaryContext),
		})
	}
	manifest = append(manifest,
		ManifestEntry{TemplatePath: "runtimes/" + runtimeName + "/config.sh.tmpl", OutputPath: o(".autonomous/config.sh"), IsStatic: false},
	)

	// Tech-specific runtime-agnostic prompts
	manifest = append(manifest,
		ManifestEntry{TemplatePath: t(".autonomous/prompts/manager.md.tmpl"), OutputPath: o(".autonomous/prompts/manager.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/strategist.md.tmpl"), OutputPath: o(".autonomous/prompts/strategist.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/developer.md.tmpl"), OutputPath: o(".autonomous/prompts/developer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/tester.md.tmpl"), OutputPath: o(".autonomous/prompts/tester.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/reviewer.md.tmpl"), OutputPath: o(".autonomous/prompts/reviewer.md"), IsStatic: false},
		ManifestEntry{TemplatePath: t(".autonomous/prompts/security.md.tmpl"), OutputPath: o(".autonomous/prompts/security.md"), IsStatic: false},
		ManifestEntry{TemplatePath: finalizerTemplate, OutputPath: o(".autonomous/prompts/finalizer.md"), IsStatic: false},
	)

	if contextSpec.ContextDir != "" {
		manifest = append(manifest,
			ManifestEntry{TemplatePath: "runtimes/claude/" + techDir + "-settings.json.tmpl", OutputPath: o(contextSpec.SettingsFile), IsStatic: false},
			ManifestEntry{TemplatePath: t(".claude/SERVICE_CONTEXT.md.tmpl"), OutputPath: o(filepath.Join(contextSpec.ContextDir, "SERVICE_CONTEXT.md")), IsStatic: false},
			ManifestEntry{TemplatePath: t(".claude/NEXT_STEPS.md.tmpl"), OutputPath: o(filepath.Join(contextSpec.ContextDir, "NEXT_STEPS.md")), IsStatic: false},
			ManifestEntry{TemplatePath: t(".claude/DECISIONS.md.tmpl"), OutputPath: o(filepath.Join(contextSpec.ContextDir, "DECISIONS.md")), IsStatic: false},
			ManifestEntry{TemplatePath: t(".claude/KNOWN_ISSUES.md.tmpl"), OutputPath: o(filepath.Join(contextSpec.ContextDir, "KNOWN_ISSUES.md")), IsStatic: false},
		)
	}

	if capabilities.AgentDefinitions && contextSpec.AgentDir != "" {
		manifest = append(manifest,
			p("runtimes/"+runtimeName+"/agents/developer.md.tmpl", "shared/prompts/developer.md.tmpl", filepath.Join(contextSpec.AgentDir, "developer.md")),
			p("runtimes/"+runtimeName+"/agents/tester.md.tmpl", "shared/prompts/tester.md.tmpl", filepath.Join(contextSpec.AgentDir, "tester.md")),
			p("runtimes/"+runtimeName+"/agents/reviewer.md.tmpl", "shared/prompts/reviewer.md.tmpl", filepath.Join(contextSpec.AgentDir, "reviewer.md")),
			p("runtimes/"+runtimeName+"/agents/strategist.md.tmpl", "shared/prompts/strategist.md.tmpl", filepath.Join(contextSpec.AgentDir, "strategist.md")),
			p("runtimes/"+runtimeName+"/agents/manager.md.tmpl", "shared/prompts/manager.md.tmpl", filepath.Join(contextSpec.AgentDir, "manager.md")),
			p("runtimes/"+runtimeName+"/agents/security-reviewer.md.tmpl", "shared/prompts/security-reviewer.md.tmpl", filepath.Join(contextSpec.AgentDir, "security-reviewer.md")),
		)
	}

	if capabilities.SkillDefinitions && contextSpec.SkillDir != "" {
		for _, skill := range []string{"continue", "commit", "owasp-review", "secret-scan", "dep-audit", "security-scan"} {
			manifest = append(manifest, p(
				filepath.Join("runtimes", runtimeName, "skills", skill, "SKILL.md.tmpl"),
				filepath.Join("shared", "prompts", "skills", skill+".md.tmpl"),
				filepath.Join(contextSpec.SkillDir, skill, "SKILL.md"),
			))
		}
	}

	if contextSpec.RulesDir != "" {
		ruleName := "node-service.md"
		if ctx.IsGo {
			ruleName = "go-service.md"
		}
		manifest = append(manifest, ManifestEntry{
			TemplatePath: t(filepath.Join(".claude/rules", ruleName+".tmpl")),
			OutputPath:   o(filepath.Join(contextSpec.RulesDir, ruleName)),
		})
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

// GenerateMorpheusFiles renders all init templates and writes files to disk.
func GenerateMorpheusFiles(ctx *ProjectContext, outputDir string, runtimeNames ...string) ([]string, error) {
	rt, _ := resolveMorpheusRuntime(runtimeNames)
	renderCtx := *ctx
	renderCtx.Runtime = rt.Name()
	setRuntimeTemplatePaths(&renderCtx.ContextFile, &renderCtx.ContextDir, &renderCtx.KnowledgeDir,
		&renderCtx.ServiceContextFile, &renderCtx.DecisionsFile, &renderCtx.KnownIssuesFile,
		&renderCtx.NextStepsFile, &renderCtx.ContextFilesLabel, rt)
	return generateManifestFiles(BuildMorpheusManifest(ctx, outputDir, rt.Name()), &renderCtx)
}

func setRuntimeTemplatePaths(
	contextFile, contextDir, knowledgeDir, serviceContextFile, decisionsFile,
	knownIssuesFile, nextStepsFile, contextFilesLabel *string,
	rt agentruntime.Runtime,
) {
	spec := rt.ContextFiles()
	*contextFile = spec.PrimaryContext
	*contextDir = spec.ContextDir
	*knowledgeDir = spec.KnowledgeDir
	if spec.ContextDir == "" {
		*serviceContextFile = spec.PrimaryContext
		*decisionsFile = spec.PrimaryContext
		*knownIssuesFile = spec.PrimaryContext
		*nextStepsFile = spec.PrimaryContext
		*contextFilesLabel = spec.PrimaryContext
		return
	}
	*serviceContextFile = filepath.Join(spec.ContextDir, "SERVICE_CONTEXT.md")
	*decisionsFile = filepath.Join(spec.ContextDir, "DECISIONS.md")
	*knownIssuesFile = filepath.Join(spec.ContextDir, "KNOWN_ISSUES.md")
	*nextStepsFile = filepath.Join(spec.ContextDir, "NEXT_STEPS.md")
	*contextFilesLabel = spec.ContextDir + "/"
}

func resolveMorpheusRuntime(runtimeNames []string) (agentruntime.Runtime, bool) {
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
