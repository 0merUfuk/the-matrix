// Package codex implements the OpenAI Codex runtime adapter.
package codex

import (
	"os"
	"os/exec"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/runtime"
)

func init() {
	runtime.Registry.Register(&CodexRuntime{})
}

// CodexRuntime adapts the Codex CLI to the runtime interface.
type CodexRuntime struct{}

func (c *CodexRuntime) Name() string { return "codex" }

func (c *CodexRuntime) DisplayName() string { return "OpenAI Codex" }

func (c *CodexRuntime) Capabilities() runtime.Capabilities {
	return runtime.Capabilities{
		HeadlessMode:       true,
		ToolRestrictions:   true,
		MaxTurnsLimit:      false,
		ModelSelection:     true,
		JSONOutput:         true,
		SessionPersistence: true,
		// The CLI supports native agents and skills, but this release's generated
		// product contract consolidates guidance in AGENTS.md and provisions neither.
		AgentDefinitions:  false,
		SkillDefinitions:  false,
		WorktreeIsolation: false,
		Memory:            true,
		MCPIntegration:    true,
		AuthCheck:         true,
	}
}

func (c *CodexRuntime) BuildCommand(opts runtime.InvokeOpts) (*runtime.Command, error) {
	args := []string{
		"-a", "never",
		"exec",
		"--ephemeral",
		"--skip-git-repo-check",
		"--sandbox", c.TranslateTools(opts.Tools),
		"--model", c.ResolveModel(opts.Model),
	}
	if opts.OutputJSON {
		args = append(args, "--json")
	}
	if opts.WorkingDir != "" {
		args = append(args, "-C", opts.WorkingDir)
	}
	args = append(args, opts.Prompt)

	return &runtime.Command{
		Binary: "codex",
		Args:   args,
	}, nil
}

func (c *CodexRuntime) CheckAuth() runtime.AuthStatus {
	status := runtime.AuthStatus{BinaryName: "codex"}
	binary, err := exec.LookPath("codex")
	if err != nil {
		status.Error = err.Error()
		return status
	}

	status.Installed = true
	if os.Getenv("CODEX_API_KEY") != "" || os.Getenv("CODEX_ACCESS_TOKEN") != "" {
		status.Authenticated = true
		return status
	}

	if err := exec.Command(binary, "login", "status").Run(); err != nil {
		status.Error = err.Error()
		return status
	}

	status.Authenticated = true
	return status
}

func (c *CodexRuntime) ContextFiles() runtime.ContextFileSpec {
	return runtime.ContextFileSpec{
		PrimaryContext: "AGENTS.md",
		KnowledgeDir:   ".claude/knowledge",
	}
}

func (c *CodexRuntime) AgentFormat() runtime.AgentFormat {
	return runtime.AgentFormat{
		Type: runtime.AgentFormatAgentsMDSection,
	}
}

func (c *CodexRuntime) SkillFormat() *runtime.SkillFormat { return nil }

func (c *CodexRuntime) SettingsFormat() runtime.SettingsFormat {
	return runtime.SettingsFormat{
		Type:       runtime.SettingsTOML,
		ToolSyntax: runtime.ToolSyntaxSandboxMode,
	}
}

func (c *CodexRuntime) EnvModifications() []runtime.EnvModification { return nil }

func (c *CodexRuntime) TranslateTools(universalTools []string) string {
	for _, tool := range universalTools {
		tool = strings.ToLower(strings.TrimSpace(tool))
		if strings.Contains(tool, "write") ||
			strings.Contains(tool, "edit") ||
			strings.Contains(tool, "bash") ||
			strings.Contains(tool, "shell") ||
			strings.Contains(tool, "exec") ||
			strings.Contains(tool, "command") ||
			strings.Contains(tool, "patch") {
			return "workspace-write"
		}
	}
	return "read-only"
}

func (c *CodexRuntime) ResolveModel(tier runtime.ModelTier) string {
	switch tier {
	case runtime.ModelFast:
		return "gpt-5.6-luna"
	case runtime.ModelBalanced:
		return "gpt-5.6-terra"
	case runtime.ModelPowerful:
		return "gpt-5.6-sol"
	default:
		return "gpt-5.6-terra"
	}
}
