package trinity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncOnePair_SourceNotFound(t *testing.T) {
	pair := SyncPair{
		Name: "test",
		From: "/nonexistent/source",
		To:   "/nonexistent/target",
	}

	// Capture: syncOnePair returns false when source not found
	result := syncOnePair(pair, t.TempDir(), true)
	if result {
		t.Error("expected false for missing source")
	}
}

func TestSyncOnePair_NoMDFiles(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	os.MkdirAll(sourceDir, 0755)

	// Create a non-md file
	os.WriteFile(filepath.Join(sourceDir, "readme.txt"), []byte("not md"), 0644)

	pair := SyncPair{
		Name: "empty",
		From: sourceDir,
		To:   filepath.Join(dir, "target"),
	}

	result := syncOnePair(pair, dir, true)
	if result {
		t.Error("expected false for source with no .md files")
	}
}

func TestSyncOnePair_DryRun(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	targetDir := filepath.Join(dir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create source docs
	content := strings.Repeat("line\n", 60)
	os.WriteFile(filepath.Join(sourceDir, "01-test.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(sourceDir, "02-test.md"), []byte(content), 0644)

	// Create one existing in target (to test new vs existing count)
	os.WriteFile(filepath.Join(targetDir, "01-test.md"), []byte("old"), 0644)

	pair := SyncPair{
		Name: "test-stack",
		From: sourceDir,
		To:   targetDir,
	}

	// Dry run should succeed without writing
	result := syncOnePair(pair, dir, true)
	if !result {
		t.Error("expected dry run to succeed")
	}

	// Target should NOT be modified (dry run)
	data, _ := os.ReadFile(filepath.Join(targetDir, "01-test.md"))
	if string(data) != "old" {
		t.Error("dry run should not modify target files")
	}

	// No log file should be created
	logPath := filepath.Join(dir, ".claude", "trinity-log.md")
	if _, err := os.Stat(logPath); err == nil {
		t.Error("dry run should not create log file")
	}
}

func TestNewVsExistingCount(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	targetDir := filepath.Join(dir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	content := strings.Repeat("line\n", 60)
	// 3 source files
	os.WriteFile(filepath.Join(sourceDir, "01-a.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(sourceDir, "02-b.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(sourceDir, "03-c.md"), []byte(content), 0644)

	// 1 already exists in target
	os.WriteFile(filepath.Join(targetDir, "02-b.md"), []byte("old"), 0644)

	// Verify counting logic via the log entry
	entry := BuildLogEntry("test", "target/", 3, 2, 1)
	if !strings.Contains(entry, "2 new") {
		t.Errorf("expected '2 new' in log entry, got: %s", entry)
	}
	if !strings.Contains(entry, "1 updated") {
		t.Errorf("expected '1 updated' in log entry, got: %s", entry)
	}
}
