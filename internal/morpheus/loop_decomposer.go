package morpheus

import (
	"fmt"

	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/subprocess"
)

// DecomposeGoalWithClaude reads the project's CLAUDE.md and calls Claude to break a goal into tasks.
func DecomposeGoalWithClaude(projectDir, goal string) (string, error) {
	claudemdPath := projectDir + "/CLAUDE.md"
	claudemd, err := config.ReadFileString(claudemdPath)
	if err != nil || claudemd == "" {
		return "", fmt.Errorf("CLAUDE.md is empty or unreadable at %s", claudemdPath)
	}

	// Truncate CLAUDE.md if very long to stay within prompt limits
	if len(claudemd) > 8000 {
		claudemd = claudemd[:8000] + "\n... (truncated)"
	}

	prompt := buildLoopDecompositionPrompt(claudemd, goal)

	result, err := subprocess.CallClaude(prompt, subprocess.Options{
		Model:     "claude-sonnet-4-6",
		MaxTurns:  10,
		TimeoutMs: 5 * 60 * 1000,
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

func buildLoopDecompositionPrompt(claudemd, goal string) string {
	return fmt.Sprintf(`You are a senior software engineer. Read the project context below, then generate a comprehensive tasks/todo.md for the given goal.

## Project Context (from CLAUDE.md)

%s

## Goal

%s

## Requirements

- Generate specific, actionable tasks organized in phases
- Checkbox format: "- [ ] Description"
- Phase headers: "## Phase X: Name"
- Dependency order: setup → implementation → testing → finalization
- Granular: each task should be completable in 1-3 Claude turns
- Include testing tasks for each implementation phase
- Start with "# Tasks" as the first line

Return ONLY the markdown content for tasks/todo.md — no preamble, no explanation.`, claudemd, goal)
}
