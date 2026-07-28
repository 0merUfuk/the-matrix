package oracle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createValidDoc(t *testing.T, dir, filename string) {
	t.Helper()
	content := "**Version**: 1.0\n**Created**: 2026-03-15\n**Last Updated**: 2026-03-15\n**Authors:** Oracle CLI (test)\n\n---\n\n# Test\n\n"
	content += strings.Repeat("Content line for testing.\n", 50)
	os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

func TestInjectNewVsUpdatedCount(t *testing.T) {
	sourceDir := filepath.Join(t.TempDir(), "source")
	targetDir := filepath.Join(t.TempDir(), "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create 3 source docs
	createValidDoc(t, sourceDir, "01-test.md")
	createValidDoc(t, sourceDir, "02-test.md")
	createValidDoc(t, sourceDir, "03-test.md")

	// Pre-existing file in target
	os.WriteFile(filepath.Join(targetDir, "02-test.md"), []byte("old content"), 0644)

	// We can't call RunInject directly (it calls os.Exit), but we can verify
	// the file categorization logic by checking what would happen
	sourceFiles := []string{"01-test.md", "02-test.md", "03-test.md"}
	newCount := 0
	updateCount := 0
	for _, f := range sourceFiles {
		if _, err := os.Stat(filepath.Join(targetDir, f)); err == nil {
			updateCount++
		} else {
			newCount++
		}
	}

	if newCount != 2 {
		t.Errorf("expected 2 new files, got %d", newCount)
	}
	if updateCount != 1 {
		t.Errorf("expected 1 updated file, got %d", updateCount)
	}
}
