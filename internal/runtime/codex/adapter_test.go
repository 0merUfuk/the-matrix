package codex

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	aralruntime "github.com/0merUfuk/the-matrix/internal/runtime"
)

func TestNameAndDisplayName(t *testing.T) {
	rt := &CodexRuntime{}

	if got := rt.Name(); got != "codex" {
		t.Errorf("Name() = %q, want %q", got, "codex")
	}
	if got := rt.DisplayName(); got != "OpenAI Codex" {
		t.Errorf("DisplayName() = %q, want %q", got, "OpenAI Codex")
	}
	if got := aralruntime.Registry.Get("codex"); got == nil {
		t.Error("Codex runtime was not registered by init")
	}
}

func TestCapabilities(t *testing.T) {
	rt := &CodexRuntime{}
	want := aralruntime.Capabilities{
		HeadlessMode:       true,
		ToolRestrictions:   true,
		MaxTurnsLimit:      false,
		ModelSelection:     true,
		JSONOutput:         true,
		SessionPersistence: true,
		AgentDefinitions:   false,
		SkillDefinitions:   false,
		WorktreeIsolation:  false,
		Memory:             true,
		MCPIntegration:     true,
		AuthCheck:          true,
	}

	if got := rt.Capabilities(); got != want {
		t.Errorf("Capabilities() = %+v, want %+v", got, want)
	}
}

func TestBuildCommandExactArgs(t *testing.T) {
	rt := &CodexRuntime{}
	got, err := rt.BuildCommand(aralruntime.InvokeOpts{
		Prompt:     "inspect this",
		Tools:      []string{"Read", "Write"},
		MaxTurns:   99,
		Model:      aralruntime.ModelPowerful,
		OutputJSON: true,
		WorkingDir: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	want := &aralruntime.Command{
		Binary: "codex",
		Args: []string{
			"-a", "never",
			"exec",
			"--ephemeral",
			"--skip-git-repo-check",
			"--sandbox", "workspace-write",
			"--model", "gpt-5.6-sol",
			"--json",
			"-C", "/tmp/project",
			"inspect this",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildCommand() = %+v, want %+v", got, want)
	}
	for _, arg := range got.Args {
		if strings.Contains(arg, "turn") || arg == "99" {
			t.Errorf("Args contain unsupported max-turn value: %q", got.Args)
		}
	}
}

func TestBuildCommandMinimalReadOnly(t *testing.T) {
	rt := &CodexRuntime{}
	got, err := rt.BuildCommand(aralruntime.InvokeOpts{
		Prompt: "summarize",
		Tools:  []string{"Read", "Glob", "Grep"},
		Model:  aralruntime.ModelFast,
	})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	wantArgs := []string{
		"-a", "never",
		"exec",
		"--ephemeral",
		"--skip-git-repo-check",
		"--sandbox", "read-only",
		"--model", "gpt-5.6-luna",
		"summarize",
	}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Errorf("Args = %q, want %q", got.Args, wantArgs)
	}
	if got.Env != nil || got.Unset != nil {
		t.Errorf("BuildCommand environment changes = Env:%v Unset:%v, want none", got.Env, got.Unset)
	}
}

func TestResolveModel(t *testing.T) {
	rt := &CodexRuntime{}
	tests := []struct {
		name string
		tier aralruntime.ModelTier
		want string
	}{
		{name: "fast", tier: aralruntime.ModelFast, want: "gpt-5.6-luna"},
		{name: "balanced", tier: aralruntime.ModelBalanced, want: "gpt-5.6-terra"},
		{name: "powerful", tier: aralruntime.ModelPowerful, want: "gpt-5.6-sol"},
		{name: "unknown", tier: aralruntime.ModelTier(99), want: "gpt-5.6-terra"},
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
	rt := &CodexRuntime{}
	tests := []struct {
		name  string
		tools []string
		want  string
	}{
		{name: "empty", want: "read-only"},
		{name: "read tools", tools: []string{"Read", "Glob", "Grep", "WebSearch"}, want: "read-only"},
		{name: "write", tools: []string{"Read", "Write"}, want: "workspace-write"},
		{name: "edit", tools: []string{"NotebookEdit"}, want: "workspace-write"},
		{name: "bash pattern", tools: []string{"Bash(make test)"}, want: "workspace-write"},
		{name: "shell", tools: []string{"Shell"}, want: "workspace-write"},
		{name: "exec", tools: []string{"Execute"}, want: "workspace-write"},
		{name: "command", tools: []string{"Command"}, want: "workspace-write"},
		{name: "patch", tools: []string{"apply_patch"}, want: "workspace-write"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rt.TranslateTools(tt.tools); got != tt.want {
				t.Errorf("TranslateTools(%q) = %q, want %q", tt.tools, got, tt.want)
			}
		})
	}
}

func TestRuntimeFormats(t *testing.T) {
	rt := &CodexRuntime{}

	wantContext := aralruntime.ContextFileSpec{
		PrimaryContext: "AGENTS.md",
		KnowledgeDir:   ".claude/knowledge",
	}
	if got := rt.ContextFiles(); got != wantContext {
		t.Errorf("ContextFiles() = %+v, want %+v", got, wantContext)
	}

	wantAgent := aralruntime.AgentFormat{Type: aralruntime.AgentFormatAgentsMDSection}
	if got := rt.AgentFormat(); !reflect.DeepEqual(got, wantAgent) {
		t.Errorf("AgentFormat() = %+v, want %+v", got, wantAgent)
	}
	if got := rt.SkillFormat(); got != nil {
		t.Errorf("SkillFormat() = %+v, want nil", got)
	}

	wantSettings := aralruntime.SettingsFormat{
		Type:       aralruntime.SettingsTOML,
		ToolSyntax: aralruntime.ToolSyntaxSandboxMode,
	}
	if got := rt.SettingsFormat(); got != wantSettings {
		t.Errorf("SettingsFormat() = %+v, want %+v", got, wantSettings)
	}
	if got := rt.EnvModifications(); got != nil {
		t.Errorf("EnvModifications() = %+v, want nil", got)
	}
}

func TestCheckAuthNotInstalled(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("CODEX_API_KEY", "")
	t.Setenv("CODEX_ACCESS_TOKEN", "")

	got := (&CodexRuntime{}).CheckAuth()
	if got.Installed || got.Authenticated {
		t.Errorf("CheckAuth() = %+v, want not installed and not authenticated", got)
	}
	if got.BinaryName != "codex" || got.Error == "" {
		t.Errorf("CheckAuth() = %+v, want codex binary name and an error", got)
	}
}

func TestCheckAuthAcceptsCredentialEnvironment(t *testing.T) {
	for _, variable := range []string{"CODEX_API_KEY", "CODEX_ACCESS_TOKEN"} {
		t.Run(variable, func(t *testing.T) {
			dir := installFakeCodex(t, "exit 97")
			t.Setenv("PATH", dir)
			t.Setenv("CODEX_API_KEY", "")
			t.Setenv("CODEX_ACCESS_TOKEN", "")
			t.Setenv(variable, "test-credential")

			got := (&CodexRuntime{}).CheckAuth()
			if !got.Installed || !got.Authenticated || got.Error != "" {
				t.Errorf("CheckAuth() = %+v, want installed and authenticated", got)
			}
		})
	}
}

func TestCheckAuthUsesLoginStatus(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		authenticated bool
	}{
		{name: "success", body: `[ "$1" = "login" ] && [ "$2" = "status" ]` + "\n", authenticated: true},
		{name: "failure", body: "exit 1\n", authenticated: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := installFakeCodex(t, tt.body)
			t.Setenv("PATH", dir)
			t.Setenv("CODEX_API_KEY", "")
			t.Setenv("CODEX_ACCESS_TOKEN", "")

			got := (&CodexRuntime{}).CheckAuth()
			if !got.Installed {
				t.Fatalf("Installed = false, want true: %+v", got)
			}
			if got.Authenticated != tt.authenticated {
				t.Errorf("Authenticated = %v, want %v: %+v", got.Authenticated, tt.authenticated, got)
			}
			if tt.authenticated && got.Error != "" {
				t.Errorf("Error = %q, want empty", got.Error)
			}
			if !tt.authenticated && got.Error == "" {
				t.Error("Error is empty, want login status failure")
			}
		})
	}
}

func installFakeCodex(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake executable helper requires a POSIX shell")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "codex")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	return dir
}
