package claude

import (
	"reflect"
	"testing"

	"github.com/0merUfuk/the-matrix/internal/runtime"
)

func TestNameAndDisplayName(t *testing.T) {
	rt := &ClaudeRuntime{}

	if got := rt.Name(); got != "claude" {
		t.Errorf("Name() = %q, want %q", got, "claude")
	}
	if got := rt.DisplayName(); got != "Claude Code" {
		t.Errorf("DisplayName() = %q, want %q", got, "Claude Code")
	}
	if got := runtime.Registry.Get("claude"); got == nil {
		t.Error("Claude runtime was not registered by init")
	}
}

func TestCapabilities(t *testing.T) {
	rt := &ClaudeRuntime{}
	want := runtime.Capabilities{
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

	if got := rt.Capabilities(); got != want {
		t.Errorf("Capabilities() = %+v, want %+v", got, want)
	}
}

func TestBuildCommand(t *testing.T) {
	rt := &ClaudeRuntime{}
	got, err := rt.BuildCommand(runtime.InvokeOpts{
		Prompt:   "inspect this",
		MaxTurns: 5,
		Model:    runtime.ModelBalanced,
	})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	wantArgs := []string{
		"-p", "inspect this",
		"--output-format", "text",
		"--max-turns", "5",
		"--model", "claude-sonnet-4-6",
		"--no-session-persistence",
	}
	if got.Binary != "claude" {
		t.Errorf("Binary = %q, want %q", got.Binary, "claude")
	}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Errorf("Args = %q, want %q", got.Args, wantArgs)
	}
}

func TestBuildCommandWithTools(t *testing.T) {
	rt := &ClaudeRuntime{}
	got, err := rt.BuildCommand(runtime.InvokeOpts{
		Prompt:   "inspect this",
		Tools:    []string{"Read", "Bash(make *)"},
		MaxTurns: 5,
		Model:    runtime.ModelBalanced,
	})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if value, ok := argumentValue(got.Args, "--allowedTools"); !ok || value != "Read,Bash(make *)" {
		t.Errorf("--allowedTools = %q, %v; want %q, true", value, ok, "Read,Bash(make *)")
	}
}

func TestBuildCommandWithJSONOutput(t *testing.T) {
	rt := &ClaudeRuntime{}
	got, err := rt.BuildCommand(runtime.InvokeOpts{
		Prompt:     "inspect this",
		MaxTurns:   5,
		Model:      runtime.ModelBalanced,
		OutputJSON: true,
	})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if value, ok := argumentValue(got.Args, "--output-format"); !ok || value != "json" {
		t.Errorf("--output-format = %q, %v; want %q, true", value, ok, "json")
	}
}

func TestResolveModel(t *testing.T) {
	rt := &ClaudeRuntime{}
	tests := []struct {
		name string
		tier runtime.ModelTier
		want string
	}{
		{name: "fast", tier: runtime.ModelFast, want: "claude-haiku-4-5"},
		{name: "balanced", tier: runtime.ModelBalanced, want: "claude-sonnet-4-6"},
		{name: "powerful", tier: runtime.ModelPowerful, want: "claude-opus-4-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rt.ResolveModel(tt.tier); got != tt.want {
				t.Errorf("ResolveModel(%v) = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestTranslateTools(t *testing.T) {
	rt := &ClaudeRuntime{}

	if got := rt.TranslateTools([]string{"Read", "Write", "Bash(*)"}); got != "Read,Write,Bash(*)" {
		t.Errorf("TranslateTools() = %q, want %q", got, "Read,Write,Bash(*)")
	}
}

func TestEnvModifications(t *testing.T) {
	rt := &ClaudeRuntime{}
	want := []runtime.EnvModification{{Action: "unset", Key: "CLAUDECODE"}}

	if got := rt.EnvModifications(); !reflect.DeepEqual(got, want) {
		t.Errorf("EnvModifications() = %+v, want %+v", got, want)
	}
}

func TestContextFiles(t *testing.T) {
	rt := &ClaudeRuntime{}
	want := runtime.ContextFileSpec{
		PrimaryContext: "CLAUDE.md",
		ContextDir:     ".claude",
		AgentDir:       ".claude/agents",
		SkillDir:       ".claude/skills",
		RulesDir:       ".claude/rules",
		SettingsFile:   ".claude/settings.json",
		KnowledgeDir:   ".claude/knowledge",
	}

	if got := rt.ContextFiles(); got != want {
		t.Errorf("ContextFiles() = %+v, want %+v", got, want)
	}
}

func TestAgentFormat(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.AgentFormat()

	if got.Type != runtime.AgentFormatFrontmatter {
		t.Errorf("Type = %v, want %v", got.Type, runtime.AgentFormatFrontmatter)
	}
	if got.Directory != ".claude/agents" {
		t.Errorf("Directory = %q, want %q", got.Directory, ".claude/agents")
	}
	if got.FilePattern != "{{.Role}}.md" {
		t.Errorf("FilePattern = %q, want %q", got.FilePattern, "{{.Role}}.md")
	}
	if len(got.Frontmatter) != 9 {
		t.Fatalf("len(Frontmatter) = %d, want 9", len(got.Frontmatter))
	}
	if got.Frontmatter[0] != (runtime.FrontmatterField{Name: "name", Required: true, Type: "string"}) {
		t.Errorf("first Frontmatter field = %+v, want required string name", got.Frontmatter[0])
	}
}

func TestSkillFormat(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.SkillFormat()

	if got == nil {
		t.Fatal("SkillFormat() = nil, want Claude skill format")
	}
	if got.Type != runtime.SkillFormatFrontmatter {
		t.Errorf("Type = %v, want %v", got.Type, runtime.SkillFormatFrontmatter)
	}
	if got.Directory != ".claude/skills" {
		t.Errorf("Directory = %q, want %q", got.Directory, ".claude/skills")
	}
	if got.FilePattern != "{{.Name}}/SKILL.md" {
		t.Errorf("FilePattern = %q, want %q", got.FilePattern, "{{.Name}}/SKILL.md")
	}
}

func TestSettingsFormat(t *testing.T) {
	rt := &ClaudeRuntime{}
	want := runtime.SettingsFormat{
		Type:       runtime.SettingsJSON,
		Path:       ".claude/settings.json",
		ToolSyntax: runtime.ToolSyntaxBashPattern,
	}

	if got := rt.SettingsFormat(); got != want {
		t.Errorf("SettingsFormat() = %+v, want %+v", got, want)
	}
}

func TestCheckAuthNotInstalled(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	rt := &ClaudeRuntime{}

	got := rt.CheckAuth()
	if got.Installed {
		t.Errorf("Installed = true, want false")
	}
	if got.Authenticated {
		t.Errorf("Authenticated = true, want false")
	}
	if got.BinaryName != "claude" {
		t.Errorf("BinaryName = %q, want %q", got.BinaryName, "claude")
	}
}

func argumentValue(args []string, name string) (string, bool) {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1], true
		}
	}
	return "", false
}
