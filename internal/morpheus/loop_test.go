package morpheus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildLoopManifest(t *testing.T) {
	ctx := NewLoopContext("test-project", "build a REST API", CycleConfig{
		MaxCycles: 12, DeveloperMaxTurns: 30, TesterMaxTurns: 20, ReviewerMaxTurns: 12,
	})

	manifest := BuildLoopManifest(ctx, "/tmp/test-project")

	if len(manifest) != 25 {
		t.Fatalf("expected 25 manifest entries, got %d", len(manifest))
	}

	// Verify expected output paths
	expectedOutputs := []string{
		".autonomous/loop.sh",
		".autonomous/config.sh",
		".autonomous/prompts/manager.md",
		".autonomous/prompts/strategist.md",
		".autonomous/prompts/developer.md",
		".autonomous/prompts/tester.md",
		".autonomous/prompts/reviewer.md",
		".autonomous/prompts/security.md",
		".claude/agents/developer.md",
		".claude/agents/tester.md",
		".claude/agents/reviewer.md",
		".claude/agents/strategist.md",
		".claude/agents/manager.md",
		".claude/skills/continue/SKILL.md",
		".claude/skills/commit/SKILL.md",
		".claude/agents/security-reviewer.md",
		".claude/skills/owasp-review/SKILL.md",
		".claude/skills/secret-scan/SKILL.md",
		".claude/skills/dep-audit/SKILL.md",
		".claude/skills/security-scan/SKILL.md",
		"tasks/current-state.md",
		"tasks/lessons.md",
		"tasks/decisions-pending.md",
		"tasks/TARGET_STATE.md",
		"tasks/DECISIONS.md",
	}
	for i, expected := range expectedOutputs {
		fullExpected := filepath.Join("/tmp/test-project", expected)
		if manifest[i].OutputPath != fullExpected {
			t.Errorf("manifest[%d]: expected output %q, got %q", i, fullExpected, manifest[i].OutputPath)
		}
	}

	// Verify loop.sh is static, rest are templated
	if !manifest[0].IsStatic {
		t.Error("loop.sh should be static")
	}
	for i := 1; i < len(manifest); i++ {
		if manifest[i].IsStatic {
			t.Errorf("manifest[%d] (%s) should not be static", i, manifest[i].TemplatePath)
		}
	}
}

func TestNewLoopContext(t *testing.T) {
	ctx := NewLoopContext("my-app", "add user auth", CycleConfig{
		MaxCycles: 10, DeveloperMaxTurns: 25, TesterMaxTurns: 15, ReviewerMaxTurns: 10,
	})

	if ctx.ProjectName != "my-app" {
		t.Errorf("expected ProjectName 'my-app', got %q", ctx.ProjectName)
	}
	if ctx.ProjectNamePascal != "MyApp" {
		t.Errorf("expected ProjectNamePascal 'MyApp', got %q", ctx.ProjectNamePascal)
	}
	if ctx.ServiceName != "my-app" {
		t.Errorf("expected ServiceName alias 'my-app', got %q", ctx.ServiceName)
	}
	if ctx.ServiceNamePascal != "MyApp" {
		t.Errorf("expected ServiceNamePascal alias 'MyApp', got %q", ctx.ServiceNamePascal)
	}
	if ctx.Goal != "add user auth" {
		t.Errorf("expected Goal 'add user auth', got %q", ctx.Goal)
	}
	if ctx.MaxCycles != 10 {
		t.Errorf("expected MaxCycles 10, got %d", ctx.MaxCycles)
	}
	if ctx.CreatedDate == "" {
		t.Error("expected CreatedDate to be set")
	}
}

func TestBuildEmptyTodo(t *testing.T) {
	todo := BuildEmptyTodo("MyApp")

	if !strings.HasPrefix(todo, "# Tasks: MyApp") {
		t.Errorf("expected todo to start with '# Tasks: MyApp', got %q", todo[:30])
	}
	if !strings.Contains(todo, "Phase 1") {
		t.Error("expected todo to contain Phase 1")
	}
	if !strings.Contains(todo, "Phase 2") {
		t.Error("expected todo to contain Phase 2")
	}
	if !strings.Contains(todo, "Phase 3") {
		t.Error("expected todo to contain Phase 3")
	}
}

func TestGenerateLoopFiles(t *testing.T) {
	// Create a temp directory for test output
	tmpDir, err := os.MkdirTemp("", "morp-loop-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := NewLoopContext("test-project", "", CycleConfig{
		MaxCycles: 12, DeveloperMaxTurns: 30, TesterMaxTurns: 20, ReviewerMaxTurns: 12,
	})

	written, err := GenerateLoopFiles(ctx, tmpDir)
	if err != nil {
		t.Fatalf("GenerateLoopFiles failed: %v", err)
	}

	if len(written) != 25 {
		t.Fatalf("expected 25 files written, got %d", len(written))
	}

	// Verify files exist on disk
	for _, path := range written {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", path)
		}
	}

	// Verify loop.sh is executable
	loopSh := filepath.Join(tmpDir, ".autonomous/loop.sh")
	info, err := os.Stat(loopSh)
	if err != nil {
		t.Fatalf("loop.sh stat failed: %v", err)
	}
	if info.Mode()&0100 == 0 {
		t.Error("loop.sh should be executable")
	}

	// Verify config.sh was rendered (not a raw template)
	configSh := filepath.Join(tmpDir, ".autonomous/config.sh")
	configContent, err := os.ReadFile(configSh)
	if err != nil {
		t.Fatalf("failed to read config.sh: %v", err)
	}
	if strings.Contains(string(configContent), "{{ .") {
		t.Error("config.sh contains unrendered template directives")
	}
	if !strings.Contains(string(configContent), "test-project") {
		t.Error("config.sh should contain project name")
	}
	if !strings.Contains(string(configContent), "MAX_CYCLES=12") {
		t.Error("config.sh should contain MAX_CYCLES=12")
	}

	// Verify prompts were rendered
	devPrompt := filepath.Join(tmpDir, ".autonomous/prompts/developer.md")
	devContent, err := os.ReadFile(devPrompt)
	if err != nil {
		t.Fatalf("failed to read developer.md: %v", err)
	}
	if !strings.Contains(string(devContent), "test-project") {
		t.Error("developer.md should contain project name")
	}
	if strings.Contains(string(devContent), "{{ .") {
		t.Error("developer.md contains unrendered template directives")
	}

	// Verify agent/skill files were rendered
	agentFiles := []string{
		".claude/agents/developer.md",
		".claude/agents/tester.md",
		".claude/agents/reviewer.md",
		".claude/agents/strategist.md",
		".claude/agents/manager.md",
		".claude/skills/continue/SKILL.md",
		".claude/skills/commit/SKILL.md",
	}
	for _, rel := range agentFiles {
		path := filepath.Join(tmpDir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", rel)
			continue
		}
		content, _ := os.ReadFile(path)
		if strings.Contains(string(content), "{{ .") {
			t.Errorf("%s contains unrendered template directives", rel)
		}
	}

	// Files that should contain the project name (agents + commit, but not continue which is generic)
	projectNameFiles := []string{
		".claude/agents/developer.md",
		".claude/agents/tester.md",
		".claude/agents/reviewer.md",
		".claude/agents/strategist.md",
		".claude/agents/manager.md",
		".claude/skills/commit/SKILL.md",
	}
	for _, rel := range projectNameFiles {
		content, _ := os.ReadFile(filepath.Join(tmpDir, rel))
		if !strings.Contains(string(content), "test-project") {
			t.Errorf("%s should contain project name", rel)
		}
	}

	// Verify shared task templates were rendered
	currentState := filepath.Join(tmpDir, "tasks/current-state.md")
	csContent, err := os.ReadFile(currentState)
	if err != nil {
		t.Fatalf("failed to read current-state.md: %v", err)
	}
	if !strings.Contains(string(csContent), "TestProject") {
		t.Error("current-state.md should contain PascalCase project name via ServiceNamePascal alias")
	}
}

func TestUpdateGitignore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "morp-gitignore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test: .gitignore doesn't exist yet
	updateGitignore(tmpDir)
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}
	if !strings.Contains(string(content), ".autonomous/") {
		t.Error("expected .gitignore to contain .autonomous/")
	}
	if !strings.Contains(string(content), "tasks/") {
		t.Error("expected .gitignore to contain tasks/")
	}

	// Test: .gitignore already has entries — should not duplicate
	updateGitignore(tmpDir)
	content2, _ := os.ReadFile(gitignorePath)
	count := strings.Count(string(content2), ".autonomous/")
	if count != 1 {
		t.Errorf("expected .autonomous/ to appear once, found %d times", count)
	}
}

func TestBuildLoopDecompositionPrompt(t *testing.T) {
	prompt := buildLoopDecompositionPrompt("# My Project\nGo service", "Add user authentication")

	if !strings.Contains(prompt, "# My Project") {
		t.Error("expected prompt to contain CLAUDE.md content")
	}
	if !strings.Contains(prompt, "Add user authentication") {
		t.Error("expected prompt to contain goal")
	}
	if !strings.Contains(prompt, "tasks/todo.md") {
		t.Error("expected prompt to reference tasks/todo.md")
	}
}

func TestBuildCycleConfig_AllDefaults(t *testing.T) {
	opts := LoopOpts{NonInteractive: true}
	cc := buildCycleConfig(opts)

	if cc.MaxCycles != 12 {
		t.Errorf("expected MaxCycles=12, got %d", cc.MaxCycles)
	}
	if cc.DeveloperMaxTurns != 30 {
		t.Errorf("expected DeveloperMaxTurns=30, got %d", cc.DeveloperMaxTurns)
	}
	if cc.TesterMaxTurns != 20 {
		t.Errorf("expected TesterMaxTurns=20, got %d", cc.TesterMaxTurns)
	}
	if cc.ReviewerMaxTurns != 12 {
		t.Errorf("expected ReviewerMaxTurns=12, got %d", cc.ReviewerMaxTurns)
	}
}

func TestBuildCycleConfig_CustomValues(t *testing.T) {
	opts := LoopOpts{
		NonInteractive:    true,
		MaxCycles:         5,
		DeveloperMaxTurns: 10,
		TesterMaxTurns:    8,
		ReviewerMaxTurns:  6,
	}
	cc := buildCycleConfig(opts)

	if cc.MaxCycles != 5 {
		t.Errorf("expected MaxCycles=5, got %d", cc.MaxCycles)
	}
	if cc.DeveloperMaxTurns != 10 {
		t.Errorf("expected DeveloperMaxTurns=10, got %d", cc.DeveloperMaxTurns)
	}
	if cc.TesterMaxTurns != 8 {
		t.Errorf("expected TesterMaxTurns=8, got %d", cc.TesterMaxTurns)
	}
	if cc.ReviewerMaxTurns != 6 {
		t.Errorf("expected ReviewerMaxTurns=6, got %d", cc.ReviewerMaxTurns)
	}
}

func TestBuildCycleConfig_PartialOverride(t *testing.T) {
	opts := LoopOpts{
		NonInteractive: true,
		MaxCycles:      5,
		// Leave other fields at zero → should fall back to defaults
	}
	cc := buildCycleConfig(opts)

	if cc.MaxCycles != 5 {
		t.Errorf("expected MaxCycles=5, got %d", cc.MaxCycles)
	}
	if cc.DeveloperMaxTurns != 30 {
		t.Errorf("expected default DeveloperMaxTurns=30, got %d", cc.DeveloperMaxTurns)
	}
	if cc.TesterMaxTurns != 20 {
		t.Errorf("expected default TesterMaxTurns=20, got %d", cc.TesterMaxTurns)
	}
	if cc.ReviewerMaxTurns != 12 {
		t.Errorf("expected default ReviewerMaxTurns=12, got %d", cc.ReviewerMaxTurns)
	}
}

func TestRunLoop_NonInteractive_NoAutonomousDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Run in dry-run + non-interactive mode — should not prompt
	RunLoop(LoopOpts{
		DryRun:         true,
		Path:           dir,
		NonInteractive: true,
		Goal:           "build something",
		MaxCycles:      3,
	})
	// If we reach here without panic or os.Exit, the test passes
}

func TestRunLoop_NonInteractive_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".autonomous"), 0755); err != nil {
		t.Fatal(err)
	}

	// Non-interactive + existing .autonomous/ → should NOT abort
	RunLoop(LoopOpts{
		DryRun:         true,
		Path:           dir,
		NonInteractive: true,
	})
	// Reaching here means no abort happened
}
