package neo

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunPresetFlow_NonInteractive is a regression guard for the launch-day
// P0: neo init --preset was advertised as non-interactive via cobra.Example
// (PR #108) but runPresetFlow previously called wizard.AskInput, which
// requires a TTY. On CI (no controlling terminal) this either exited 1 with
// a bubbletea "opening TTY" error or — worse — exited 0 on closed stdin,
// silently corrupting pipelines.
//
// runPresetFlow must now derive the project name from filepath.Base(--path)
// and never touch stdin.
func TestRunPresetFlow_NonInteractive(t *testing.T) {
	// Redirect stdin to an empty reader so any accidental AskInput call would
	// fail loudly instead of hanging the test.
	withClosedStdin(t)

	t.Run("derives name from --path basename", func(t *testing.T) {
		tmp := t.TempDir()
		projectDir := filepath.Join(tmp, "my-service")
		if err := os.Mkdir(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		profile, err := runPresetFlow("go-service", projectDir)
		if err != nil {
			t.Fatalf("runPresetFlow returned error: %v", err)
		}
		if profile.ProjectName != "my-service" {
			t.Errorf("ProjectName = %q, want %q", profile.ProjectName, "my-service")
		}
		if profile.OutputPath != projectDir {
			t.Errorf("OutputPath = %q, want %q", profile.OutputPath, projectDir)
		}
	})

	t.Run("uses cwd basename when --path is empty", func(t *testing.T) {
		// Save and restore cwd.
		orig, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(orig) })

		tmp := t.TempDir()
		projectDir := filepath.Join(tmp, "cwd-project")
		if err := os.Mkdir(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.Chdir(projectDir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		profile, err := runPresetFlow("go-service", "")
		if err != nil {
			t.Fatalf("runPresetFlow returned error: %v", err)
		}
		// On macOS, t.TempDir() can live under /var which is a symlink to
		// /private/var; filepath.Base is lexical, so compare by basename.
		if profile.ProjectName != "cwd-project" {
			t.Errorf("ProjectName = %q, want %q", profile.ProjectName, "cwd-project")
		}
	})

	t.Run("rejects root path", func(t *testing.T) {
		_, err := runPresetFlow("go-service", "/")
		if err == nil {
			t.Fatal("runPresetFlow(\"go-service\", \"/\") returned nil error; want explicit rejection")
		}
		if !strings.Contains(err.Error(), "cannot derive project name") {
			t.Errorf("error = %q, want substring %q", err.Error(), "cannot derive project name")
		}
	})

	t.Run("rejects non-kebab-case basename", func(t *testing.T) {
		// Paths whose basename is not kebab-case (uppercase, underscore, etc.)
		// should fail with a clear, non-panicking error.
		tmp := t.TempDir()
		badDir := filepath.Join(tmp, "Bad_Name")
		if err := os.Mkdir(badDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		_, err := runPresetFlow("go-service", badDir)
		if err == nil {
			t.Fatal("runPresetFlow with non-kebab-case basename returned nil error; want validation error")
		}
		if !strings.Contains(err.Error(), "kebab-case") {
			t.Errorf("error = %q, want substring %q", err.Error(), "kebab-case")
		}
	})

	t.Run("rejects unknown preset", func(t *testing.T) {
		_, err := runPresetFlow("nonexistent-preset", "/tmp/foo")
		if err == nil {
			t.Fatal("runPresetFlow with unknown preset returned nil error")
		}
		if !strings.Contains(err.Error(), "unknown preset") {
			t.Errorf("error = %q, want substring %q", err.Error(), "unknown preset")
		}
	})
}

// withClosedStdin replaces os.Stdin with an empty reader for the duration of
// the test, ensuring any accidental attempt to read user input fails fast
// rather than hanging. Restored via t.Cleanup.
func withClosedStdin(t *testing.T) {
	t.Helper()
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	// Close the write end immediately so reads return EOF.
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		_ = r.Close()
	})

	// Sanity: confirm the pipe read-end swapped into os.Stdin really returns
	// EOF with zero bytes read. Exercises os.Stdin directly so the assertion
	// tracks the actual thing under test, not a stand-in reader.
	var probe [1]byte
	n, err := os.Stdin.Read(probe[:])
	if n != 0 || err != io.EOF {
		t.Fatalf("closed stdin sanity: Read returned n=%d err=%v, want 0, io.EOF", n, err)
	}
}
