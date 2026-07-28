package trinity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildLogEntry_AllNew(t *testing.T) {
	entry := BuildLogEntry("go-rules", "~/.claude/knowledge/go-rules/", 5, 5, 0)

	if !strings.Contains(entry, "| sync |") {
		t.Errorf("expected '| sync |' in entry, got: %s", entry)
	}
	if !strings.Contains(entry, "go-rules →") {
		t.Errorf("expected 'go-rules →' in entry, got: %s", entry)
	}
	if !strings.Contains(entry, "5 docs") {
		t.Errorf("expected '5 docs' in entry, got: %s", entry)
	}
	if !strings.Contains(entry, "5 new") {
		t.Errorf("expected '5 new' in entry, got: %s", entry)
	}
	// When all new (unchanged = total - newCount = 0), "unchanged" should NOT appear
	if strings.Contains(entry, "unchanged") {
		t.Errorf("should not contain 'unchanged' when all files are new, got: %s", entry)
	}
}

func TestBuildLogEntry_MixedCounts(t *testing.T) {
	entry := BuildLogEntry("node-rules", "target/", 10, 3, 7)

	if !strings.Contains(entry, "3 new") {
		t.Errorf("expected '3 new', got: %s", entry)
	}
	if !strings.Contains(entry, "7 updated") {
		t.Errorf("expected '7 updated', got: %s", entry)
	}
	// "unchanged" should no longer appear — sync always overwrites existing files
	if strings.Contains(entry, "unchanged") {
		t.Errorf("should not contain 'unchanged', got: %s", entry)
	}
}

func TestBuildLogEntry_AllExisting(t *testing.T) {
	entry := BuildLogEntry("go-rules", "target/", 5, 0, 5)

	if !strings.Contains(entry, "5 updated") {
		t.Errorf("expected '5 updated', got: %s", entry)
	}
	// No "unchanged" when newCount == 0
	if strings.Contains(entry, "unchanged") {
		t.Errorf("should not contain 'unchanged' when no new files, got: %s", entry)
	}
}

func TestAppendLog_NewFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, ".claude", "trinity-log.md")

	err := AppendLog(logPath, "2026-03-15 14:30 | sync | test → target | 5 docs | 5 new")
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.HasPrefix(s, "# Trinity Sync Log") {
		t.Errorf("expected header, got: %s", s[:50])
	}
	if !strings.Contains(s, "Auto-maintained") {
		t.Errorf("expected auto-maintained note, got: %s", s)
	}
	if !strings.Contains(s, "2026-03-15 14:30") {
		t.Errorf("expected log entry, got: %s", s)
	}
}

func TestAppendLog_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "trinity-log.md")

	// Create initial file
	if err := os.WriteFile(logPath, []byte("# Trinity Sync Log\n\nexisting entry"), 0644); err != nil {
		t.Fatal(err)
	}

	err := AppendLog(logPath, "new entry line")
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "existing entry") {
		t.Error("expected existing content preserved")
	}
	if !strings.Contains(s, "\nnew entry line") {
		t.Errorf("expected new entry appended with newline, got: %s", s)
	}
}
