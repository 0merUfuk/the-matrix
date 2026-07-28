package oracle

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ── ParseTopicsFlag tests ──────────────────────────────────────

func TestParseTopicsFlag(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		expectErr bool
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace-only returns nil",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "single topic",
			input:    "04-error-handling",
			expected: []string{"04-error-handling"},
		},
		{
			name:     "two topics",
			input:    "04-error-handling,11-redis-caching",
			expected: []string{"04-error-handling", "11-redis-caching"},
		},
		{
			name:     "spaces around commas are trimmed",
			input:    "04-error-handling , 11-redis-caching , 05-testing",
			expected: []string{"04-error-handling", "11-redis-caching", "05-testing"},
		},
		{
			name:     "trailing comma is ignored",
			input:    "04-error-handling,",
			expected: []string{"04-error-handling"},
		},
		{
			name:     "leading comma produces non-empty entries only",
			input:    ",04-error-handling",
			expected: []string{"04-error-handling"},
		},
		{
			name:     "multiple commas between topics",
			input:    "04-error-handling,,11-redis-caching",
			expected: []string{"04-error-handling", "11-redis-caching"},
		},
		{
			name:      "path traversal rejected",
			input:     "../../../etc/passwd",
			expectErr: true,
		},
		{
			name:      "dotted path rejected",
			input:     "..secret",
			expectErr: true,
		},
		{
			name:      "slash in topic rejected",
			input:     "foo/bar",
			expectErr: true,
		},
		{
			name:      "topic starting with hyphen rejected",
			input:     "-leading-hyphen",
			expectErr: true,
		},
		{
			name:     "underscores are allowed",
			input:    "my_topic_01",
			expected: []string{"my_topic_01"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTopicsFlag(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result: %v)", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d topics, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("topic[%d]: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}

// ── validateWorkspace tests ────────────────────────────────────

func TestValidateWorkspace(t *testing.T) {
	t.Run("returns error for non-existent directory", func(t *testing.T) {
		err := validateWorkspace("/non/existent/path")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when config.sh is missing", func(t *testing.T) {
		dir := t.TempDir()
		// Create .autonomous dir but no config.sh
		if err := os.MkdirAll(filepath.Join(dir, ".autonomous"), 0755); err != nil {
			t.Fatal(err)
		}

		err := validateWorkspace(dir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns nil for valid workspace", func(t *testing.T) {
		dir := t.TempDir()
		autoDir := filepath.Join(dir, ".autonomous")
		if err := os.MkdirAll(autoDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(autoDir, "config.sh"), []byte("STACK_NAME=\"test\"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		err := validateWorkspace(dir)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

// ── discoverStaleTopics tests ──────────────────────────────────

func TestDiscoverStaleTopics(t *testing.T) {
	t.Run("returns nil for empty directory", func(t *testing.T) {
		dir := t.TempDir()
		result := discoverStaleTopics(dir, 6)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("returns nil for non-existent directory", func(t *testing.T) {
		result := discoverStaleTopics("/non/existent", 6)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("returns stale topics only", func(t *testing.T) {
		dir := t.TempDir()

		// Fresh doc (today's date)
		today := time.Now().Format("2006-01-02")
		freshContent := "**Version**: 1.0\n**Created**: " + today + "\n**Last Updated**: " + today + "\n**Authors:** test\n\n---\n\n# Fresh Doc\nContent here.\n"
		if err := os.WriteFile(filepath.Join(dir, "05-testing.md"), []byte(freshContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Stale doc (2 years ago)
		staleContent := "**Version**: 1.0\n**Created**: 2024-01-15\n**Last Updated**: 2024-01-15\n**Authors:** test\n\n---\n\n# Stale Doc\nContent here.\n"
		if err := os.WriteFile(filepath.Join(dir, "04-error-handling.md"), []byte(staleContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Another stale doc
		stale2Content := "**Version**: 1.0\n**Created**: 2024-03-01\n**Last Updated**: 2024-03-01\n**Authors:** test\n\n---\n\n# Stale Doc 2\nContent here.\n"
		if err := os.WriteFile(filepath.Join(dir, "11-redis-caching.md"), []byte(stale2Content), 0644); err != nil {
			t.Fatal(err)
		}

		// Doc with no date — should be skipped
		noDateContent := "# No Date Doc\nNo frontmatter here.\n"
		if err := os.WriteFile(filepath.Join(dir, "99-other.md"), []byte(noDateContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := discoverStaleTopics(dir, 6)

		// Should find 2 stale topics (sorted)
		if len(result) != 2 {
			t.Fatalf("expected 2 stale topics, got %d: %v", len(result), result)
		}
		if result[0] != "04-error-handling" {
			t.Errorf("expected 04-error-handling, got %s", result[0])
		}
		if result[1] != "11-redis-caching" {
			t.Errorf("expected 11-redis-caching, got %s", result[1])
		}
	})

	t.Run("returns all topics when all are stale", func(t *testing.T) {
		dir := t.TempDir()

		staleContent := "**Version**: 1.0\n**Created**: 2023-06-15\n**Last Updated**: 2023-06-15\n**Authors:** test\n\n---\n\n# Stale\n"
		if err := os.WriteFile(filepath.Join(dir, "00-core.md"), []byte(staleContent), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "01-structure.md"), []byte(staleContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := discoverStaleTopics(dir, 6)
		if len(result) != 2 {
			t.Fatalf("expected 2, got %d: %v", len(result), result)
		}
	})

	t.Run("returns nil when all docs are fresh", func(t *testing.T) {
		dir := t.TempDir()

		today := time.Now().Format("2006-01-02")
		freshContent := "**Version**: 1.0\n**Created**: " + today + "\n**Last Updated**: " + today + "\n**Authors:** test\n\n---\n\n# Fresh\n"
		if err := os.WriteFile(filepath.Join(dir, "00-core.md"), []byte(freshContent), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "01-structure.md"), []byte(freshContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := discoverStaleTopics(dir, 6)
		if len(result) != 0 {
			t.Errorf("expected 0 stale topics, got %d: %v", len(result), result)
		}
	})
}

// ── readStackNameFromConfig tests ──────────────────────────────

func TestReadStackNameFromConfig(t *testing.T) {
	t.Run("reads quoted stack name", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.sh")
		content := "#!/bin/bash\nSTACK_NAME=\"go-internal\"\nRESEARCH_CYCLES=4\n"
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result := readStackNameFromConfig(configPath)
		if result != "go-internal" {
			t.Errorf("expected go-internal, got %s", result)
		}
	})

	t.Run("reads single-quoted stack name", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.sh")
		content := "STACK_NAME='nodejs-express'\n"
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result := readStackNameFromConfig(configPath)
		if result != "nodejs-express" {
			t.Errorf("expected nodejs-express, got %s", result)
		}
	})

	t.Run("returns empty for missing file", func(t *testing.T) {
		result := readStackNameFromConfig("/nonexistent/config.sh")
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})

	t.Run("returns empty when STACK_NAME is not present", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.sh")
		content := "RESEARCH_CYCLES=4\nMODEL=sonnet\n"
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		result := readStackNameFromConfig(configPath)
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})
}

// ── scaffoldUpdateWorkspace tests ──────────────────────────────

func TestScaffoldUpdateWorkspace(t *testing.T) {
	t.Run("creates expected directory structure and files", func(t *testing.T) {
		// Set up a mock original workspace
		origDir := t.TempDir()
		autoDir := filepath.Join(origDir, ".autonomous")
		promptsDir := filepath.Join(autoDir, "prompts")
		if err := os.MkdirAll(promptsDir, 0755); err != nil {
			t.Fatal(err)
		}

		configContent := "STACK_NAME=\"go-test\"\nRESEARCH_CYCLES=4\nUPDATE_RESEARCH_CYCLES=2\n"
		if err := os.WriteFile(filepath.Join(autoDir, "config.sh"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "researcher.md"), []byte("researcher prompt"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "reviewer.md"), []byte("reviewer prompt"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(promptsDir, "synthesizer.md"), []byte("synthesizer prompt"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(origDir, "CLAUDE.md"), []byte("# Oracle Workspace"), 0644); err != nil {
			t.Fatal(err)
		}

		// Scaffold update workspace
		updateDir := filepath.Join(t.TempDir(), "update-2026-03-26")
		topics := []string{"04-error-handling", "11-redis-caching"}

		err := scaffoldUpdateWorkspace(origDir, updateDir, topics)
		if err != nil {
			t.Fatalf("scaffoldUpdateWorkspace failed: %v", err)
		}

		// Verify directory structure
		expectDirs := []string{
			filepath.Join(updateDir, ".autonomous", "prompts"),
			filepath.Join(updateDir, "tasks", "research"),
			filepath.Join(updateDir, "tasks", "review"),
			filepath.Join(updateDir, "tasks", "cycles"),
			filepath.Join(updateDir, "output", "go-test"),
		}
		for _, d := range expectDirs {
			if info, err := os.Stat(d); err != nil || !info.IsDir() {
				t.Errorf("expected directory to exist: %s", d)
			}
		}

		// Verify copied files
		expectFiles := []string{
			filepath.Join(updateDir, ".autonomous", "config.sh"),
			filepath.Join(updateDir, ".autonomous", "prompts", "researcher.md"),
			filepath.Join(updateDir, ".autonomous", "prompts", "reviewer.md"),
			filepath.Join(updateDir, ".autonomous", "prompts", "synthesizer.md"),
			filepath.Join(updateDir, "CLAUDE.md"),
			filepath.Join(updateDir, "topics.txt"),
			filepath.Join(updateDir, ".autonomous", "update.sh"),
			filepath.Join(updateDir, "tasks", "status.txt"),
		}
		for _, f := range expectFiles {
			if _, err := os.Stat(f); err != nil {
				t.Errorf("expected file to exist: %s", f)
			}
		}

		// Verify topics.txt content
		topicsData, err := os.ReadFile(filepath.Join(updateDir, "topics.txt"))
		if err != nil {
			t.Fatal(err)
		}
		expectedTopics := "04-error-handling\n11-redis-caching\n"
		if string(topicsData) != expectedTopics {
			t.Errorf("topics.txt: expected %q, got %q", expectedTopics, string(topicsData))
		}

		// Verify update.sh is executable
		info, err := os.Stat(filepath.Join(updateDir, ".autonomous", "update.sh"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0100 == 0 {
			t.Error("update.sh should be executable")
		}

		// Verify status.txt
		statusData, err := os.ReadFile(filepath.Join(updateDir, "tasks", "status.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(statusData) != "INITIALIZED\n" {
			t.Errorf("status.txt: expected INITIALIZED, got %q", string(statusData))
		}
	})
}

// ── injectUpdatedDocs tests ────────────────────────────────────

func TestInjectUpdatedDocs(t *testing.T) {
	// Helper to create a valid doc (50+ lines with required frontmatter)
	makeValidDoc := func(topic string) string {
		today := time.Now().Format("2006-01-02")
		lines := "**Version**: 1.0\n**Created**: " + today + "\n**Last Updated**: " + today + "\n**Authors:** oracle-cli (test)\n\n---\n\n"
		lines += "# " + topic + "\n\n"
		lines += "**Purpose**: Test doc\n**Audience**: Developers\n**Stack**: test\n\n---\n\n"
		lines += "## Overview\n\nThis is a test document for testing the inject flow.\n\n---\n\n"
		lines += "## Rules\n\n### Always\n- Rule 1\n- Rule 2\n- Rule 3\n\n### Never\n- Anti 1\n- Anti 2\n\n---\n\n"
		lines += "## Patterns\n\n### Pattern: Example\n\n```go\n// DO: correct\nfunc Example() {}\n```\n\n"
		lines += "```go\n// DON'T: wrong\nfunc Bad() {}\n```\n\n---\n\n"
		lines += "## Checklist\n\n- [ ] Item 1\n- [ ] Item 2\n- [ ] Item 3\n- [ ] Item 4\n- [ ] Item 5\n"
		return lines
	}

	t.Run("injects only matching topic docs", func(t *testing.T) {
		outputDir := t.TempDir()
		knowledgeDir := t.TempDir()

		// Write docs to output dir — one matching, one not
		if err := os.WriteFile(filepath.Join(outputDir, "04-error-handling.md"), []byte(makeValidDoc("Error Handling")), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "05-testing.md"), []byte(makeValidDoc("Testing")), 0644); err != nil {
			t.Fatal(err)
		}

		topics := []string{"04-error-handling"}
		injected, err := injectUpdatedDocs(outputDir, knowledgeDir, topics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(injected) != 1 {
			t.Fatalf("expected 1 injected, got %d: %v", len(injected), injected)
		}
		if injected[0] != "04-error-handling" {
			t.Errorf("expected 04-error-handling, got %s", injected[0])
		}

		// Verify the file was written to knowledge dir
		if _, err := os.Stat(filepath.Join(knowledgeDir, "04-error-handling.md")); err != nil {
			t.Error("expected 04-error-handling.md in knowledge dir")
		}
		// Verify the non-matching file was NOT written
		if _, err := os.Stat(filepath.Join(knowledgeDir, "05-testing.md")); err == nil {
			t.Error("05-testing.md should NOT be in knowledge dir")
		}
	})

	t.Run("skips docs that fail validation", func(t *testing.T) {
		outputDir := t.TempDir()
		knowledgeDir := t.TempDir()

		// Write a short/invalid doc
		shortContent := "# Too Short\nOnly a few lines.\n"
		if err := os.WriteFile(filepath.Join(outputDir, "04-error-handling.md"), []byte(shortContent), 0644); err != nil {
			t.Fatal(err)
		}

		topics := []string{"04-error-handling"}
		injected, err := injectUpdatedDocs(outputDir, knowledgeDir, topics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(injected) != 0 {
			t.Errorf("expected 0 injected (validation failure), got %d", len(injected))
		}
	})

	t.Run("returns empty when no matching files exist", func(t *testing.T) {
		outputDir := t.TempDir()
		knowledgeDir := t.TempDir()

		// Write a doc that doesn't match any topic
		if err := os.WriteFile(filepath.Join(outputDir, "00-core.md"), []byte(makeValidDoc("Core")), 0644); err != nil {
			t.Fatal(err)
		}

		topics := []string{"04-error-handling"}
		injected, err := injectUpdatedDocs(outputDir, knowledgeDir, topics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(injected) != 0 {
			t.Errorf("expected 0, got %d", len(injected))
		}
	})

	t.Run("returns error when output dir does not exist", func(t *testing.T) {
		nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")
		knowledgeDir := t.TempDir()

		topics := []string{"04-error-handling"}
		_, err := injectUpdatedDocs(nonExistentDir, knowledgeDir, topics)
		if err == nil {
			t.Fatal("expected error for non-existent output dir, got nil")
		}
		if !strings.Contains(err.Error(), "reading output directory") {
			t.Errorf("expected 'reading output directory' in error, got: %v", err)
		}
	})
}

// ── discoverStaleTopics — subdirectory and unreadable file edge cases ──

func TestDiscoverStaleTopics_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()

	// Create a subdirectory — should be skipped, not treated as a .md file
	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a valid stale doc alongside the subdirectory
	staleContent := "**Version**: 1.0\n**Created**: 2024-01-15\n**Last Updated**: 2024-01-15\n**Authors:** test\n\n---\n\n# Stale Doc\nContent here.\n"
	if err := os.WriteFile(filepath.Join(dir, "04-error-handling.md"), []byte(staleContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Also create a .md file inside the subdirectory — should NOT be discovered
	if err := os.WriteFile(filepath.Join(subDir, "nested.md"), []byte(staleContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := discoverStaleTopics(dir, 6)

	// Only the top-level .md file should be found; subdirectory is skipped
	if len(result) != 1 {
		t.Fatalf("expected 1 stale topic, got %d: %v", len(result), result)
	}
	if result[0] != "04-error-handling" {
		t.Errorf("expected 04-error-handling, got %s", result[0])
	}
}

// ── readStackNameFromConfig — edge cases ───────────────────────

func TestReadStackNameFromConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty value after equals",
			content:  "STACK_NAME=\nRESEARCH_CYCLES=4\n",
			expected: "",
		},
		{
			name:     "unquoted value",
			content:  "STACK_NAME=go-internal\n",
			expected: "go-internal",
		},
		{
			name:     "value with surrounding spaces in quotes",
			content:  "STACK_NAME=\" go-internal \"\n",
			expected: " go-internal ",
		},
		{
			name:     "STACK_NAME appears in a comment line — skipped",
			content:  "# STACK_NAME=commented-out\nSTACK_NAME=real-name\n",
			expected: "real-name",
		},
		{
			name:     "empty file",
			content:  "",
			expected: "",
		},
		{
			name:     "STACK_NAME key with extra whitespace before equals",
			content:  "STACK_NAME =bad-format\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.sh")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			result := readStackNameFromConfig(configPath)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// ── scaffoldUpdateWorkspace — no STACK_NAME in config ─────────

func TestScaffoldUpdateWorkspace_NoStackName(t *testing.T) {
	// Set up workspace with config.sh that has NO STACK_NAME
	origDir := t.TempDir()
	autoDir := filepath.Join(origDir, ".autonomous")
	promptsDir := filepath.Join(autoDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Config with no STACK_NAME — simulates a malformed workspace
	configContent := "RESEARCH_CYCLES=4\nUPDATE_RESEARCH_CYCLES=2\n"
	if err := os.WriteFile(filepath.Join(autoDir, "config.sh"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	updateDir := filepath.Join(t.TempDir(), "update-2026-03-26")
	topics := []string{"04-error-handling"}

	err := scaffoldUpdateWorkspace(origDir, updateDir, topics)
	if err != nil {
		t.Fatalf("scaffoldUpdateWorkspace should not fail for missing STACK_NAME, got: %v", err)
	}

	// The base update workspace structure must still exist
	expectDirs := []string{
		filepath.Join(updateDir, ".autonomous", "prompts"),
		filepath.Join(updateDir, "tasks", "research"),
		filepath.Join(updateDir, "tasks", "review"),
		filepath.Join(updateDir, "tasks", "cycles"),
	}
	for _, d := range expectDirs {
		if info, err := os.Stat(d); err != nil || !info.IsDir() {
			t.Errorf("expected directory to exist: %s", d)
		}
	}

	// output/ dir for the stack should NOT be created (no stack name to use)
	outputDir := filepath.Join(updateDir, "output")
	if _, err := os.Stat(outputDir); err == nil {
		// output/ itself may not exist OR may exist but with no subdirs
		entries, _ := os.ReadDir(outputDir)
		if len(entries) > 0 {
			t.Errorf("expected no output subdirectory when STACK_NAME is empty, found: %v", entries)
		}
	}
}

// ── injectUpdatedDocs — atomic rollback on failure (P0-C4) ────

// makeValidInjectDoc returns a doc that satisfies ValidateInjectDoc so tests
// can exercise the write path (not the validation path).
func makeValidInjectDoc(topic string) string {
	today := time.Now().Format("2006-01-02")
	lines := "**Version**: 1.0\n**Created**: " + today + "\n**Last Updated**: " + today + "\n**Authors:** oracle-cli (test)\n\n---\n\n"
	lines += "# " + topic + "\n\n"
	lines += "**Purpose**: Test doc\n**Audience**: Developers\n**Stack**: test\n\n---\n\n"
	lines += "## Overview\n\nThis is a test document for testing the inject flow.\n\n---\n\n"
	lines += "## Rules\n\n### Always\n- Rule 1\n- Rule 2\n- Rule 3\n\n### Never\n- Anti 1\n- Anti 2\n\n---\n\n"
	lines += "## Patterns\n\n### Pattern: Example\n\n```go\n// DO: correct\nfunc Example() {}\n```\n\n"
	lines += "```go\n// DON'T: wrong\nfunc Bad() {}\n```\n\n---\n\n"
	lines += "## Checklist\n\n- [ ] Item 1\n- [ ] Item 2\n- [ ] Item 3\n- [ ] Item 4\n- [ ] Item 5\n"
	return lines
}

// TestInjectUpdatedDocs_RollbackOnWriteFailure asserts that when a write
// mid-way through injection fails, the knowledge directory is rolled back:
// pre-existing files are restored and newly created files are removed.
// Pattern: force the second write to fail by placing a directory at its
// destination path (WriteFile to a directory returns EISDIR on Unix).
func TestInjectUpdatedDocs_RollbackOnWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("WriteFile semantics over a directory differ on Windows")
	}

	outputDir := t.TempDir()
	knowledgeDir := t.TempDir()

	// Candidate 1: overwrites an existing doc with prior contents we can verify
	// was restored by rollback.
	priorContents := []byte("PRIOR CONTENTS — must be restored after rollback\n")
	if err := os.WriteFile(filepath.Join(knowledgeDir, "04-error-handling.md"), priorContents, 0644); err != nil {
		t.Fatal(err)
	}

	// Candidate 2: destination is a directory, so WriteFile will fail.
	if err := os.Mkdir(filepath.Join(knowledgeDir, "11-redis-caching.md"), 0755); err != nil {
		t.Fatal(err)
	}

	// Two valid source docs — ListMDFiles sorts alphabetically, so 04 runs
	// before 11 and we'll have one successful write before the failure.
	if err := os.WriteFile(filepath.Join(outputDir, "04-error-handling.md"), []byte(makeValidInjectDoc("Error Handling")), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "11-redis-caching.md"), []byte(makeValidInjectDoc("Redis Caching")), 0644); err != nil {
		t.Fatal(err)
	}

	topics := []string{"04-error-handling", "11-redis-caching"}
	injected, err := injectUpdatedDocs(outputDir, knowledgeDir, topics)
	if err == nil {
		t.Fatalf("expected error when second write fails, got nil (injected=%v)", injected)
	}
	if len(injected) != 0 {
		t.Errorf("expected injected to be empty on failure (rolled back), got %v", injected)
	}

	// Rollback: the existing file's contents must be restored.
	got, rerr := os.ReadFile(filepath.Join(knowledgeDir, "04-error-handling.md"))
	if rerr != nil {
		t.Fatalf("rollback should have restored 04-error-handling.md: %v", rerr)
	}
	if string(got) != string(priorContents) {
		t.Errorf("rollback did not restore prior contents\n  got:  %q\n  want: %q", string(got), string(priorContents))
	}
}

func TestInjectUpdatedDocs_RollbackRemovesNewlyCreatedFilesOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("WriteFile semantics over a directory differ on Windows")
	}

	outputDir := t.TempDir()
	knowledgeDir := t.TempDir()

	// 04-error-handling.md: does NOT pre-exist in knowledgeDir (new file)
	// 11-redis-caching.md:  destination is a directory → WriteFile fails
	if err := os.Mkdir(filepath.Join(knowledgeDir, "11-redis-caching.md"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(outputDir, "04-error-handling.md"), []byte(makeValidInjectDoc("Error Handling")), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "11-redis-caching.md"), []byte(makeValidInjectDoc("Redis Caching")), 0644); err != nil {
		t.Fatal(err)
	}

	topics := []string{"04-error-handling", "11-redis-caching"}
	_, err := injectUpdatedDocs(outputDir, knowledgeDir, topics)
	if err == nil {
		t.Fatal("expected error when second write fails, got nil")
	}

	// Newly created file must have been removed by rollback.
	if _, statErr := os.Stat(filepath.Join(knowledgeDir, "04-error-handling.md")); !os.IsNotExist(statErr) {
		t.Errorf("rollback should have removed newly created 04-error-handling.md (stat err: %v)", statErr)
	}
}

// ── runUpdateScript — timeout kills the subprocess (P0-C3) ────

func TestRunUpdateScriptWithTimeout_KillsOnDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash not available on Windows CI")
	}

	updateDir := t.TempDir()
	autoDir := filepath.Join(updateDir, ".autonomous")
	if err := os.MkdirAll(autoDir, 0755); err != nil {
		t.Fatal(err)
	}

	// A script that sleeps far longer than our test timeout. If runUpdateScript
	// does not enforce the deadline, this test will hang for 30 seconds and
	// trip the 10s timeout below.
	scriptPath := filepath.Join(autoDir, "update.sh")
	script := "#!/bin/bash\nsleep 30\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- runUpdateScriptWithTimeout(updateDir, 500*time.Millisecond)
	}()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("expected 'timed out' in error, got: %v", err)
		}
		// The kill should happen close to the 500ms deadline, well under 10s.
		if elapsed > 10*time.Second {
			t.Errorf("runUpdateScriptWithTimeout took %s — kill did not fire promptly", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("runUpdateScriptWithTimeout did not return within 10s — subprocess was not killed")
	}
}
