package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRuntime_Default(t *testing.T) {
	dir := isolatedProject(t)

	if got := DetectRuntime(dir); got != "claude" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "claude")
	}
}

func TestDetectRuntime_AGENTSMD(t *testing.T) {
	dir := isolatedProject(t)
	writeProjectFile(t, dir, "AGENTS.md", "# Codex\n")

	if got := DetectRuntime(dir); got != "codex" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "codex")
	}
}

func TestDetectRuntime_CLAUDEMD(t *testing.T) {
	dir := isolatedProject(t)
	writeProjectFile(t, dir, "CLAUDE.md", "# Claude\n")

	if got := DetectRuntime(dir); got != "claude" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "claude")
	}
}

func TestDetectRuntime_ExplicitOverride(t *testing.T) {
	Registry.Register(&mockRuntime{name: "codex"})
	dir := isolatedProject(t)
	writeProjectFile(t, dir, "CLAUDE.md", "# Claude\n")
	writeProjectFile(t, dir, ".matrix.yaml", "runtime: codex\n")

	if got := DetectRuntime(dir); got != "codex" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "codex")
	}
}

func TestDetectRuntime_BothContextFilesPreferClaude(t *testing.T) {
	dir := isolatedProject(t)
	writeProjectFile(t, dir, "CLAUDE.md", "# Claude\n")
	writeProjectFile(t, dir, "AGENTS.md", "# Codex\n")

	if got := DetectRuntime(dir); got != "claude" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "claude")
	}
}

func TestDetectRuntime_MatrixYML(t *testing.T) {
	Registry.Register(&mockRuntime{name: "codex"})
	dir := isolatedProject(t)
	writeProjectFile(t, dir, ".matrix.yml", "runtime: codex\n")

	if got := DetectRuntime(dir); got != "codex" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "codex")
	}
}

func TestDetectRuntime_ConfigSelectsAnyRegisteredRuntime(t *testing.T) {
	for _, name := range []string{"claude", "codex", "third-party-test-runtime"} {
		t.Run(name, func(t *testing.T) {
			Registry.Register(&mockRuntime{name: name})
			dir := isolatedProject(t)
			writeProjectFile(t, dir, ".matrix.yaml", "runtime: "+name+"\n")

			if got := DetectRuntime(dir); got != name {
				t.Fatalf("DetectRuntime() = %q, want %q", got, name)
			}
		})
	}
}

func TestDetectRuntime_MalformedConfigFallsBack(t *testing.T) {
	dir := isolatedProject(t)
	writeProjectFile(t, dir, ".matrix.yaml", "runtime:\n\t- broken: {\n")
	writeProjectFile(t, dir, "AGENTS.md", "# Codex\n")

	if got := DetectRuntime(dir); got != "codex" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "codex")
	}
}

func TestDetectRuntime_EmptyRuntimeFallsBack(t *testing.T) {
	dir := isolatedProject(t)
	writeProjectFile(t, dir, ".matrix.yaml", "runtime: \"\"\n")
	writeProjectFile(t, dir, "AGENTS.md", "# Codex\n")

	if got := DetectRuntime(dir); got != "codex" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "codex")
	}
}

func TestDetectRuntime_UnknownRuntimeFallsBack(t *testing.T) {
	dir := isolatedProject(t)
	writeProjectFile(t, dir, ".matrix.yaml", "runtime: unknown\n")
	writeProjectFile(t, dir, "CLAUDE.md", "# Claude\n")
	writeProjectFile(t, dir, "AGENTS.md", "# Codex\n")

	if got := DetectRuntime(dir); got != "claude" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "claude")
	}
}

func TestDetectRuntime_InvalidConfigFallsBackToDefault(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "malformed", content: "runtime:\n\t- broken: {\n"},
		{name: "empty", content: "runtime: \"\"\n"},
		{name: "unknown", content: "runtime: unknown\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := isolatedProject(t)
			writeProjectFile(t, dir, ".matrix.yaml", tt.content)

			if got := DetectRuntime(dir); got != "claude" {
				t.Fatalf("DetectRuntime() = %q, want %q", got, "claude")
			}
		})
	}
}

func TestDetectRuntime_IgnoresGlobalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeProjectFile(t, filepath.Join(home, ".config", "the-matrix"), "config.yaml", "runtime: codex\n")
	dir := t.TempDir()

	if got := DetectRuntime(dir); got != "claude" {
		t.Fatalf("DetectRuntime() = %q, want %q", got, "claude")
	}
}

func isolatedProject(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return t.TempDir()
}

func writeProjectFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
