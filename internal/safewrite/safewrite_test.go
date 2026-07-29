package safewrite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutputPathRejectsProtectedValues(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("get home directory: %v", err)
	}

	for _, dir := range []string{"", "/", home} {
		t.Run(dir, func(t *testing.T) {
			if _, err := ResolveOutputPath(dir); err == nil {
				t.Errorf("ResolveOutputPath(%q) returned nil error", dir)
			}
		})
	}
}

func TestResolveOutputPathReturnsAbsolutePathForRelativeDir(t *testing.T) {
	const dir = "matrix-safe-output"

	got, err := ResolveOutputPath(dir)
	if err != nil {
		t.Fatalf("ResolveOutputPath(%q): %v", dir, err)
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("resolve expected path: %v", err)
	}
	if got != want {
		t.Errorf("ResolveOutputPath(%q) = %q, want %q", dir, got, want)
	}
}

func TestIsProtectedPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("get home directory: %v", err)
	}

	for _, path := range []string{"", "/", home} {
		t.Run("protected_"+path, func(t *testing.T) {
			if !IsProtectedPath(path) {
				t.Errorf("IsProtectedPath(%q) = false, want true", path)
			}
		})
	}

	tempDir := t.TempDir()
	if IsProtectedPath(tempDir) {
		t.Errorf("IsProtectedPath(%q) = true, want false", tempDir)
	}
}

func TestConfirmOverwriteAllowsMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")

	got, err := ConfirmOverwrite(OverwriteOpts{Path: path})
	if err != nil {
		t.Fatalf("ConfirmOverwrite() error: %v", err)
	}
	if !got {
		t.Error("ConfirmOverwrite() = false, want true for a missing directory")
	}
}

func TestConfirmOverwriteAllowsForceForExistingDirectory(t *testing.T) {
	got, err := ConfirmOverwrite(OverwriteOpts{Path: t.TempDir(), Force: true})
	if err != nil {
		t.Fatalf("ConfirmOverwrite() error: %v", err)
	}
	if !got {
		t.Error("ConfirmOverwrite() = false, want true when force is set")
	}
}

func TestConfirmOverwriteRejectsExistingDirectoryWithoutTTYOrForce(t *testing.T) {
	got, err := ConfirmOverwrite(OverwriteOpts{Path: t.TempDir(), IsTTY: false})
	if err != nil {
		t.Fatalf("ConfirmOverwrite() error: %v", err)
	}
	if got {
		t.Error("ConfirmOverwrite() = true, want false without TTY or force")
	}
}

func TestConfirmOverwritePromptsInInteractiveMode(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  bool
	}{
		{name: "yes", input: "y\n", want: true},
		{name: "no", input: "n\n", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withStdin(t, tc.input)

			got, err := ConfirmOverwrite(OverwriteOpts{Path: t.TempDir(), IsTTY: true})
			if err != nil {
				t.Fatalf("ConfirmOverwrite() error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ConfirmOverwrite() = %t, want %t", got, tc.want)
			}
		})
	}
}

func withStdin(t *testing.T, input string) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := writer.WriteString(input); err != nil {
		reader.Close()
		writer.Close()
		t.Fatalf("write stdin pipe: %v", err)
	}
	if err := writer.Close(); err != nil {
		reader.Close()
		t.Fatalf("close stdin writer: %v", err)
	}

	original := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = original
		reader.Close()
	})
}
