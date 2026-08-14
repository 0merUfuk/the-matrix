package morpheus

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPhase4MorpheusRuntimeGeneration(t *testing.T) {
	ctx := testProjectContext()

	t.Run("default and explicit claude select existing outputs", func(t *testing.T) {
		root := t.TempDir()
		defaultManifest := BuildMorpheusManifest(ctx, root)
		explicitManifest := BuildMorpheusManifest(ctx, root, "claude")

		if !reflect.DeepEqual(
			phase4MorpheusManifestSignature(t, root, defaultManifest),
			phase4MorpheusManifestSignature(t, root, explicitManifest),
		) {
			t.Fatal("default and explicit claude manifests differ")
		}
		phase4MorpheusRequireManifestEntry(t, defaultManifest, root, "CLAUDE.md", "go/CLAUDE.md.tmpl")
		phase4MorpheusRequireManifestEntry(t, defaultManifest, root, ".autonomous/config.sh", "runtimes/claude/config.sh.tmpl")
		phase4MorpheusRequireManifestPath(t, defaultManifest, root, ".claude/agents/developer.md")
		phase4MorpheusRequireManifestPath(t, defaultManifest, root, ".claude/skills/commit/SKILL.md")
		phase4MorpheusRejectManifestPath(t, defaultManifest, root, "AGENTS.md")
	})

	t.Run("loop manifest selects runtime config", func(t *testing.T) {
		root := t.TempDir()
		loopCtx := &LoopContext{}
		defaultManifest := BuildLoopManifest(loopCtx, root)
		explicitManifest := BuildLoopManifest(loopCtx, root, "claude")

		if !reflect.DeepEqual(
			phase4MorpheusManifestSignature(t, root, defaultManifest),
			phase4MorpheusManifestSignature(t, root, explicitManifest),
		) {
			t.Fatal("default and explicit claude loop manifests differ")
		}
		phase4MorpheusRequireManifestEntry(t, defaultManifest, root, ".autonomous/config.sh", "runtimes/claude/loop-config.sh.tmpl")

		codexManifest := BuildLoopManifest(loopCtx, root, "codex")
		phase4MorpheusRequireManifestEntry(t, codexManifest, root, ".autonomous/config.sh", "runtimes/codex/config.sh.tmpl")
		phase4MorpheusRejectManifestPrefix(t, codexManifest, root, ".claude/agents/")
		phase4MorpheusRejectManifestPrefix(t, codexManifest, root, ".claude/skills/")
	})

	t.Run("explicit codex selects and generates runtime files", func(t *testing.T) {
		root := t.TempDir()
		manifest := BuildMorpheusManifest(ctx, root, "codex")

		phase4MorpheusRequireManifestEntry(t, manifest, root, "AGENTS.md", "runtimes/codex/AGENTS.md.tmpl")
		phase4MorpheusRequireManifestEntry(t, manifest, root, ".autonomous/config.sh", "runtimes/codex/config.sh.tmpl")
		phase4MorpheusRejectManifestPath(t, manifest, root, "CLAUDE.md")
		phase4MorpheusRejectManifestPrefix(t, manifest, root, ".claude/agents/")
		phase4MorpheusRejectManifestPrefix(t, manifest, root, ".claude/skills/")

		if _, err := GenerateMorpheusFiles(ctx, root, "codex"); err != nil {
			t.Fatalf("GenerateMorpheusFiles(codex): %v", err)
		}

		phase4MorpheusRequireFileContains(t, filepath.Join(root, "AGENTS.md"), ctx.ServiceName)
		agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if err != nil {
			t.Fatalf("read AGENTS.md: %v", err)
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
		phase4MorpheusRequireFileContains(t, filepath.Join(root, ".autonomous", "config.sh"), `codex "${search_args[@]}" -a never exec`)
		phase4MorpheusRequireFileContains(t, filepath.Join(root, ".autonomous", "config.sh"), "--skip-git-repo-check")
		phase4MorpheusRequireFileContains(t, filepath.Join(root, ".autonomous", "loop.sh"), `run_agent "$prompt" "$allowed_tools" "$max_turns" "$output_file"`)
		phase4MorpheusRequireAbsent(t, filepath.Join(root, "CLAUDE.md"))
		phase4MorpheusRequireAbsent(t, filepath.Join(root, ".claude", "agents"))
		phase4MorpheusRequireAbsent(t, filepath.Join(root, ".claude", "skills"))
		phase4MorpheusRejectGeneratedClaudeState(t, root)
	})

	t.Run("claude prompts retain runtime-specific guidance", func(t *testing.T) {
		root := t.TempDir()
		if _, err := GenerateMorpheusFiles(ctx, root, "claude"); err != nil {
			t.Fatalf("GenerateMorpheusFiles(claude): %v", err)
		}
		checks := map[string]string{
			".claude/agents/manager.md":           "Agent tool",
			".claude/agents/developer.md":         "mcp__context7",
			".claude/agents/security-reviewer.md": "/owasp-review",
			".claude/skills/commit/SKILL.md":      "Pre-Commit Status",
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

func phase4MorpheusManifestSignature(t *testing.T, root string, manifest []ManifestEntry) []string {
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

func phase4MorpheusRequireManifestEntry(t *testing.T, manifest []ManifestEntry, root, rel, templatePath string) {
	t.Helper()
	want := filepath.Join(root, filepath.FromSlash(rel))
	for _, entry := range manifest {
		if entry.OutputPath == want {
			if entry.TemplatePath != templatePath {
				t.Fatalf("template for %s = %q, want %q", rel, entry.TemplatePath, templatePath)
			}
			return
		}
	}
	t.Fatalf("manifest does not contain %s", rel)
}

func phase4MorpheusRequireManifestPath(t *testing.T, manifest []ManifestEntry, root, rel string) {
	t.Helper()
	want := filepath.Join(root, filepath.FromSlash(rel))
	for _, entry := range manifest {
		if entry.OutputPath == want {
			return
		}
	}
	t.Fatalf("manifest does not contain %s", rel)
}

func phase4MorpheusRejectManifestPath(t *testing.T, manifest []ManifestEntry, root, rel string) {
	t.Helper()
	want := filepath.Join(root, filepath.FromSlash(rel))
	for _, entry := range manifest {
		if entry.OutputPath == want {
			t.Fatalf("manifest unexpectedly contains %s", rel)
		}
	}
}

func phase4MorpheusRejectManifestPrefix(t *testing.T, manifest []ManifestEntry, root, prefix string) {
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

func phase4MorpheusRequireFileContains(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s does not contain %q", path, want)
	}
}

func phase4MorpheusRequireAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, stat error: %v", path, err)
	}
}

func phase4MorpheusRejectGeneratedClaudeState(t *testing.T, root string) {
	t.Helper()
	forbidden := []string{
		"CLAUDE.md",
		".claude/SERVICE_CONTEXT.md",
		".claude/DECISIONS.md",
		".claude/KNOWN_ISSUES.md",
		".claude/NEXT_STEPS.md",
		".claude/agents/",
		".claude/skills/",
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, stale := range forbidden {
			if strings.Contains(string(content), stale) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				t.Errorf("generated Codex file %s contains Claude-only path %q", rel, stale)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan generated Codex files: %v", err)
	}
}
