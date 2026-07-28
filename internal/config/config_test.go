package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadConfigSh(t *testing.T) {
	content := `# Configuration
STACK_NAME="test-stack"

MAX_CYCLES=20
DEVELOPER_MAX_TURNS=35
TESTER_MAX_TURNS=25
REVIEWER_MAX_TURNS=15

# Old values:
# MAX_CYCLES=10
# DEVELOPER_MAX_TURNS=20

MODEL="claude-sonnet-4-6"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.sh")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := ReadConfigSh(path)
	if err != nil {
		t.Fatalf("ReadConfigSh() error: %v", err)
	}

	if cfg.MaxCycles != 20 {
		t.Errorf("MaxCycles = %d, want 20", cfg.MaxCycles)
	}
	if cfg.DeveloperMaxTurns != 35 {
		t.Errorf("DeveloperMaxTurns = %d, want 35", cfg.DeveloperMaxTurns)
	}
	if cfg.TesterMaxTurns != 25 {
		t.Errorf("TesterMaxTurns = %d, want 25", cfg.TesterMaxTurns)
	}
	if cfg.ReviewerMaxTurns != 15 {
		t.Errorf("ReviewerMaxTurns = %d, want 15", cfg.ReviewerMaxTurns)
	}
}

func TestReadConfigSh_SkipsComments(t *testing.T) {
	content := `# MAX_CYCLES=99
MAX_CYCLES=5
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.sh")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := ReadConfigSh(path)
	if err != nil {
		t.Fatalf("ReadConfigSh() error: %v", err)
	}
	if cfg.MaxCycles != 5 {
		t.Errorf("MaxCycles = %d, want 5 (should skip commented line)", cfg.MaxCycles)
	}
}

func TestReadJSON(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	os.WriteFile(path, []byte(`{"name":"my-service","port":8080}`), 0644)

	result, err := ReadJSON[testStruct](path)
	if err != nil {
		t.Fatalf("ReadJSON() error: %v", err)
	}
	if result.Name != "my-service" || result.Port != 8080 {
		t.Errorf("ReadJSON() = %+v, want {Name:my-service Port:8080}", result)
	}
}

func TestWriteJSON(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "test.json")

	err := WriteJSON(path, testStruct{Name: "test"})
	if err != nil {
		t.Fatalf("WriteJSON() error: %v", err)
	}

	if !FileExists(path) {
		t.Error("WriteJSON() did not create file")
	}
}

func TestListMDFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "01-core.md"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "02-structure.md"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(dir, "_index.md"), []byte("index"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("text"), 0644)

	files, err := ListMDFiles(dir)
	if err != nil {
		t.Fatalf("ListMDFiles() error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("ListMDFiles() returned %d files, want 2 (excluding _index.md and .txt)", len(files))
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	os.WriteFile(path, []byte("x"), 0644)

	if !FileExists(path) {
		t.Error("FileExists() should return true for existing file")
	}
	if FileExists(filepath.Join(dir, "nope.txt")) {
		t.Error("FileExists() should return false for non-existing file")
	}
}

// ─── H-02: CopyFile path-traversal containment ────────────────────────────────

// TestCopyFile_HappyPath verifies that a destination inside baseDst succeeds
// and the file is actually copied with 0644 perms.
func TestCopyFile_HappyPath(t *testing.T) {
	srcDir := t.TempDir()
	baseDst := t.TempDir()

	src := filepath.Join(srcDir, "payload.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(baseDst, "sub", "payload.txt")
	if err := CopyFile(src, dst, baseDst); err != nil {
		t.Fatalf("CopyFile() unexpected error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("destination not created: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("copied content = %q, want %q", got, "hello")
	}
}

// TestCopyFile_RejectsParentTraversal verifies that a `..` segment that would
// place the write outside baseDst is rejected before reading or writing.
func TestCopyFile_RejectsParentTraversal(t *testing.T) {
	srcDir := t.TempDir()
	root := t.TempDir()
	baseDst := filepath.Join(root, "allowed")
	if err := os.MkdirAll(baseDst, 0755); err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(root, "escaped")
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(srcDir, "payload.txt")
	if err := os.WriteFile(src, []byte("malicious"), 0644); err != nil {
		t.Fatal(err)
	}

	// An absolute dst that lands outside baseDst must be rejected.
	dst := filepath.Join(sibling, "stolen.txt")
	err := CopyFile(src, dst, baseDst)
	if err == nil {
		t.Fatal("CopyFile() should have rejected dst outside baseDst; got nil error")
	}
	if !strings.Contains(err.Error(), "escapes base") {
		t.Errorf("error should mention base escape, got: %v", err)
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		t.Errorf("CopyFile() wrote file despite rejection: %s", dst)
	}

	// A relative dst that tries to escape via ".." must also be rejected.
	relDst := filepath.Join("..", "escaped", "stolen2.txt")
	err = CopyFile(src, relDst, baseDst)
	if err == nil {
		t.Fatal("CopyFile() should have rejected ../ relative traversal; got nil error")
	}
}

// TestCopyFile_RejectsSymlinkEscape verifies that a symlinked parent directory
// pointing outside baseDst is detected by resolving symlinks before the
// prefix check (security-baseline §1 requires this).
func TestCopyFile_RejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows; covered on unix CI")
	}

	srcDir := t.TempDir()
	root := t.TempDir()
	baseDst := filepath.Join(root, "allowed")
	if err := os.MkdirAll(baseDst, 0755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink INSIDE baseDst that points to `outside`. A naive
	// lexical-only containment check would accept baseDst/trap/file because
	// it starts with baseDst; the real path resolves to outside/file.
	trap := filepath.Join(baseDst, "trap")
	if err := os.Symlink(outside, trap); err != nil {
		t.Fatalf("os.Symlink failed: %v", err)
	}

	src := filepath.Join(srcDir, "payload.txt")
	if err := os.WriteFile(src, []byte("malicious"), 0644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(trap, "stolen.txt")
	err := CopyFile(src, dst, baseDst)
	if err == nil {
		t.Fatal("CopyFile() should have rejected symlinked dst escaping baseDst")
	}
	if !strings.Contains(err.Error(), "escapes base") {
		t.Errorf("error should mention base escape, got: %v", err)
	}
	// Assert no file was written at the resolved outside location.
	if _, statErr := os.Stat(filepath.Join(outside, "stolen.txt")); statErr == nil {
		t.Errorf("CopyFile() wrote file to outside-base location via symlink")
	}
}

// TestCopyFile_RequiresBaseDst verifies the safety net: an empty baseDst is
// never accepted, preventing accidental callers from bypassing containment.
func TestCopyFile_RequiresBaseDst(t *testing.T) {
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "payload.txt")
	if err := os.WriteFile(src, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "out.txt")

	if err := CopyFile(src, dst, ""); err == nil {
		t.Error("CopyFile() with empty baseDst should error")
	}
}
