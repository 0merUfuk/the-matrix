package morpheus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0merUfuk/the-matrix/internal/cli"
)

// createFile is a test helper that creates a file with the given content and permissions.
// It explicitly sets chmod after writing because os.WriteFile only applies perm at
// creation time — overwriting an existing file does not change its mode.
func createFile(t *testing.T, dir, relPath, content string, perm os.FileMode) {
	t.Helper()
	absPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		t.Fatalf("creating directory for %s: %v", relPath, err)
	}
	if err := os.WriteFile(absPath, []byte(content), perm); err != nil {
		t.Fatalf("writing %s: %v", relPath, err)
	}
	if err := os.Chmod(absPath, perm); err != nil {
		t.Fatalf("chmod %s: %v", relPath, err)
	}
}

// createCoreFiles creates all CORE_FILES with valid content and appropriate permissions.
func createCoreFiles(t *testing.T, dir string) {
	t.Helper()
	for _, f := range CORE_FILES {
		perm := os.FileMode(0644)
		if EXEC_FILES[f] {
			perm = 0755
		}
		createFile(t, dir, f, "valid content", perm)
	}
}

// createInitOnlyFiles creates all INIT_ONLY_FILES with valid content.
// settings.json gets valid JSON with permissions.allow.
func createInitOnlyFiles(t *testing.T, dir string) {
	t.Helper()
	for _, f := range INIT_ONLY_FILES {
		perm := os.FileMode(0644)
		if EXEC_FILES[f] {
			perm = 0755
		}
		content := "valid content"
		if f == ".claude/settings.json" {
			content = `{"permissions": {"allow": ["Bash(*)"]}}`
		}
		createFile(t, dir, f, content, perm)
	}
}

func TestCheckDoctor(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		wantPass    int
		wantFail    int
		wantWarn    int
		notAService bool
	}{
		{
			name: "not a morpheus service",
			setup: func(t *testing.T, dir string) {
				// Empty directory — no .autonomous/loop.sh
			},
			notAService: true,
		},
		{
			name: "init-scaffolded project with all files present",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createInitOnlyFiles(t, dir)
				// Add tech detection rule so we don't get a tech detection warn
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) + len(INIT_ONLY_FILES),
			wantFail: 0,
			wantWarn: 0,
		},
		{
			name: "loop-only project with all core files",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				// No init-only files, no finalizer.md marker
				// Add tech detection rule
				createFile(t, dir, ".claude/rules/node-service.md", "node rules", 0644)
			},
			wantPass: len(CORE_FILES),
			wantFail: 0,
			// tech detection passes because node-service.md exists
			wantWarn: 0,
		},
		{
			name: "missing core file",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				// Remove config.sh
				if err := os.Remove(filepath.Join(dir, ".autonomous/config.sh")); err != nil {
					t.Fatal(err)
				}
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) - 1,
			wantFail: 1,
			wantWarn: 0,
		},
		{
			name: "missing init-only file in init project",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createInitOnlyFiles(t, dir)
				// Remove CLAUDE.md (an init-only file)
				if err := os.Remove(filepath.Join(dir, "CLAUDE.md")); err != nil {
					t.Fatal(err)
				}
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) + len(INIT_ONLY_FILES) - 1,
			wantFail: 1,
			wantWarn: 0,
		},
		{
			name: "non-executable shell script",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				// Overwrite loop.sh with 0644 (not executable)
				createFile(t, dir, ".autonomous/loop.sh", "#!/bin/bash", 0644)
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) - 1, // all except loop.sh
			wantFail: 0,
			wantWarn: 1, // loop.sh not executable
		},
		{
			name: "invalid settings.json",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createInitOnlyFiles(t, dir)
				// Overwrite settings.json with invalid JSON
				createFile(t, dir, ".claude/settings.json", "not json {{{", 0644)
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) + len(INIT_ONLY_FILES) - 1,
			wantFail: 1, // invalid JSON
			wantWarn: 0,
		},
		{
			name: "skeleton todo.md detection",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				// Overwrite todo.md with skeleton content
				createFile(t, dir, "tasks/todo.md", "This is a skeleton template for tasks.", 0644)
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) - 1, // all except todo.md
			wantFail: 0,
			wantWarn: 1, // skeleton todo.md
		},
		{
			name: "missing permissions key in settings.json",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createInitOnlyFiles(t, dir)
				// Overwrite settings.json with valid JSON but no permissions key
				createFile(t, dir, ".claude/settings.json", `{"other": "value"}`, 0644)
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) + len(INIT_ONLY_FILES) - 1,
			wantFail: 0,
			wantWarn: 1, // missing permissions key
		},
		{
			name: "tech detection warns when neither rule file exists",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				// No .claude/rules/go-service.md or node-service.md
			},
			wantPass: len(CORE_FILES),
			wantFail: 0,
			wantWarn: 1, // tech detection
		},
		{
			name: "tech detection passes with go-service.md",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES),
			wantFail: 0,
			wantWarn: 0,
		},
		{
			name: "tech detection passes with node-service.md",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createFile(t, dir, ".claude/rules/node-service.md", "node rules", 0644)
			},
			wantPass: len(CORE_FILES),
			wantFail: 0,
			wantWarn: 0,
		},
		{
			name: "multiple failures accumulate",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createInitOnlyFiles(t, dir)
				// Remove two init-only files
				if err := os.Remove(filepath.Join(dir, "CLAUDE.md")); err != nil { t.Fatal(err) }
				if err := os.Remove(filepath.Join(dir, ".claude/SERVICE_CONTEXT.md")); err != nil { t.Fatal(err) }
				// Make loop.sh non-executable
				createFile(t, dir, ".autonomous/loop.sh", "#!/bin/bash", 0644)
				// No tech detection rule
			},
			// loop.sh: warn (not executable)
			// CLAUDE.md: fail (missing)
			// SERVICE_CONTEXT.md: fail (missing)
			// tech detection: warn
			// pass: core(8) + init(8) - loop.sh(1) - CLAUDE.md(1) - SERVICE_CONTEXT.md(1) = 13
			wantPass: len(CORE_FILES) + len(INIT_ONLY_FILES) - 3,
			wantFail: 2,
			wantWarn: 2,
		},
		{
			name: "missing permissions.allow array",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createInitOnlyFiles(t, dir)
				// permissions key exists but allow is not an array
				createFile(t, dir, ".claude/settings.json", `{"permissions": {"deny": []}}`, 0644)
				createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)
			},
			wantPass: len(CORE_FILES) + len(INIT_ONLY_FILES) - 1,
			wantFail: 1, // missing permissions.allow array
			wantWarn: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)

			result := CheckDoctor(dir)

			if result.NotAService != tt.notAService {
				t.Errorf("NotAService = %v, want %v", result.NotAService, tt.notAService)
			}
			if tt.notAService {
				return // no further checks for non-service directories
			}
			if result.Pass != tt.wantPass {
				t.Errorf("Pass = %d, want %d", result.Pass, tt.wantPass)
				for _, m := range result.Messages {
					t.Logf("  %v: %s -> %s", m.Status, m.Label, m.Detail)
				}
			}
			if result.Fail != tt.wantFail {
				t.Errorf("Fail = %d, want %d", result.Fail, tt.wantFail)
				for _, m := range result.Messages {
					t.Logf("  %v: %s -> %s", m.Status, m.Label, m.Detail)
				}
			}
			if result.Warn != tt.wantWarn {
				t.Errorf("Warn = %d, want %d", result.Warn, tt.wantWarn)
				for _, m := range result.Messages {
					t.Logf("  %v: %s -> %s", m.Status, m.Label, m.Detail)
				}
			}
		})
	}
}

// TestCheckDoctorMessageDetails verifies that specific messages contain
// expected detail strings for key check scenarios.
func TestCheckDoctorMessageDetails(t *testing.T) {
	t.Run("missing file detail says MISSING", func(t *testing.T) {
		dir := t.TempDir()
		createCoreFiles(t, dir)
		if err := os.Remove(filepath.Join(dir, ".autonomous/config.sh")); err != nil { t.Fatal(err) }
		createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)

		result := CheckDoctor(dir)
		found := false
		for _, m := range result.Messages {
			if m.Status == cli.Fail && m.Detail == "MISSING" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected a Fail message with detail 'MISSING'")
		}
	})

	t.Run("non-executable detail says chmod +x", func(t *testing.T) {
		dir := t.TempDir()
		createCoreFiles(t, dir)
		createFile(t, dir, ".autonomous/loop.sh", "#!/bin/bash", 0644)
		createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)

		result := CheckDoctor(dir)
		found := false
		for _, m := range result.Messages {
			if m.Status == cli.Warn && m.Detail == "not executable \u2014 chmod +x" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected a Warn message with detail 'not executable \u2014 chmod +x'")
		}
	})

	t.Run("skeleton todo detail mentions skeleton template", func(t *testing.T) {
		dir := t.TempDir()
		createCoreFiles(t, dir)
		createFile(t, dir, "tasks/todo.md", "This is a skeleton template", 0644)
		createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)

		result := CheckDoctor(dir)
		found := false
		for _, m := range result.Messages {
			if m.Status == cli.Warn && m.Detail == "skeleton template (Claude was unavailable)" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected a Warn message with detail 'skeleton template (Claude was unavailable)'")
		}
	})

	t.Run("missing permissions key detail", func(t *testing.T) {
		dir := t.TempDir()
		createCoreFiles(t, dir)
		createInitOnlyFiles(t, dir)
		createFile(t, dir, ".claude/settings.json", `{"other": true}`, 0644)
		createFile(t, dir, ".claude/rules/go-service.md", "go rules", 0644)

		result := CheckDoctor(dir)
		found := false
		for _, m := range result.Messages {
			if m.Status == cli.Warn && m.Detail == "missing 'permissions' key" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected a Warn message with detail \"missing 'permissions' key\"")
		}
	})
}

// TestCheckDoctorResultConsistency verifies that the result always satisfies:
// Pass + Fail + Warn == len(Messages)
func TestCheckDoctorResultConsistency(t *testing.T) {
	dirs := []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{
			name: "all pass",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createFile(t, dir, ".claude/rules/go-service.md", "", 0644)
			},
		},
		{
			name: "mixed results",
			setup: func(t *testing.T, dir string) {
				createCoreFiles(t, dir)
				createInitOnlyFiles(t, dir)
				if err := os.Remove(filepath.Join(dir, "CLAUDE.md")); err != nil { t.Fatal(err) }
				createFile(t, dir, ".autonomous/loop.sh", "#!/bin/bash", 0644)
			},
		},
	}

	for _, d := range dirs {
		t.Run(d.name, func(t *testing.T) {
			dir := t.TempDir()
			d.setup(t, dir)

			result := CheckDoctor(dir)
			total := result.Pass + result.Fail + result.Warn
			if total != len(result.Messages) {
				t.Errorf("Pass(%d) + Fail(%d) + Warn(%d) = %d, but len(Messages) = %d",
					result.Pass, result.Fail, result.Warn, total, len(result.Messages))
			}
		})
	}
}
