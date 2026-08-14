package morpheus

import (
	"fmt"
	"path/filepath"
)

// BuildLoopManifest returns the file manifest for a morp loop setup.
// Uses shared templates (loop.sh, tasks/*) and loop-specific templates (config, prompts).
func BuildLoopManifest(ctx *LoopContext, projectDir string, runtimeNames ...string) []ManifestEntry {
	o := func(rel string) string { return filepath.Join(projectDir, rel) }
	rt, _ := resolveMorpheusRuntime(runtimeNames)
	runtimeName := rt.Name()
	contextSpec := rt.ContextFiles()
	capabilities := rt.Capabilities()
	configTemplate := "runtimes/" + runtimeName + "/config.sh.tmpl"
	if runtimeName == "claude" {
		configTemplate = "runtimes/claude/loop-config.sh.tmpl"
	}
	p := func(templatePath, partialPath, outputPath string) ManifestEntry {
		return ManifestEntry{
			TemplatePath: templatePath,
			PartialPath:  partialPath,
			OutputPath:   o(outputPath),
		}
	}

	manifest := []ManifestEntry{
		// Shared: loop orchestrator (static, copied as-is)
		{TemplatePath: "shared/loop.sh", OutputPath: o(".autonomous/loop.sh"), IsStatic: true},

		// Loop-specific: config + prompts (templated)
		{TemplatePath: configTemplate, OutputPath: o(".autonomous/config.sh"), IsStatic: false},
		{TemplatePath: "loop/prompts/manager.md.tmpl", OutputPath: o(".autonomous/prompts/manager.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/strategist.md.tmpl", OutputPath: o(".autonomous/prompts/strategist.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/developer.md.tmpl", OutputPath: o(".autonomous/prompts/developer.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/tester.md.tmpl", OutputPath: o(".autonomous/prompts/tester.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/reviewer.md.tmpl", OutputPath: o(".autonomous/prompts/reviewer.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/security.md.tmpl", OutputPath: o(".autonomous/prompts/security.md"), IsStatic: false},
	}

	if capabilities.AgentDefinitions && contextSpec.AgentDir != "" {
		for _, role := range []string{"developer", "tester", "reviewer", "strategist", "manager"} {
			manifest = append(manifest, p(
				filepath.Join("runtimes", runtimeName, "loop", "agents", role+".md.tmpl"),
				filepath.Join("shared", "prompts", "loop", role+".md.tmpl"),
				filepath.Join(contextSpec.AgentDir, role+".md"),
			))
		}
	}

	if capabilities.SkillDefinitions && contextSpec.SkillDir != "" {
		for _, skill := range []string{"continue", "commit"} {
			manifest = append(manifest, p(
				filepath.Join("runtimes", runtimeName, "loop", "skills", skill, "SKILL.md.tmpl"),
				filepath.Join("shared", "prompts", "loop", "skills", skill+".md.tmpl"),
				filepath.Join(contextSpec.SkillDir, skill, "SKILL.md"),
			))
		}
	}

	if capabilities.AgentDefinitions && contextSpec.AgentDir != "" {
		manifest = append(manifest, p(
			filepath.Join("runtimes", runtimeName, "loop", "agents", "security-reviewer.md.tmpl"),
			filepath.Join("shared", "prompts", "loop", "security-reviewer.md.tmpl"),
			filepath.Join(contextSpec.AgentDir, "security-reviewer.md"),
		))
	}

	if capabilities.SkillDefinitions && contextSpec.SkillDir != "" {
		for _, skill := range []string{"owasp-review", "secret-scan", "dep-audit", "security-scan"} {
			manifest = append(manifest, p(
				filepath.Join("runtimes", runtimeName, "loop", "skills", skill, "SKILL.md.tmpl"),
				filepath.Join("shared", "prompts", "loop", "skills", skill+".md.tmpl"),
				filepath.Join(contextSpec.SkillDir, skill, "SKILL.md"),
			))
		}
	}

	manifest = append(manifest,
		// Shared: task tracking files (templated — use .ServiceName alias)
		ManifestEntry{TemplatePath: "shared/tasks/current-state.tmpl", OutputPath: o("tasks/current-state.md"), IsStatic: false},
		ManifestEntry{TemplatePath: "shared/tasks/lessons.tmpl", OutputPath: o("tasks/lessons.md"), IsStatic: false},
		ManifestEntry{TemplatePath: "shared/tasks/decisions-pending.tmpl", OutputPath: o("tasks/decisions-pending.md"), IsStatic: false},

		// Loop-specific: architectural reference files (generic skeletons)
		ManifestEntry{TemplatePath: "loop/tasks/TARGET_STATE.md.tmpl", OutputPath: o("tasks/TARGET_STATE.md"), IsStatic: false},
		ManifestEntry{TemplatePath: "loop/tasks/DECISIONS.md.tmpl", OutputPath: o("tasks/DECISIONS.md"), IsStatic: false},
	)

	return manifest
}

// GenerateLoopFiles renders all loop templates and writes files to disk.
func GenerateLoopFiles(ctx *LoopContext, projectDir string, runtimeNames ...string) ([]string, error) {
	rt, _ := resolveMorpheusRuntime(runtimeNames)
	renderCtx := *ctx
	renderCtx.Runtime = rt.Name()
	setRuntimeTemplatePaths(&renderCtx.ContextFile, &renderCtx.ContextDir, &renderCtx.KnowledgeDir,
		&renderCtx.ServiceContextFile, &renderCtx.DecisionsFile, &renderCtx.KnownIssuesFile,
		&renderCtx.NextStepsFile, &renderCtx.ContextFilesLabel, rt)
	return generateManifestFiles(BuildLoopManifest(ctx, projectDir, rt.Name()), &renderCtx)
}

// BuildEmptyTodo generates a skeleton tasks/todo.md when no goal is provided.
func BuildEmptyTodo(projectNamePascal string) string {
	return fmt.Sprintf(`# Tasks: %s

> Fill in tasks manually, or re-run morp loop with a goal to auto-generate.

## Phase 1: Setup

- [ ] (add your tasks here)

## Phase 2: Implementation

- [ ] (add your tasks here)

## Phase 3: Testing

- [ ] (add your tasks here)
`, projectNamePascal)
}
