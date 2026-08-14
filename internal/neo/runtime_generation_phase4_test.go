package neo

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPhase4NeoRuntimeGeneration(t *testing.T) {
	profile := testProfile()

	t.Run("default and explicit claude select existing outputs", func(t *testing.T) {
		root := t.TempDir()
		defaultManifest := BuildNeoManifest(profile, root)
		explicitManifest := BuildNeoManifest(profile, root, "claude")

		if !reflect.DeepEqual(
			phase4NeoManifestSignature(t, root, defaultManifest),
			phase4NeoManifestSignature(t, root, explicitManifest),
		) {
			t.Fatal("default and explicit claude manifests differ")
		}
		phase4NeoRequireManifestPath(t, defaultManifest, root, "CLAUDE.md")
		phase4NeoRequireManifestPath(t, defaultManifest, root, ".claude/agents/developer.md")
		phase4NeoRequireManifestPath(t, defaultManifest, root, ".claude/skills/commit/SKILL.md")
		phase4NeoRejectManifestPath(t, defaultManifest, root, "AGENTS.md")
	})

	t.Run("explicit codex selects and generates AGENTS only", func(t *testing.T) {
		root := t.TempDir()
		manifest := BuildNeoManifest(profile, root, "codex")

		phase4NeoRequireManifestPath(t, manifest, root, "AGENTS.md")
		phase4NeoRejectManifestPath(t, manifest, root, "CLAUDE.md")
		phase4NeoRejectManifestPrefix(t, manifest, root, ".claude/agents/")
		phase4NeoRejectManifestPrefix(t, manifest, root, ".claude/skills/")

		if _, err := GenerateEcosystem(profile, root, "codex"); err != nil {
			t.Fatalf("GenerateEcosystem(codex): %v", err)
		}

		agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if err != nil {
			t.Fatalf("read AGENTS.md: %v", err)
		}
		if !strings.Contains(string(agents), profile.ProjectName) {
			t.Fatalf("AGENTS.md does not contain project name %q", profile.ProjectName)
		}
		for _, forbidden := range []string{
			"mcp__context7",
			"mcp__MCP_DOCKER",
			"WebSearch",
			"Agent tool",
			"worktree",
			"/owasp-review",
			"/commit",
			"/audit",
		} {
			if strings.Contains(string(agents), forbidden) {
				t.Errorf("Codex AGENTS.md contains Claude-only reference %q", forbidden)
			}
		}
		phase4NeoRequireAbsent(t, filepath.Join(root, "CLAUDE.md"))
		phase4NeoRequireAbsent(t, filepath.Join(root, ".claude", "agents"))
		phase4NeoRequireAbsent(t, filepath.Join(root, ".claude", "skills"))
	})

	t.Run("claude prompts retain runtime-specific guidance", func(t *testing.T) {
		root := t.TempDir()
		if _, err := GenerateEcosystem(profile, root, "claude"); err != nil {
			t.Fatalf("GenerateEcosystem(claude): %v", err)
		}
		checks := map[string]string{
			".claude/agents/manager.md":           "Agent tool",
			".claude/agents/developer.md":         "mcp__context7",
			".claude/agents/security-reviewer.md": "/owasp-review",
			".claude/skills/commit/SKILL.md":      "/commit",
		}
		for rel, want := range checks {
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}
			if !strings.Contains(string(content), want) {
				t.Errorf("%s does not retain %q", rel, want)
			}
		}
	})
}

func phase4NeoManifestSignature(t *testing.T, root string, manifest []ManifestEntry) []string {
	t.Helper()
	signature := make([]string, 0, len(manifest))
	for _, entry := range manifest {
		rel, err := filepath.Rel(root, entry.OutputPath)
		if err != nil {
			t.Fatalf("relative output path: %v", err)
		}
		signature = append(signature, entry.TemplatePath+"\x00"+filepath.ToSlash(rel))
	}
	return signature
}

func phase4NeoRequireManifestPath(t *testing.T, manifest []ManifestEntry, root, rel string) {
	t.Helper()
	want := filepath.Join(root, filepath.FromSlash(rel))
	for _, entry := range manifest {
		if entry.OutputPath == want {
			return
		}
	}
	t.Fatalf("manifest does not contain %s", rel)
}

func phase4NeoRejectManifestPath(t *testing.T, manifest []ManifestEntry, root, rel string) {
	t.Helper()
	want := filepath.Join(root, filepath.FromSlash(rel))
	for _, entry := range manifest {
		if entry.OutputPath == want {
			t.Fatalf("manifest unexpectedly contains %s", rel)
		}
	}
}

func phase4NeoRejectManifestPrefix(t *testing.T, manifest []ManifestEntry, root, prefix string) {
	t.Helper()
	for _, entry := range manifest {
		rel, err := filepath.Rel(root, entry.OutputPath)
		if err != nil {
			t.Fatalf("relative output path: %v", err)
		}
		if strings.HasPrefix(filepath.ToSlash(rel), prefix) {
			t.Fatalf("manifest unexpectedly contains Claude-only output %s", rel)
		}
	}
}

func phase4NeoRequireAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, stat error: %v", path, err)
	}
}
