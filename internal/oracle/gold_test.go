package oracle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageToConventionDir(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Go variants
		{"go", "go"},
		{"golang", "go"},
		{"Go", "go"},
		{"go-internal", "go"},
		{"golang-microservice", "go"},

		// Node.js variants
		{"nodejs", "nodejs"},
		{"node", "nodejs"},
		{"javascript", "nodejs"},
		{"typescript", "nodejs"},
		{"nodejs-express", "nodejs"},
		{"node-api", "nodejs"},

		// Python
		{"python", "python"},
		{"python-django", "python"},

		// Dart/Flutter
		{"dart", "dart"},
		{"flutter", "dart"},
		{"dart-app", "dart"},
		{"flutter-mobile", "dart"},

		// Rust
		{"rust", "rust"},
		{"rust-actix", "rust"},

		// Ruby
		{"ruby", "ruby"},
		{"ruby-rails", "ruby"},

		// Unknown
		{"unknown", ""},
		{"java", ""},
		{"csharp", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := LanguageToConventionDir(tt.input)
			if result != tt.expected {
				t.Errorf("LanguageToConventionDir(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolveGoldStandardsDir_FlagPath_Valid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ResolveGoldStandardsDir("go-test", dir)
	if result != dir {
		t.Errorf("expected %q, got %q", dir, result)
	}
}

func TestResolveGoldStandardsDir_FlagPath_NoMd(t *testing.T) {
	dir := t.TempDir()
	// Create a non-md file
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not markdown"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ResolveGoldStandardsDir("go-test", dir)
	if result != "" {
		t.Errorf("expected empty string for dir with no .md files, got %q", result)
	}
}

func TestResolveGoldStandardsDir_FlagPath_NonExistent(t *testing.T) {
	result := ResolveGoldStandardsDir("go-test", "/nonexistent/path/that/does/not/exist")
	if result != "" {
		t.Errorf("expected empty string for nonexistent path, got %q", result)
	}
}

func TestResolveGoldStandardsDir_ConventionDir(t *testing.T) {
	// Create a temporary home directory structure
	tmpHome := t.TempDir()
	goldDir := filepath.Join(tmpHome, ".the-matrix", "gold-standards", "go")
	if err := os.MkdirAll(goldDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goldDir, "02-architecture.md"), []byte("# Architecture"), 0644); err != nil {
		t.Fatal(err)
	}

	// Override HOME for this test
	origHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", origHome) })
	os.Setenv("HOME", tmpHome)

	result := ResolveGoldStandardsDir("go-internal", "")
	if result != goldDir {
		t.Errorf("expected convention dir %q, got %q", goldDir, result)
	}
}

func TestResolveGoldStandardsDir_NoConventionDir(t *testing.T) {
	// Create a temporary home with no gold-standards dir
	tmpHome := t.TempDir()

	origHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", origHome) })
	os.Setenv("HOME", tmpHome)

	result := ResolveGoldStandardsDir("go-test", "")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestResolveGoldStandardsDir_FlagTakesPrecedence(t *testing.T) {
	// Set up both flag dir and convention dir
	flagDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(flagDir, "custom.md"), []byte("# Custom"), 0644); err != nil {
		t.Fatal(err)
	}

	tmpHome := t.TempDir()
	conventionDir := filepath.Join(tmpHome, ".the-matrix", "gold-standards", "go")
	if err := os.MkdirAll(conventionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conventionDir, "default.md"), []byte("# Default"), 0644); err != nil {
		t.Fatal(err)
	}

	origHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", origHome) })
	os.Setenv("HOME", tmpHome)

	result := ResolveGoldStandardsDir("go-test", flagDir)
	if result != flagDir {
		t.Errorf("flag dir should take precedence: expected %q, got %q", flagDir, result)
	}
}

func TestLoadGoldStandardsFromDir_Valid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "02-architecture.md"), []byte("# Architecture"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "04-errors.md"), []byte("# Errors"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "05-testing.md"), []byte("# Testing"), 0644); err != nil {
		t.Fatal(err)
	}

	result := LoadGoldStandardsFromDir(dir)

	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	if result["02-architecture"] != "# Architecture" {
		t.Errorf("wrong content for 02-architecture: %q", result["02-architecture"])
	}
	if result["04-errors"] != "# Errors" {
		t.Errorf("wrong content for 04-errors: %q", result["04-errors"])
	}
	if result["05-testing"] != "# Testing" {
		t.Errorf("wrong content for 05-testing: %q", result["05-testing"])
	}
}

func TestLoadGoldStandardsFromDir_MaxFiles(t *testing.T) {
	dir := t.TempDir()
	// Create 7 files — only 5 should be loaded
	for i := 0; i < 7; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("%02d-topic.md", i))
		if err := os.WriteFile(filename, []byte(fmt.Sprintf("# Topic %d", i)), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result := LoadGoldStandardsFromDir(dir)

	if len(result) != 5 {
		t.Errorf("expected 5 entries (max), got %d", len(result))
	}
	// Should have the first 5 alphabetically: 00, 01, 02, 03, 04
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("%02d-topic", i)
		if _, ok := result[key]; !ok {
			t.Errorf("expected key %q in result", key)
		}
	}
}

func TestLoadGoldStandardsFromDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := LoadGoldStandardsFromDir(dir)
	if len(result) != 0 {
		t.Errorf("expected empty map for empty dir, got %d entries", len(result))
	}
}

func TestLoadGoldStandardsFromDir_NonExistent(t *testing.T) {
	result := LoadGoldStandardsFromDir("/nonexistent/path")
	if len(result) != 0 {
		t.Errorf("expected empty map for nonexistent dir, got %d entries", len(result))
	}
}

func TestLoadGoldStandardsFromDir_EmptyString(t *testing.T) {
	result := LoadGoldStandardsFromDir("")
	if len(result) != 0 {
		t.Errorf("expected empty map for empty string, got %d entries", len(result))
	}
}

func TestGoldStandardRegistrationFiles(t *testing.T) {
	dir := t.TempDir()

	// Create files matching priority patterns
	files := []string{
		"00-core-principles.md",
		"01-project-structure.md",
		"02-layered-architecture.md",
		"04-error-handling.md",
		"05-testing.md",
		"06-database.md",
		"07-configuration.md",
		"_index.md", // should be excluded
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("# "+f), 0644); err != nil {
			t.Fatal(err)
		}
	}

	selected := GoldStandardRegistrationFiles(dir)

	if len(selected) != 5 {
		t.Fatalf("expected 5 selected files, got %d: %v", len(selected), selected)
	}

	// _index.md should never be selected
	for _, f := range selected {
		if f == "_index.md" {
			t.Error("_index.md should be excluded from registration")
		}
	}

	// Priority files should be selected
	priorityFiles := map[string]bool{
		"04-error-handling.md":       false,
		"05-testing.md":              false,
		"02-layered-architecture.md": false,
		"00-core-principles.md":      false,
		"01-project-structure.md":    false,
	}
	for _, f := range selected {
		priorityFiles[f] = true
	}
	for f, found := range priorityFiles {
		if !found {
			t.Errorf("expected priority file %q to be selected", f)
		}
	}
}
