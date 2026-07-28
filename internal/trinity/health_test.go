package trinity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0merUfuk/the-matrix/internal/cli"
)

// createTestEcosystem builds a minimal .claude/ ecosystem in a temp directory.
func createTestEcosystem(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .claude/ directory
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Create CLAUDE.md
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# CLAUDE.md\n"), 0644)

	// Create .claude/knowledge/ with one subdir
	knowledgeDir := filepath.Join(claudeDir, "knowledge", "go-rules")
	os.MkdirAll(knowledgeDir, 0755)

	// Create a valid knowledge doc (>= 50 lines)
	validDoc := "**Version**: 1.0\n**Created**: 2026-03-01\n**Last Updated**: 2026-03-01\n**Authors:** Test\n\n---\n\n"
	validDoc += strings.Repeat("Content line for testing purposes.\n", 50)
	os.WriteFile(filepath.Join(knowledgeDir, "01-test-doc.md"), []byte(validDoc), 0644)

	// Create _index.md
	os.WriteFile(filepath.Join(knowledgeDir, "_index.md"), []byte("# Index\n"), 0644)

	// Create context files
	os.WriteFile(filepath.Join(claudeDir, "SERVICE_CONTEXT.md"), []byte("# Context\n"), 0644)
	os.WriteFile(filepath.Join(claudeDir, "NEXT_STEPS.md"), []byte("# Next\n"), 0644)

	return dir
}

func TestCheckKnowledgeDir_ValidDocs(t *testing.T) {
	dir := createTestEcosystem(t)
	knowledgeDir := filepath.Join(dir, ".claude", "knowledge", "go-rules")
	theme := cli.GetTheme()

	results := &Results{}
	checkKnowledgeDir(theme, knowledgeDir, "go-rules", 6, results)

	// Should have passes for: dir listing, _index.md, and the doc itself
	if results.Pass < 3 {
		t.Errorf("expected at least 3 passes, got %d (fail=%d, warn=%d)", results.Pass, results.Fail, results.Warn)
	}
	if results.Fail > 0 {
		t.Errorf("expected 0 failures, got %d", results.Fail)
	}
}

func TestCheckKnowledgeDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	emptyDir := filepath.Join(dir, "empty")
	os.MkdirAll(emptyDir, 0755)
	theme := cli.GetTheme()

	results := &Results{}
	checkKnowledgeDir(theme, emptyDir, "empty", 6, results)

	if results.Warn != 1 {
		t.Errorf("expected 1 warning for empty dir, got %d", results.Warn)
	}
}

func TestCheckKnowledgeDir_ShortDoc(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "short")
	os.MkdirAll(subdir, 0755)
	theme := cli.GetTheme()

	// Short doc (< 50 lines)
	os.WriteFile(filepath.Join(subdir, "01-short.md"), []byte("Too short\n"), 0644)

	results := &Results{}
	checkKnowledgeDir(theme, subdir, "short", 6, results)

	if results.Fail != 1 {
		t.Errorf("expected 1 failure for short doc, got %d", results.Fail)
	}
}

func TestCheckKnowledgeDir_NoDate(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "nodate")
	os.MkdirAll(subdir, 0755)
	theme := cli.GetTheme()

	// Valid length but no date
	content := strings.Repeat("No date frontmatter line.\n", 55)
	os.WriteFile(filepath.Join(subdir, "01-nodate.md"), []byte(content), 0644)

	results := &Results{}
	checkKnowledgeDir(theme, subdir, "nodate", 6, results)

	if results.Warn < 1 {
		t.Errorf("expected warning for missing date, got warn=%d, fail=%d, pass=%d", results.Warn, results.Fail, results.Pass)
	}
}

func TestCheckContextFiles_AllPresent(t *testing.T) {
	dir := createTestEcosystem(t)
	theme := cli.GetTheme()

	results := &Results{}
	checkContextFiles(theme, dir, results)

	if results.Pass != 2 {
		t.Errorf("expected 2 passes for context files, got %d", results.Pass)
	}
}

func TestCheckContextFiles_Missing(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude"), 0755)
	theme := cli.GetTheme()

	results := &Results{}
	checkContextFiles(theme, dir, results)

	if results.Warn != 2 {
		t.Errorf("expected 2 warnings for missing context files, got %d", results.Warn)
	}
}

func TestCheckTrinityLog_Missing(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude"), 0755)
	theme := cli.GetTheme()

	results := &Results{}
	checkTrinityLog(theme, dir, results)

	if results.Warn != 1 {
		t.Errorf("expected 1 warning for missing log, got %d", results.Warn)
	}
}

func TestExitCode(t *testing.T) {
	if exitCode(&Results{Pass: 5}) != 0 {
		t.Error("expected exit 0 with only passes")
	}
	if exitCode(&Results{Pass: 5, Warn: 2}) != 0 {
		t.Error("expected exit 0 with warnings only")
	}
	if exitCode(&Results{Pass: 5, Fail: 1}) != 1 {
		t.Error("expected exit 1 with any failure")
	}
}

// TestExpectedEcosystemMatchesRepo pins the agent and skill lists against the
// repo-root `.claude/` ecosystem when present. That ecosystem is private/local
// and absent from public clean checkouts, so this is an optional local drift
// check. If a skill or agent is added or removed without updating
// ExpectedAgents/ExpectedSkills, this test fails — preventing the silent drift
// that previously caused `trinity health --path .` to flag the-matrix's own
// repo as CRITICAL.
func TestExpectedEcosystemMatchesRepo(t *testing.T) {
	// Walk up from the test file location until we find the repo root
	// (identified by go.mod). The test must work both from the worktree
	// and from a regular checkout.
	root := findRepoRoot(t)

	agentsDir := filepath.Join(root, ".claude", "agents")
	skillsDir := filepath.Join(root, ".claude", "skills")
	_, agentsErr := os.Stat(agentsDir)
	_, skillsErr := os.Stat(skillsDir)
	if agentsErr != nil && !os.IsNotExist(agentsErr) {
		t.Fatalf("stat .claude/agents/: %v", agentsErr)
	}
	if skillsErr != nil && !os.IsNotExist(skillsErr) {
		t.Fatalf("stat .claude/skills/: %v", skillsErr)
	}
	if os.IsNotExist(agentsErr) || os.IsNotExist(skillsErr) {
		t.Skip("private .claude ecosystem is not tracked in public checkouts; skipping optional local drift check")
	}

	// Agents
	gotAgents, err := listEntries(agentsDir, false)
	if err != nil {
		t.Fatalf("read .claude/agents/: %v", err)
	}
	wantAgents := append([]string(nil), ExpectedAgents...)
	if !equalStringSets(gotAgents, wantAgents) {
		t.Errorf("ExpectedAgents drift:\n  on disk: %v\n  in code: %v\n  → update internal/trinity/health.go ExpectedAgents to match .claude/agents/", gotAgents, wantAgents)
	}

	// Skills (directories, not files)
	gotSkills, err := listEntries(skillsDir, true)
	if err != nil {
		t.Fatalf("read .claude/skills/: %v", err)
	}
	wantSkills := append([]string(nil), ExpectedSkills...)
	if !equalStringSets(gotSkills, wantSkills) {
		t.Errorf("ExpectedSkills drift:\n  on disk: %v\n  in code: %v\n  → update internal/trinity/health.go ExpectedSkills to match .claude/skills/", gotSkills, wantSkills)
	}
}

// findRepoRoot walks up from the cwd until it finds go.mod. Tests run with
// the package directory as cwd, so we ascend at most a few levels.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate go.mod walking up from cwd")
	return ""
}

// listEntries returns sorted entry names from dir. If dirsOnly is true, only
// directories are returned; otherwise only `.md` files.
func listEntries(dir string, dirsOnly bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if dirsOnly {
			if e.IsDir() {
				names = append(names, name)
			}
		} else {
			if !e.IsDir() && strings.HasSuffix(name, ".md") {
				names = append(names, name)
			}
		}
	}
	return names, nil
}

func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]struct{}, len(a))
	for _, s := range a {
		am[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := am[s]; !ok {
			return false
		}
	}
	return true
}
