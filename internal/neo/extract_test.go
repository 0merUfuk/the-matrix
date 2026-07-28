package neo

import (
	"os"
	"path/filepath"
	"testing"
)

// ── parseMakefileTargets ──────────────────────────────────────────────────────

func TestParseMakefileTargets(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name: "standard Makefile with dev test build",
			content: ".PHONY: dev test build\n\ndev:\n\tgo run ./cmd/...\n\ntest:\n\tgo test ./...\n\nbuild:\n\tgo build ./...\n",
			want:    []string{"dev", "test", "build"},
		},
		{
			name:    "empty Makefile returns empty slice",
			content: "",
			want:    nil,
		},
		{
			name: "comment lines are skipped",
			content: "# This is a comment\nbuild:\n\tgo build\n",
			want:    []string{"build"},
		},
		{
			name: "recipe lines (tab-prefixed) are skipped",
			content: "build:\n\tgo build ./...\n\techo done\n",
			want:    []string{"build"},
		},
		{
			name: "dot-prefixed special targets are skipped",
			content: ".PHONY: build test\nbuild:\n\tgo build\ntest:\n\tgo test\n",
			want:    []string{"build", "test"},
		},
		{
			name: "variable assignments are skipped",
			content: "VERSION := 1.0\nBIN_NAME = myapp\nbuild:\n\tgo build\n",
			want:    []string{"build"},
		},
		{
			name: "target with dependencies",
			content: "build: deps\n\tgo build\ndeps:\n\tgo mod download\n",
			want:    []string{"build", "deps"},
		},
		{
			name: "duplicate targets appear only once",
			content: "build:\n\tgo build\nbuild:\n\tgo build -v\n",
			want:    []string{"build"},
		},
		{
			name: "hyphenated and underscored target names",
			content: "dev-standalone:\n\tgo run main.go\nrun_tests:\n\tgo test ./...\n",
			want:    []string{"dev-standalone", "run_tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMakefileTargets(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("parseMakefileTargets() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseMakefileTargets()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ── extractBuildCommands ──────────────────────────────────────────────────────

func TestExtractBuildCommands(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		wantSubset  map[string]string // subset of expected commands
		wantAbsent  []string          // command names that must NOT be present
	}{
		{
			name: "Makefile with dev test build targets",
			files: map[string]string{
				"Makefile": ".PHONY: dev test build\n\ndev:\n\tgo run ./cmd/...\n\ntest:\n\tgo test ./...\n\nbuild:\n\tgo build ./...\n",
			},
			wantSubset: map[string]string{
				"dev":   "make dev",
				"test":  "make test",
				"build": "make build",
			},
		},
		{
			name: "package.json with npm scripts",
			files: map[string]string{
				"package.json": `{"scripts":{"dev":"node server.js","test":"jest","build":"tsc","lint":"eslint ."}}`,
			},
			wantSubset: map[string]string{
				"dev":   "npm run dev",
				"test":  "npm run test",
				"build": "npm run build",
				"lint":  "npm run lint",
			},
		},
		{
			name: "Makefile takes precedence over package.json for common script names",
			files: map[string]string{
				"Makefile":     "test:\n\tgo test ./...\n",
				"package.json": `{"scripts":{"test":"jest"}}`,
			},
			wantSubset: map[string]string{
				"test": "make test", // Makefile entries are never overwritten by npm scripts
			},
		},
		{
			name: "Makefile takes precedence over package.json for non-common script names",
			files: map[string]string{
				"Makefile":     "migrate:\n\tgo run ./cmd/migrate\n",
				"package.json": `{"scripts":{"migrate":"node migrate.js"}}`,
			},
			wantSubset: map[string]string{
				"migrate": "make migrate", // non-standard name — Makefile wins via default branch
			},
		},
		{
			name:       "no build files returns empty map",
			files:      map[string]string{},
			wantSubset: map[string]string{},
		},
		{
			name: "malformed package.json is ignored gracefully",
			files: map[string]string{
				"package.json": `{invalid json`,
			},
			wantSubset: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := extractBuildCommands(dir)

			for key, wantVal := range tt.wantSubset {
				if gotVal, ok := got[key]; !ok {
					t.Errorf("missing command %q in result", key)
				} else if gotVal != wantVal {
					t.Errorf("command[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}

			for _, absent := range tt.wantAbsent {
				if _, ok := got[absent]; ok {
					t.Errorf("command %q should not be present", absent)
				}
			}
		})
	}
}

func TestExtractBuildCommands_ReturnsMap(t *testing.T) {
	// Always returns a non-nil map, even with no files
	dir := t.TempDir()
	got := extractBuildCommands(dir)
	if got == nil {
		t.Error("extractBuildCommands() returned nil, want empty map")
	}
}

// ── extractCICD ───────────────────────────────────────────────────────────────

func TestExtractCICD(t *testing.T) {
	tests := []struct {
		name           string
		dirs           []string
		files          map[string]string
		wantGitHub     bool
		wantGitLab     bool
		wantCircleCI   bool
	}{
		{
			name:         "GitHub Actions .github/workflows dir",
			dirs:         []string{".github/workflows"},
			wantGitHub:   true,
			wantGitLab:   false,
			wantCircleCI: false,
		},
		{
			name:         "GitLab .gitlab-ci.yml file",
			files:        map[string]string{".gitlab-ci.yml": "stages:\n  - test\n"},
			wantGitHub:   false,
			wantGitLab:   true,
			wantCircleCI: false,
		},
		{
			name:         "CircleCI .circleci dir",
			dirs:         []string{".circleci"},
			wantGitHub:   false,
			wantGitLab:   false,
			wantCircleCI: true,
		},
		{
			name:         "all three present",
			dirs:         []string{".github/workflows", ".circleci"},
			files:        map[string]string{".gitlab-ci.yml": "stages:\n  - test\n"},
			wantGitHub:   true,
			wantGitLab:   true,
			wantCircleCI: true,
		},
		{
			name:         "no CI/CD",
			wantGitHub:   false,
			wantGitLab:   false,
			wantCircleCI: false,
		},
		{
			name: ".github exists but no workflows subdir — not GitHub Actions",
			dirs: []string{".github"},
			// .github exists but .github/workflows does not
			wantGitHub:   false,
			wantGitLab:   false,
			wantCircleCI: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, d := range tt.dirs {
				if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
					t.Fatal(err)
				}
			}
			for name, content := range tt.files {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			hasGitHub, hasGitLab, hasCircleCI := extractCICD(dir)

			if hasGitHub != tt.wantGitHub {
				t.Errorf("hasGitHub = %v, want %v", hasGitHub, tt.wantGitHub)
			}
			if hasGitLab != tt.wantGitLab {
				t.Errorf("hasGitLab = %v, want %v", hasGitLab, tt.wantGitLab)
			}
			if hasCircleCI != tt.wantCircleCI {
				t.Errorf("hasCircleCI = %v, want %v", hasCircleCI, tt.wantCircleCI)
			}
		})
	}
}

// ── extractExistingAI ─────────────────────────────────────────────────────────

func TestExtractExistingAI(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantBody string // substring that must appear in result
		wantEmpty bool
	}{
		{
			name: "CLAUDE.md takes priority",
			files: map[string]string{
				"CLAUDE.md":    "# My Claude instructions\n",
				".cursorrules": "cursor rules here\n",
				"AGENTS.md":    "agents here\n",
			},
			wantBody: "My Claude instructions",
		},
		{
			name: ".cursorrules used when CLAUDE.md absent",
			files: map[string]string{
				".cursorrules": "Use 2 space indentation\n",
				"AGENTS.md":    "agents here\n",
			},
			wantBody: "Use 2 space indentation",
		},
		{
			name: "AGENTS.md used when CLAUDE.md and .cursorrules absent",
			files: map[string]string{
				"AGENTS.md": "# Agents\n",
			},
			wantBody: "# Agents",
		},
		{
			name: ".github/copilot-instructions.md used as fallback",
			files: map[string]string{
				".github/copilot-instructions.md": "Copilot rules\n",
			},
			wantBody: "Copilot rules",
		},
		{
			name:      "no AI context file returns empty string",
			files:     map[string]string{},
			wantEmpty: true,
		},
		{
			name: "empty CLAUDE.md falls through to .cursorrules",
			files: map[string]string{
				"CLAUDE.md":    "", // empty file — readFileQuiet returns ""
				".cursorrules": "cursor instructions\n",
			},
			wantBody: "cursor instructions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeTempProject(t, tt.files)
			got := extractExistingAI(dir)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("extractExistingAI() = %q, want empty string", got)
				}
				return
			}

			if got == "" {
				t.Error("extractExistingAI() returned empty string, want non-empty")
				return
			}

			if tt.wantBody != "" {
				found := false
				for i := 0; i+len(tt.wantBody) <= len(got); i++ {
					if got[i:i+len(tt.wantBody)] == tt.wantBody {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractExistingAI() = %q, want to contain %q", got, tt.wantBody)
				}
			}
		})
	}
}

// ── RunExtraction ─────────────────────────────────────────────────────────────

func TestRunExtraction_FullGoProject(t *testing.T) {
	dir := makeTempProject(t, map[string]string{
		"Makefile":     ".PHONY: dev test build\n\ndev:\n\tgo run ./cmd/...\n\ntest:\n\tgo test ./...\n\nbuild:\n\tgo build ./...\n",
		"CLAUDE.md":    "# My Project\n",
		".gitlab-ci.yml": "stages:\n  - test\n",
	})
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0755); err != nil {
		t.Fatal(err)
	}

	result := RunExtraction(dir)

	// Build commands
	if _, ok := result.BuildCommands["dev"]; !ok {
		t.Error("expected 'dev' in BuildCommands")
	}
	if _, ok := result.BuildCommands["test"]; !ok {
		t.Error("expected 'test' in BuildCommands")
	}
	if _, ok := result.BuildCommands["build"]; !ok {
		t.Error("expected 'build' in BuildCommands")
	}

	// CI/CD
	if !result.HasGitHub {
		t.Error("HasGitHub should be true")
	}
	if !result.HasGitLab {
		t.Error("HasGitLab should be true")
	}
	if result.HasCircleCI {
		t.Error("HasCircleCI should be false")
	}

	// Existing AI context
	if result.ExistingAI == "" {
		t.Error("ExistingAI should not be empty when CLAUDE.md exists")
	}
}

func TestRunExtraction_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	result := RunExtraction(dir)

	if len(result.BuildCommands) != 0 {
		t.Errorf("BuildCommands should be empty for empty project, got %v", result.BuildCommands)
	}
	if result.HasGitHub || result.HasGitLab || result.HasCircleCI {
		t.Error("no CI/CD should be detected for empty project")
	}
	if result.ExistingAI != "" {
		t.Errorf("ExistingAI should be empty for empty project, got %q", result.ExistingAI)
	}
}
