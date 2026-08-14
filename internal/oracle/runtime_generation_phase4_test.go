package oracle

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPhase4OracleRuntimeGeneration(t *testing.T) {
	ctx := testStackContext()

	t.Run("default and explicit claude select existing outputs", func(t *testing.T) {
		root := t.TempDir()
		defaultManifest := BuildFileManifest(ctx, root)
		explicitManifest := BuildFileManifest(ctx, root, "claude")

		if !reflect.DeepEqual(
			phase4OracleManifestSignature(t, root, defaultManifest),
			phase4OracleManifestSignature(t, root, explicitManifest),
		) {
			t.Fatal("default and explicit claude manifests differ")
		}
		phase4OracleRequireManifestEntry(t, defaultManifest, root, "CLAUDE.md", "CLAUDE.md.tmpl")
		phase4OracleRequireManifestEntry(t, defaultManifest, root, ".autonomous/config.sh", "runtimes/claude/config.sh.tmpl")
		phase4OracleRejectManifestPath(t, defaultManifest, root, "AGENTS.md")
	})

	t.Run("explicit codex selects and generates runtime files", func(t *testing.T) {
		root := t.TempDir()
		manifest := BuildFileManifest(ctx, root, "codex")

		phase4OracleRequireManifestEntry(t, manifest, root, "AGENTS.md", "runtimes/codex/AGENTS.md.tmpl")
		phase4OracleRequireManifestEntry(t, manifest, root, ".autonomous/config.sh", "runtimes/codex/config.sh.tmpl")
		phase4OracleRejectManifestPath(t, manifest, root, "CLAUDE.md")

		if _, err := GenerateFiles(ctx, root, "codex"); err != nil {
			t.Fatalf("GenerateFiles(codex): %v", err)
		}

		phase4OracleRequireFileContains(t, filepath.Join(root, "AGENTS.md"), ctx.StackName)
		phase4OracleRequireFileContains(t, filepath.Join(root, ".autonomous", "config.sh"), `codex "${search_args[@]}" -a never exec`)
		phase4OracleRequireFileContains(t, filepath.Join(root, ".autonomous", "config.sh"), "--skip-git-repo-check")
		phase4OracleRequireFileContains(t, filepath.Join(root, ".autonomous", "loop.sh"), `run_agent "$prompt" "$allowed_tools" "$max_turns" "$output_file"`)
		phase4OracleRequireAbsent(t, filepath.Join(root, "CLAUDE.md"))
		phase4OracleRequireAbsent(t, filepath.Join(root, ".claude", "agents"))
		phase4OracleRequireAbsent(t, filepath.Join(root, ".claude", "skills"))
		phase4OracleRejectGeneratedClaudeState(t, root)
	})
}

func TestPhase4OracleRuntimeConditionalResearchTools(t *testing.T) {
	for _, tt := range []struct {
		name        string
		runtimeName string
		want        bool
	}{
		{name: "claude retains named tools", runtimeName: "claude", want: true},
		{name: "codex omits named tools", runtimeName: "codex", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := GenerateFiles(testStackContext(), root, tt.runtimeName); err != nil {
				t.Fatalf("GenerateFiles(%s): %v", tt.runtimeName, err)
			}
			content, err := os.ReadFile(filepath.Join(root, ".autonomous", "prompts", "researcher.md"))
			if err != nil {
				t.Fatalf("read researcher prompt: %v", err)
			}
			for _, term := range []string{"WebSearch", "WebFetch"} {
				if got := strings.Contains(string(content), term); got != tt.want {
					t.Errorf("%s contains %q = %v, want %v", tt.runtimeName, term, got, tt.want)
				}
			}
		})
	}
}

func phase4OracleManifestSignature(t *testing.T, root string, manifest []ManifestEntry) []string {
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

func phase4OracleRequireManifestEntry(t *testing.T, manifest []ManifestEntry, root, rel, templatePath string) {
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

func phase4OracleRejectManifestPath(t *testing.T, manifest []ManifestEntry, root, rel string) {
	t.Helper()
	want := filepath.Join(root, filepath.FromSlash(rel))
	for _, entry := range manifest {
		if entry.OutputPath == want {
			t.Fatalf("manifest unexpectedly contains %s", rel)
		}
	}
}

func phase4OracleRequireFileContains(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s does not contain %q", path, want)
	}
}

func phase4OracleRequireAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, stat error: %v", path, err)
	}
}

func phase4OracleRejectGeneratedClaudeState(t *testing.T, root string) {
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
