package morpheus

import (
	"fmt"
	"path/filepath"

	"github.com/0merUfuk/the-matrix/internal/config"
	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
	"github.com/0merUfuk/the-matrix/internal/subprocess"
)

// DecomposeGoal reads the selected runtime's context and breaks a goal into tasks.
func DecomposeGoal(projectDir, goal, runtimeName string) (string, error) {
	rt := agentruntime.Registry.Get(runtimeName)
	if rt == nil {
		return "", fmt.Errorf("runtime %q not registered", runtimeName)
	}

	contextName := rt.ContextFiles().PrimaryContext
	contextPath := filepath.Join(projectDir, contextName)
	contextDoc, err := config.ReadFileString(contextPath)
	if err != nil || contextDoc == "" {
		return "", fmt.Errorf("%s is empty or unreadable at %s", contextName, contextPath)
	}

	if len(contextDoc) > 8000 {
		contextDoc = contextDoc[:8000] + "\n... (truncated)"
	}

	prompt := buildLoopDecompositionPromptForContext(contextName, contextDoc, goal)

	result, err := subprocess.Invoke(prompt, subprocess.Options{
		ModelTier:    agentruntime.ModelBalanced,
		ModelTierSet: true,
		MaxTurns:     10,
		TimeoutMs:    5 * 60 * 1000,
		Runtime:      runtimeName,
		WorkingDir:   projectDir,
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// DecomposeGoalWithClaude is the backward-compatible Claude entry point.
func DecomposeGoalWithClaude(projectDir, goal string) (string, error) {
	return DecomposeGoal(projectDir, goal, "claude")
}

func buildLoopDecompositionPrompt(claudemd, goal string) string {
	return buildLoopDecompositionPromptForContext("CLAUDE.md", claudemd, goal)
}

func buildLoopDecompositionPromptForContext(contextName, contextDoc, goal string) string {
	return fmt.Sprintf(`You are a senior software engineer. Read the project context below, then generate a comprehensive tasks/todo.md for the given goal.

## Project Context (from %s)

%s

## Goal

%s

## Requirements

- Generate specific, actionable tasks organized in phases
- Checkbox format: "- [ ] Description"
- Phase headers: "## Phase X: Name"
- Dependency order: setup → implementation → testing → finalization
- Granular: each task should be completable in 1-3 agent turns
- Include testing tasks for each implementation phase
- Start with "# Tasks" as the first line

Return ONLY the markdown content for tasks/todo.md — no preamble, no explanation.`, contextName, contextDoc, goal)
}
