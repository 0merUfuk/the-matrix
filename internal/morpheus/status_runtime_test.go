package morpheus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTechLabelUsesRuntimeRulesDirectory(t *testing.T) {
	t.Run("claude rules identify Go", func(t *testing.T) {
		root := t.TempDir()
		writeStatusRuntimeFile(t, root, ".matrix.yaml", "runtime: claude\n")
		writeStatusRuntimeFile(t, root, ".claude/rules/go-service.md", "# Go\n")
		if got := detectTechLabel(root); got != "Go" {
			t.Errorf("detectTechLabel() = %q, want Go", got)
		}
	})

	t.Run("codex ignores Claude rules", func(t *testing.T) {
		root := t.TempDir()
		writeStatusRuntimeFile(t, root, ".matrix.yaml", "runtime: codex\n")
		writeStatusRuntimeFile(t, root, ".claude/rules/go-service.md", "# stale\n")
		if got := detectTechLabel(root); got != "Unknown" {
			t.Errorf("detectTechLabel() = %q, want Unknown", got)
		}
	})
}

func writeStatusRuntimeFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", relativePath, err)
	}
}
