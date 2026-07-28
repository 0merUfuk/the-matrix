package matrixcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_NoFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate from real global config
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// All pointer fields should be nil.
	if cfg.Models.Default != nil {
		t.Error("expected Models.Default to be nil")
	}
	if cfg.Morpheus.MaxCycles != nil {
		t.Error("expected Morpheus.MaxCycles to be nil")
	}
	if cfg.Trinity.MaxAgeMonths != nil {
		t.Error("expected Trinity.MaxAgeMonths to be nil")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	content := `
models:
  default: "claude-sonnet-4-20250514"
  manager: "claude-opus-4-20250514"
  architect: "claude-opus-4-20250514"
  developer: "claude-sonnet-4-20250514"
  tester: "claude-sonnet-4-20250514"
  reviewer: "claude-haiku-3-5"

neo:
  auto-oracle: true
  auto-trinity: false
  team-size: "small"

morpheus:
  max-cycles: 3
  developer-turns: 100
  tester-turns: 50
  reviewer-turns: 25

oracle:
  researcher-turns: 80
  reviewer-turns: 40
  synthesizer-turns: 60
  output-dir: "./output"

trinity:
  max-age-months: 3.5
`
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Models
	assertStringPtr(t, "Models.Default", cfg.Models.Default, "claude-sonnet-4-20250514")
	assertStringPtr(t, "Models.Manager", cfg.Models.Manager, "claude-opus-4-20250514")
	assertStringPtr(t, "Models.Architect", cfg.Models.Architect, "claude-opus-4-20250514")
	assertStringPtr(t, "Models.Developer", cfg.Models.Developer, "claude-sonnet-4-20250514")
	assertStringPtr(t, "Models.Tester", cfg.Models.Tester, "claude-sonnet-4-20250514")
	assertStringPtr(t, "Models.Reviewer", cfg.Models.Reviewer, "claude-haiku-3-5")

	// Neo
	assertBoolPtr(t, "Neo.AutoOracle", cfg.Neo.AutoOracle, true)
	assertBoolPtr(t, "Neo.AutoTrinity", cfg.Neo.AutoTrinity, false)
	assertStringPtr(t, "Neo.TeamSize", cfg.Neo.TeamSize, "small")

	// Morpheus
	assertIntPtr(t, "Morpheus.MaxCycles", cfg.Morpheus.MaxCycles, 3)
	assertIntPtr(t, "Morpheus.DeveloperTurns", cfg.Morpheus.DeveloperTurns, 100)
	assertIntPtr(t, "Morpheus.TesterTurns", cfg.Morpheus.TesterTurns, 50)
	assertIntPtr(t, "Morpheus.ReviewerTurns", cfg.Morpheus.ReviewerTurns, 25)

	// Oracle
	assertIntPtr(t, "Oracle.ResearcherTurns", cfg.Oracle.ResearcherTurns, 80)
	assertIntPtr(t, "Oracle.ReviewerTurns", cfg.Oracle.ReviewerTurns, 40)
	assertIntPtr(t, "Oracle.SynthesizerTurns", cfg.Oracle.SynthesizerTurns, 60)
	assertStringPtr(t, "Oracle.OutputDir", cfg.Oracle.OutputDir, "./output")

	// Trinity
	assertFloat64Ptr(t, "Trinity.MaxAgeMonths", cfg.Trinity.MaxAgeMonths, 3.5)
}

func TestLoad_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	content := `
morpheus:
  max-cycles: 5

trinity:
  max-age-months: 6.0
`
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Set fields
	assertIntPtr(t, "Morpheus.MaxCycles", cfg.Morpheus.MaxCycles, 5)
	assertFloat64Ptr(t, "Trinity.MaxAgeMonths", cfg.Trinity.MaxAgeMonths, 6.0)

	// Unset sections should have nil fields
	if cfg.Models.Default != nil {
		t.Error("expected Models.Default to be nil")
	}
	if cfg.Neo.TeamSize != nil {
		t.Error("expected Neo.TeamSize to be nil")
	}
	if cfg.Oracle.ResearcherTurns != nil {
		t.Error("expected Oracle.ResearcherTurns to be nil")
	}
	if cfg.Morpheus.DeveloperTurns != nil {
		t.Error("expected Morpheus.DeveloperTurns to be nil")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Models.Default != nil {
		t.Error("expected Models.Default to be nil for empty file")
	}
	if cfg.Morpheus.MaxCycles != nil {
		t.Error("expected Morpheus.MaxCycles to be nil for empty file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".matrix.yaml")
	// Use genuinely invalid YAML (tab indentation in a mapping value context).
	if err := os.WriteFile(path, []byte("models:\n\t- broken: {\n  invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	// Error should mention the file path.
	if got := err.Error(); !strings.Contains(got, path) {
		t.Errorf("error should contain file path %q, got %q", path, got)
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "negative max-age-months",
			content: `
trinity:
  max-age-months: -2.5
`,
			wantErr: "trinity.max-age-months must be > 0",
		},
		{
			name: "empty output-dir",
			content: `
oracle:
  output-dir: ""
`,
			wantErr: "oracle.output-dir must be non-empty",
		},
		{
			name: "multiple errors joined",
			content: `
trinity:
  max-age-months: -1
oracle:
  output-dir: ""
`,
			wantErr: "trinity.max-age-months must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load(dir)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}

	// Verify the "multiple errors joined" case contains both errors separated by "; ".
	t.Run("multiple errors joined contains both", func(t *testing.T) {
		dir := t.TempDir()
		content := `
trinity:
  max-age-months: -1
oracle:
  output-dir: ""
`
		if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(dir)
		if err == nil {
			t.Fatal("expected validation error")
		}
		got := err.Error()
		if !strings.Contains(got, "trinity.max-age-months must be > 0") {
			t.Errorf("expected trinity error in %q", got)
		}
		if !strings.Contains(got, "oracle.output-dir must be non-empty") {
			t.Errorf("expected oracle error in %q", got)
		}
		if !strings.Contains(got, "; ") {
			t.Errorf("expected errors joined with '; ' in %q", got)
		}
	})
}

func TestLoad_ProjectOverridesGlobal(t *testing.T) {
	// Create a fake home dir with a global config.
	fakeHome := t.TempDir()
	globalDir := filepath.Join(fakeHome, ".config", "the-matrix")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalContent := `
trinity:
  max-age-months: 12
`
	if err := os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(globalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a project-level config with a different value.
	projectDir := t.TempDir()
	projectContent := `
trinity:
  max-age-months: 3
`
	if err := os.WriteFile(filepath.Join(projectDir, ".matrix.yaml"), []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Redirect HOME so the global config is discoverable.
	t.Setenv("HOME", fakeHome)

	cfg, err := Load(projectDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Project value (3) must win over global (12).
	assertFloat64Ptr(t, "Trinity.MaxAgeMonths", cfg.Trinity.MaxAgeMonths, 3)
}

func TestLoad_GlobalFallback(t *testing.T) {
	// Redirect HOME to an isolated temp dir so a developer's real
	// ~/.config/the-matrix/config.yaml doesn't cause false failures.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	globalDir := filepath.Join(fakeHome, ".config", "the-matrix")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalContent := `
trinity:
  max-age-months: 9
`
	if err := os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(globalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Project dir has NO .matrix.yaml — should fall through to global.
	projectDir := t.TempDir()
	cfg, err := Load(projectDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertFloat64Ptr(t, "Trinity.MaxAgeMonths", cfg.Trinity.MaxAgeMonths, 9)
}

func TestLoad_YAMLExtensionVariant(t *testing.T) {
	dir := t.TempDir()
	content := `
morpheus:
  max-cycles: 7
`
	// Use .yml instead of .yaml.
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertIntPtr(t, "Morpheus.MaxCycles", cfg.Morpheus.MaxCycles, 7)
}

func TestLoad_UnknownKeys(t *testing.T) {
	dir := t.TempDir()
	content := `
morpheus:
  max-cycles: 3
  unknown-field: "should be ignored"
future-section:
  something: true
`
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error for unknown keys, got %v", err)
	}
	assertIntPtr(t, "Morpheus.MaxCycles", cfg.Morpheus.MaxCycles, 3)
}

func TestLoad_YAMLPriority(t *testing.T) {
	// When both .matrix.yaml and .matrix.yml exist, .yaml wins.
	dir := t.TempDir()
	yamlContent := `
morpheus:
  max-cycles: 100
`
	ymlContent := `
morpheus:
  max-cycles: 200
`
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yml"), []byte(ymlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertIntPtr(t, "Morpheus.MaxCycles", cfg.Morpheus.MaxCycles, 100)
}

// Test helpers

func assertStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected %q, got nil", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s: expected %q, got %q", field, want, *got)
	}
}

func assertIntPtr(t *testing.T, field string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected %d, got nil", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s: expected %d, got %d", field, want, *got)
	}
}

func assertFloat64Ptr(t *testing.T, field string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected %g, got nil", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s: expected %g, got %g", field, want, *got)
	}
}

func assertBoolPtr(t *testing.T, field string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected %v, got nil", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s: expected %v, got %v", field, want, *got)
	}
}

