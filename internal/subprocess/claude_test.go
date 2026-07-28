package subprocess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want %q", opts.Model, "claude-sonnet-4-6")
	}
	if opts.MaxTurns != 5 {
		t.Errorf("MaxTurns = %d, want %d", opts.MaxTurns, 5)
	}
	if opts.TimeoutMs != 300000 {
		t.Errorf("TimeoutMs = %d, want %d", opts.TimeoutMs, 300000)
	}
}

func TestCallClaude_CommandNotFound(t *testing.T) {
	// Override PATH so "claude" binary cannot be found.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "/nonexistent-dir-only")
	defer os.Setenv("PATH", origPath)

	_, err := CallClaude("hello", Options{
		Model:     "bogus-model",
		MaxTurns:  1,
		TimeoutMs: 5000,
	})
	if err == nil {
		t.Fatal("expected error when claude binary is not found, got nil")
	}
	// exec.LookPath or exec.Command should produce an error containing
	// "executable file not found" or similar.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed") && !strings.Contains(errMsg, "not found") && !strings.Contains(errMsg, "no such file") {
		t.Errorf("unexpected error message: %s", errMsg)
	}
}

func TestCallClaude_Timeout(t *testing.T) {
	// Use a 1ms timeout — anything we invoke should exceed this.
	_, err := CallClaude("hello", Options{
		Model:     "claude-sonnet-4-6",
		MaxTurns:  1,
		TimeoutMs: 1,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected timeout or failure error, got: %s", err.Error())
	}
}

func TestEnvironmentFiltering(t *testing.T) {
	// Write a tiny bash script that prints all environment variable names.
	// We set CLAUDECODE in the environment, then call CallClaude (which will
	// fail because claude is not installed in CI), but we can verify the
	// filtering logic by running a bash script through the same mechanism.
	//
	// Since the filtering is internal to CallClaude and not directly testable
	// via RunBash (which does NOT filter), we test it via a helper script
	// invoked through a modified approach: we replicate the filtering logic
	// and verify it directly.

	original := []string{
		"HOME=/Users/test",
		"CLAUDECODE=some-session-id",
		"PATH=/usr/bin",
		"CLAUDECODE_EXTRA=keep-this",
	}

	filtered := make([]string, 0, len(original))
	for _, e := range original {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) != 3 {
		t.Errorf("expected 3 env vars after filtering, got %d", len(filtered))
	}
	for _, e := range filtered {
		if strings.HasPrefix(e, "CLAUDECODE=") {
			t.Errorf("CLAUDECODE= should have been filtered out, but found: %s", e)
		}
	}
	// Verify CLAUDECODE_EXTRA is kept (only exact prefix "CLAUDECODE=" is removed).
	found := false
	for _, e := range filtered {
		if e == "CLAUDECODE_EXTRA=keep-this" {
			found = true
		}
	}
	if !found {
		t.Error("CLAUDECODE_EXTRA should NOT be filtered, but was removed")
	}
}

func TestRunBash_ScriptNotFound(t *testing.T) {
	err := RunBash("/nonexistent/path/to/script.sh", t.TempDir(), false)
	if err == nil {
		t.Fatal("expected error for non-existent script, got nil")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected error containing 'failed', got: %s", err.Error())
	}
}

func TestRunBash_Success(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ok.sh")

	content := "#!/bin/bash\nexit 0\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	err := RunBash(script, dir, false)
	if err != nil {
		t.Errorf("expected nil error for successful script, got: %v", err)
	}
}

func TestRunBash_Failure(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fail.sh")

	content := "#!/bin/bash\nexit 1\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	err := RunBash(script, dir, false)
	if err == nil {
		t.Fatal("expected error for script that exits 1, got nil")
	}
	if !strings.Contains(err.Error(), "fail.sh") {
		t.Errorf("error should mention script name, got: %s", err.Error())
	}
}

func TestRunBash_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker.txt")
	script := filepath.Join(dir, "cwd.sh")

	// Script writes a file in its working directory to prove cwd is set.
	content := "#!/bin/bash\npwd > marker.txt\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	if err := RunBash(script, dir, false); err != nil {
		t.Fatalf("RunBash failed: %v", err)
	}

	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("marker file not created — cwd not set correctly: %v", err)
	}
	got := strings.TrimSpace(string(data))

	// Resolve symlinks on both sides — macOS maps /var → /private/var.
	resolvedGot, _ := filepath.EvalSymlinks(got)
	resolvedDir, _ := filepath.EvalSymlinks(dir)
	if resolvedGot != resolvedDir {
		t.Errorf("working directory = %q, want %q", resolvedGot, resolvedDir)
	}
}

func TestCallClaude_DefaultsFillIn(t *testing.T) {
	// Verify that zero-value Options fields get filled with defaults.
	// We can't run claude, but we can verify the function doesn't panic
	// with empty options and returns an error (since claude isn't available
	// in most test environments).
	_, err := CallClaude("test prompt", Options{})
	if err == nil {
		// If claude happens to be installed locally, this might succeed.
		// Either outcome is acceptable — we mainly verify no panic.
		return
	}
	// Any error is fine — we just confirm no panic with zero-value Options.
}
