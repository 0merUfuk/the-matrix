package morpheus

import (
	"fmt"
	"path/filepath"
)

// BuildLoopManifest returns the file manifest for a morp loop setup.
// Uses shared templates (loop.sh, tasks/*) and loop-specific templates (config, prompts).
func BuildLoopManifest(ctx *LoopContext, projectDir string) []ManifestEntry {
	o := func(rel string) string { return filepath.Join(projectDir, rel) }

	return []ManifestEntry{
		// Shared: loop orchestrator (static, copied as-is)
		{TemplatePath: "shared/loop.sh", OutputPath: o(".autonomous/loop.sh"), IsStatic: true},

		// Loop-specific: config + prompts (templated)
		{TemplatePath: "loop/config.sh.tmpl", OutputPath: o(".autonomous/config.sh"), IsStatic: false},
		{TemplatePath: "loop/prompts/manager.md.tmpl", OutputPath: o(".autonomous/prompts/manager.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/strategist.md.tmpl", OutputPath: o(".autonomous/prompts/strategist.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/developer.md.tmpl", OutputPath: o(".autonomous/prompts/developer.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/tester.md.tmpl", OutputPath: o(".autonomous/prompts/tester.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/reviewer.md.tmpl", OutputPath: o(".autonomous/prompts/reviewer.md"), IsStatic: false},
		{TemplatePath: "loop/prompts/security.md.tmpl", OutputPath: o(".autonomous/prompts/security.md"), IsStatic: false},

		// Claude Code agents (native --agent model, coexists with .autonomous/ prompts)
		{TemplatePath: "loop/agents/developer.md.tmpl", OutputPath: o(".claude/agents/developer.md"), IsStatic: false},
		{TemplatePath: "loop/agents/tester.md.tmpl", OutputPath: o(".claude/agents/tester.md"), IsStatic: false},
		{TemplatePath: "loop/agents/reviewer.md.tmpl", OutputPath: o(".claude/agents/reviewer.md"), IsStatic: false},
		{TemplatePath: "loop/agents/strategist.md.tmpl", OutputPath: o(".claude/agents/strategist.md"), IsStatic: false},
		{TemplatePath: "loop/agents/manager.md.tmpl", OutputPath: o(".claude/agents/manager.md"), IsStatic: false},

		// Claude Code skills (native /skill model)
		{TemplatePath: "loop/skills/continue/SKILL.md.tmpl", OutputPath: o(".claude/skills/continue/SKILL.md"), IsStatic: false},
		{TemplatePath: "loop/skills/commit/SKILL.md.tmpl", OutputPath: o(".claude/skills/commit/SKILL.md"), IsStatic: false},

		// Security agent + skills
		{TemplatePath: "loop/agents/security-reviewer.md.tmpl", OutputPath: o(".claude/agents/security-reviewer.md"), IsStatic: false},
		{TemplatePath: "loop/skills/owasp-review/SKILL.md.tmpl", OutputPath: o(".claude/skills/owasp-review/SKILL.md"), IsStatic: false},
		{TemplatePath: "loop/skills/secret-scan/SKILL.md.tmpl", OutputPath: o(".claude/skills/secret-scan/SKILL.md"), IsStatic: false},
		{TemplatePath: "loop/skills/dep-audit/SKILL.md.tmpl", OutputPath: o(".claude/skills/dep-audit/SKILL.md"), IsStatic: false},
		{TemplatePath: "loop/skills/security-scan/SKILL.md.tmpl", OutputPath: o(".claude/skills/security-scan/SKILL.md"), IsStatic: false},

		// Shared: task tracking files (templated — use .ServiceName alias)
		{TemplatePath: "shared/tasks/current-state.tmpl", OutputPath: o("tasks/current-state.md"), IsStatic: false},
		{TemplatePath: "shared/tasks/lessons.tmpl", OutputPath: o("tasks/lessons.md"), IsStatic: false},
		{TemplatePath: "shared/tasks/decisions-pending.tmpl", OutputPath: o("tasks/decisions-pending.md"), IsStatic: false},

		// Loop-specific: architectural reference files (generic skeletons)
		{TemplatePath: "loop/tasks/TARGET_STATE.md.tmpl", OutputPath: o("tasks/TARGET_STATE.md"), IsStatic: false},
		{TemplatePath: "loop/tasks/DECISIONS.md.tmpl", OutputPath: o("tasks/DECISIONS.md"), IsStatic: false},
	}
}

// GenerateLoopFiles renders all loop templates and writes files to disk.
func GenerateLoopFiles(ctx *LoopContext, projectDir string) ([]string, error) {
	return generateManifestFiles(BuildLoopManifest(ctx, projectDir), ctx)
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
