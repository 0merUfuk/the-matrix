package oracle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testStackContext() *StackContext {
	return &StackContext{
		StackName:         "test-stack",
		FrameworkVersion:  "Test Framework 1.0",
		AppType:           "api-only",
		ArchStyle:         "layered",
		StateApproach:     "n/a",
		DataLayer:         "none",
		TestingFramework:  "Jest",
		HasQueue:          false,
		HasBackgroundJobs: false,
		SelectedTopics:    []string{"00-core-principles", "01-project-structure", "05-testing"},
		TopicCount:        3,
		OutputPath:        "/tmp/test",
		CreatedDate:       "2026-03-15",
		GoldStandards: map[string]string{
			"layered-architecture": "[test gold standard]",
			"error-handling":       "[test gold standard]",
			"testing":              "[test gold standard]",
		},
	}
}

func TestBuildFileManifest(t *testing.T) {
	ctx := testStackContext()
	manifest := BuildFileManifest(ctx, "/tmp/output")

	if len(manifest) != 6 {
		t.Fatalf("expected 6 manifest entries, got %d", len(manifest))
	}

	// First entry should be static loop.sh
	if !manifest[0].IsStatic {
		t.Error("loop.sh should be static")
	}
	if manifest[0].TemplatePath != "loop.sh" {
		t.Errorf("expected loop.sh, got %s", manifest[0].TemplatePath)
	}

	// Count static vs template
	staticCount := 0
	templateCount := 0
	for _, m := range manifest {
		if m.IsStatic {
			staticCount++
		} else {
			templateCount++
		}
	}
	if staticCount != 1 {
		t.Errorf("expected 1 static file, got %d", staticCount)
	}
	if templateCount != 5 {
		t.Errorf("expected 5 template files, got %d", templateCount)
	}
}

func TestBuildDryRunManifest(t *testing.T) {
	ctx := testStackContext()
	entries := BuildDryRunManifest(ctx, "/tmp/output")

	if len(entries) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.TemplateName == "" {
			t.Error("template name should not be empty")
		}
	}
}

func TestGenerateFiles_FullGeneration(t *testing.T) {
	dir := t.TempDir()
	ctx := testStackContext()

	written, err := GenerateFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateFiles failed: %v", err)
	}

	if len(written) != 6 {
		t.Errorf("expected 6 written files, got %d", len(written))
	}

	// Check key files exist
	mustExist := []string{
		".autonomous/loop.sh",
		".autonomous/config.sh",
		"CLAUDE.md",
		".autonomous/prompts/researcher.md",
		".autonomous/prompts/reviewer.md",
		".autonomous/prompts/synthesizer.md",
	}
	for _, rel := range mustExist {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist", rel)
		}
	}

	// Check seed directories
	seedDirs := []string{"tasks/research", "tasks/review", "tasks/cycles", "output/test-stack"}
	for _, rel := range seedDirs {
		p := filepath.Join(dir, rel)
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			t.Errorf("expected directory %s to exist", rel)
		}
	}

	// Check status.txt
	statusContent, err := os.ReadFile(filepath.Join(dir, "tasks/status.txt"))
	if err != nil {
		t.Fatal("expected tasks/status.txt")
	}
	if strings.TrimSpace(string(statusContent)) != "INITIALIZED" {
		t.Errorf("expected INITIALIZED, got %q", string(statusContent))
	}

	// Check .gitignore
	gitignoreContent, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal("expected .gitignore")
	}
	if !strings.Contains(string(gitignoreContent), "tasks/") {
		t.Error("expected tasks/ in .gitignore")
	}

	// Check config.sh has stack name substituted
	configContent, err := os.ReadFile(filepath.Join(dir, ".autonomous/config.sh"))
	if err != nil {
		t.Fatal("expected config.sh")
	}
	if !strings.Contains(string(configContent), `STACK_NAME="test-stack"`) {
		t.Errorf("expected stack name in config.sh, got: %s", string(configContent)[:200])
	}

	// Check no raw EJS tags in any generated file
	for _, rel := range mustExist {
		content, _ := os.ReadFile(filepath.Join(dir, rel))
		if strings.Contains(string(content), "<%") {
			t.Errorf("raw EJS tag found in %s", rel)
		}
	}

	// Check loop.sh is executable
	info, _ := os.Stat(filepath.Join(dir, ".autonomous/loop.sh"))
	if info.Mode()&0111 == 0 {
		t.Error("loop.sh should be executable")
	}

	// Check config.sh is executable
	info, _ = os.Stat(filepath.Join(dir, ".autonomous/config.sh"))
	if info.Mode()&0111 == 0 {
		t.Error("config.sh should be executable")
	}
}

func TestGenerateFiles_SynthesizerHasGoldStandards(t *testing.T) {
	dir := t.TempDir()
	ctx := testStackContext()

	_, err := GenerateFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateFiles failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".autonomous/prompts/synthesizer.md"))
	s := string(content)

	// Gold standards should be injected — dynamic iteration over map entries
	if !strings.Contains(s, "GOLD_STANDARD_EXAMPLE") {
		t.Error("expected GOLD_STANDARD_EXAMPLE tags in synthesizer.md")
	}
	// Check all 3 pre-set gold standard topics are rendered
	for _, topic := range []string{"layered-architecture", "error-handling", "testing"} {
		if !strings.Contains(s, fmt.Sprintf(`topic="%s"`, topic)) {
			t.Errorf("expected gold standard section for %q in synthesizer.md", topic)
		}
	}
}

func TestGenerateFiles_SynthesizerNoGoldStandards(t *testing.T) {
	dir := t.TempDir()
	ctx := testStackContext()
	// Clear gold standards to test fallback
	ctx.GoldStandards = map[string]string{}

	_, err := GenerateFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateFiles failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".autonomous/prompts/synthesizer.md"))
	s := string(content)

	// With no gold standards, should show the fallback message
	if !strings.Contains(s, "No gold standard examples provided") {
		t.Error("expected fallback message when no gold standards are set")
	}
}

func TestGenerateFiles_ConfigHasGoldStandardsDir(t *testing.T) {
	dir := t.TempDir()
	ctx := testStackContext()
	ctx.GoldStandardsDir = "/home/user/.the-matrix/gold-standards/go"

	_, err := GenerateFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateFiles failed: %v", err)
	}

	configContent, _ := os.ReadFile(filepath.Join(dir, ".autonomous/config.sh"))
	s := string(configContent)

	if !strings.Contains(s, "GOLD_STANDARDS_DIR=") {
		t.Error("expected GOLD_STANDARDS_DIR in config.sh")
	}
	if !strings.Contains(s, "/home/user/.the-matrix/gold-standards/go") {
		t.Errorf("expected gold standards dir path in config.sh, got: %s", s)
	}
}

func TestGenerateFiles_TopicListInSynthesizer(t *testing.T) {
	dir := t.TempDir()
	ctx := testStackContext()

	_, err := GenerateFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateFiles failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".autonomous/prompts/synthesizer.md"))
	s := string(content)

	for _, topic := range ctx.SelectedTopics {
		expected := fmt.Sprintf("output/%s/%s.md", ctx.StackName, topic)
		if !strings.Contains(s, expected) {
			t.Errorf("expected topic %q in synthesizer.md", expected)
		}
	}
}
