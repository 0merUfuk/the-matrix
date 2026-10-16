// Package claude implements the Claude Code runtime adapter.
package claude

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/runtime"
)

func init() {
	runtime.Registry.Register(&ClaudeRuntime{})
}

// ClaudeRuntime adapts Claude Code to the runtime interface.
type ClaudeRuntime struct{}

func (c *ClaudeRuntime) Name() string { return "claude" }

func (c *ClaudeRuntime) DisplayName() string { return "Claude Code" }

func (c *ClaudeRuntime) Capabilities() runtime.Capabilities {
	return runtime.Capabilities{
		HeadlessMode:       true,
		ToolRestrictions:   true,
		MaxTurnsLimit:      true,
		ModelSelection:     true,
		JSONOutput:         true,
		SessionPersistence: true,
		AgentDefinitions:   true,
		SkillDefinitions:   true,
		WorktreeIsolation:  true,
		Memory:             true,
		MCPIntegration:     true,
		AuthCheck:          true,
	}
}

func (c *ClaudeRuntime) BuildCommand(opts runtime.InvokeOpts) (*runtime.Command, error) {
	outputFormat := "text"
	if opts.OutputJSON {
		outputFormat = "json"
	}

	args := []string{
		"-p", opts.Prompt,
		"--output-format", outputFormat,
		"--max-turns", strconv.Itoa(opts.MaxTurns),
		"--model", c.ResolveModel(opts.Model),
		"--no-session-persistence",
	}

	if len(opts.Tools) > 0 {
		args = append(args, "--allowedTools", c.TranslateTools(opts.Tools))
	}

	return &runtime.Command{
		Binary: "claude",
		Args:   args,
		Unset:  []string{"CLAUDECODE"},
	}, nil
}

func (c *ClaudeRuntime) CheckAuth() runtime.AuthStatus {
	status := runtime.AuthStatus{BinaryName: "claude"}
	binary, err := exec.LookPath("claude")
	if err != nil {
		status.Error = err.Error()
		return status
	}

	status.Installed = true
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		status.Authenticated = true
		return status
	}

	if err := exec.Command(binary, "auth", "status").Run(); err != nil { // #nosec G204 -- argv-only exec of the runtime's own CLI binary, no shell string
		status.Error = err.Error()
		return status
	}

	status.Authenticated = true
	return status
}

func (c *ClaudeRuntime) ContextFiles() runtime.ContextFileSpec {
	return runtime.ContextFileSpec{
		PrimaryContext: "CLAUDE.md",
		ContextDir:     ".claude",
		AgentDir:       ".claude/agents",
		SkillDir:       ".claude/skills",
		RulesDir:       ".claude/rules",
		SettingsFile:   ".claude/settings.json",
		KnowledgeDir:   ".claude/knowledge",
	}
}

func (c *ClaudeRuntime) AgentFormat() runtime.AgentFormat {
	return runtime.AgentFormat{
		Type:        runtime.AgentFormatFrontmatter,
		Directory:   ".claude/agents",
		FilePattern: "{{.Role}}.md",
		Frontmatter: []runtime.FrontmatterField{
			{Name: "name", Required: true, Type: "string"},
			{Name: "description", Required: true, Type: "string"},
			{Name: "tools", Required: true, Type: "list"},
			{Name: "model", Required: false, Type: "string"},
			{Name: "memory", Required: false, Type: "string"},
			{Name: "isolation", Required: false, Type: "string"},
			{Name: "maxTurns", Required: false, Type: "int"},
			{Name: "permissionMode", Required: false, Type: "string"},
			{Name: "skills", Required: false, Type: "list"},
		},
	}
}

func (c *ClaudeRuntime) SkillFormat() *runtime.SkillFormat {
	return &runtime.SkillFormat{
		Type:        runtime.SkillFormatFrontmatter,
		Directory:   ".claude/skills",
		FilePattern: "{{.Name}}/SKILL.md",
		Frontmatter: []runtime.FrontmatterField{
			{Name: "description", Required: true, Type: "string"},
			{Name: "argument-hint", Required: false, Type: "string"},
			{Name: "allowed-tools", Required: true, Type: "list"},
		},
	}
}

func (c *ClaudeRuntime) SettingsFormat() runtime.SettingsFormat {
	return runtime.SettingsFormat{
		Type:       runtime.SettingsJSON,
		Path:       ".claude/settings.json",
		ToolSyntax: runtime.ToolSyntaxBashPattern,
	}
}

func (c *ClaudeRuntime) EnvModifications() []runtime.EnvModification {
	return []runtime.EnvModification{
		{Action: "unset", Key: "CLAUDECODE"},
	}
}

func (c *ClaudeRuntime) TranslateTools(universalTools []string) string {
	return strings.Join(universalTools, ",")
}

func (c *ClaudeRuntime) ResolveModel(tier runtime.ModelTier) string {
	switch tier {
	case runtime.ModelFast:
		return "claude-haiku-4-5"
	case runtime.ModelBalanced:
		return "claude-sonnet-4-6"
	case runtime.ModelPowerful:
		return "claude-opus-4-1"
	default:
		return "claude-sonnet-4-6"
	}
}
