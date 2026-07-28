package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromPathFlag(t *testing.T) {
	// Create a temp directory with a .matrix.yaml that sets max-age-months to 2.5.
	dir := t.TempDir()
	content := `
trinity:
  max-age-months: 2.5
`
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Isolate from real global config.
	t.Setenv("HOME", t.TempDir())

	cfg := loadConfig(dir)

	if cfg.Trinity.MaxAgeMonths == nil {
		t.Fatal("expected Trinity.MaxAgeMonths to be set from --path directory config")
	}
	if *cfg.Trinity.MaxAgeMonths != 2.5 {
		t.Errorf("expected max-age-months 2.5, got %g", *cfg.Trinity.MaxAgeMonths)
	}
}

func TestLoadConfig_FallsToCWDWhenEmpty(t *testing.T) {
	// Create a .matrix.yaml in a temp dir and chdir into it.
	dir := t.TempDir()
	content := `
trinity:
  max-age-months: 7.0
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

	// Empty pathFlag should load from CWD.
	cfg := loadConfig("")

	if cfg.Trinity.MaxAgeMonths == nil {
		t.Fatal("expected Trinity.MaxAgeMonths to be set from CWD config")
	}
	if *cfg.Trinity.MaxAgeMonths != 7.0 {
		t.Errorf("expected max-age-months 7.0, got %g", *cfg.Trinity.MaxAgeMonths)
	}
}

func TestLoadConfig_PathDirRespected(t *testing.T) {
	// Create two directories with different configs to prove --path wins.
	cwdDir := t.TempDir()
	pathDir := t.TempDir()

	cwdContent := `
trinity:
  max-age-months: 12
`
	pathContent := `
trinity:
  max-age-months: 3
`
	if err := os.WriteFile(filepath.Join(cwdDir, ".matrix.yaml"), []byte(cwdContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pathDir, ".matrix.yaml"), []byte(pathContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Isolate from real global config.
	t.Setenv("HOME", t.TempDir())

	// chdir to cwdDir so CWD has max-age-months=12
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	if err := os.Chdir(cwdDir); err != nil {
		t.Fatal(err)
	}

	// loadConfig with pathDir should use pathDir's config (3), not CWD's (12).
	cfg := loadConfig(pathDir)

	if cfg.Trinity.MaxAgeMonths == nil {
		t.Fatal("expected Trinity.MaxAgeMonths to be set")
	}
	if *cfg.Trinity.MaxAgeMonths != 3 {
		t.Errorf("expected max-age-months 3 from --path dir, got %g", *cfg.Trinity.MaxAgeMonths)
	}
}

func TestLoadConfig_NoConfigReturnsDefaults(t *testing.T) {
	// Empty directory with no config file.
	dir := t.TempDir()

	// Isolate from real global config.
	t.Setenv("HOME", t.TempDir())

	cfg := loadConfig(dir)

	// Should return zero-value config with nil pointer fields.
	if cfg.Trinity.MaxAgeMonths != nil {
		t.Errorf("expected nil MaxAgeMonths for missing config, got %g", *cfg.Trinity.MaxAgeMonths)
	}
}

func TestLoadConfig_InvalidConfigReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	// Write invalid YAML.
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte("invalid: {\nyaml"), 0644); err != nil {
		t.Fatal(err)
	}

	// Isolate from real global config.
	t.Setenv("HOME", t.TempDir())

	// Should not panic, should return zero config.
	cfg := loadConfig(dir)

	if cfg.Trinity.MaxAgeMonths != nil {
		t.Errorf("expected nil MaxAgeMonths for invalid config, got %g", *cfg.Trinity.MaxAgeMonths)
	}
}
