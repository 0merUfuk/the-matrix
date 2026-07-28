package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromCWD(t *testing.T) {
	// Create a .matrix.yaml in a temp dir and chdir into it.
	dir := t.TempDir()
	content := `
oracle:
  output-dir: "./custom-output"
`
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Isolate from real global config.
	t.Setenv("HOME", t.TempDir())

	// Save and restore cwd.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()

	if cfg.Oracle.OutputDir == nil {
		t.Fatal("expected Oracle.OutputDir to be set from CWD config")
	}
	if *cfg.Oracle.OutputDir != "./custom-output" {
		t.Errorf("expected output-dir './custom-output', got %q", *cfg.Oracle.OutputDir)
	}
}

func TestLoadConfig_NoConfigReturnsDefaults(t *testing.T) {
	// chdir to an empty temp dir.
	dir := t.TempDir()

	// Isolate from real global config.
	t.Setenv("HOME", t.TempDir())

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()

	if cfg.Oracle.OutputDir != nil {
		t.Errorf("expected nil OutputDir for missing config, got %q", *cfg.Oracle.OutputDir)
	}
}

func TestLoadConfig_InvalidConfigReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte("invalid: {\nyaml"), 0644); err != nil {
		t.Fatal(err)
	}

	// Isolate from real global config.
	t.Setenv("HOME", t.TempDir())

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := loadConfig()

	if cfg.Oracle.OutputDir != nil {
		t.Errorf("expected nil OutputDir for invalid config, got %q", *cfg.Oracle.OutputDir)
	}
}
