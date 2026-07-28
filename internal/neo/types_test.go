package neo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testProfile() *ProjectProfile {
	return &ProjectProfile{
		ProjectName: "test-project",
		ProjectType: "single-app",
		TeamSize:    "solo",
		Stacks: []StackProfile{
			{
				Name:             "go-api",
				Language:         "go",
				Framework:        "chi v5",
				ArchStyle:        "layered",
				TestingFramework: "Ginkgo",
				HasQueue:         false,
				DataLayer:        "Bun ORM",
			},
		},
		IsInternal:    false,
		CreatedDate: "2026-03-15",
		OutputPath:  "/tmp/neo-test",
	}
}

func TestBuildNeoManifest(t *testing.T) {
	p := testProfile()
	manifest := BuildNeoManifest(p, "/tmp/neo-test")

	// Any team: CLAUDE.md(1) + 4 context + 2 rules(session+1stack) + 5 agents(always manager) + 6 skills = 18
	if len(manifest) < 18 {
		t.Errorf("expected at least 18 manifest entries, got %d", len(manifest))
	}

	// Check CLAUDE.md is in manifest
	hasClaude := false
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "CLAUDE.md") {
			hasClaude = true
		}
	}
	if !hasClaude {
		t.Error("expected CLAUDE.md in manifest")
	}

	// Check rules per stack
	hasStackRules := false
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "go-api-services.md") {
			hasStackRules = true
		}
	}
	if !hasStackRules {
		t.Error("expected go-api-services.md rules file")
	}
}

func TestBuildNeoManifest_AlwaysHasManager(t *testing.T) {
	for _, size := range []string{"solo", "small", "mid"} {
		t.Run(size, func(t *testing.T) {
			p := testProfile()
			p.TeamSize = size
			manifest := BuildNeoManifest(p, "/tmp/neo-test")

			hasManager := false
			for _, m := range manifest {
				if strings.Contains(m.OutputPath, "agents/manager.md") {
					hasManager = true
				}
			}
			if !hasManager {
				t.Errorf("%s team should have manager agent", size)
			}
		})
	}
}

func TestBuildNeoManifest_HasStrategistNotArchitect(t *testing.T) {
	p := testProfile()
	manifest := BuildNeoManifest(p, "/tmp/neo-test")

	hasStrategist := false
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "agents/strategist.md") {
			hasStrategist = true
		}
		if strings.Contains(m.OutputPath, "agents/architect.md") {
			t.Error("architect agent should not be in manifest (replaced by strategist)")
		}
	}
	if !hasStrategist {
		t.Error("expected strategist agent in manifest")
	}
}

func TestBuildNeoManifest_MultiStack(t *testing.T) {
	p := testProfile()
	p.Stacks = append(p.Stacks, StackProfile{
		Name:     "node-frontend",
		Language: "nodejs",
	})
	manifest := BuildNeoManifest(p, "/tmp/neo-test")

	// Should have 2 stack-specific rule files
	ruleCount := 0
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "-services.md") {
			ruleCount++
		}
	}
	if ruleCount != 2 {
		t.Errorf("expected 2 stack rule files for 2 stacks, got %d", ruleCount)
	}
}

func TestBuildNeoManifest_SkillsAreDynamic(t *testing.T) {
	p := testProfile()
	manifest := BuildNeoManifest(p, "/tmp/neo-test")

	for _, m := range manifest {
		if strings.Contains(m.OutputPath, ".claude/skills/") && m.IsStatic {
			t.Errorf("skill %s should be dynamic (IsStatic=false), got IsStatic=true", m.TemplatePath)
		}
	}
}

func TestBuildNeoManifest_Has10Skills(t *testing.T) {
	p := testProfile()
	manifest := BuildNeoManifest(p, "/tmp/neo-test")

	skillCount := 0
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, ".claude/skills/") {
			skillCount++
		}
	}
	if skillCount != 10 {
		t.Errorf("expected 10 skill files, got %d", skillCount)
	}
}

func TestBuildNeoManifest_NoCommands(t *testing.T) {
	p := testProfile()
	manifest := BuildNeoManifest(p, "/tmp/neo-test")

	for _, m := range manifest {
		if strings.Contains(m.OutputPath, ".claude/commands/") {
			t.Errorf("commands/ should not be in manifest (migrated to skills/), found: %s", m.OutputPath)
		}
	}
}

func TestGenerateEcosystem(t *testing.T) {
	dir := t.TempDir()
	p := testProfile()

	written, err := GenerateEcosystem(p, dir)
	if err != nil {
		t.Fatalf("GenerateEcosystem failed: %v", err)
	}

	// Any team: 18 manifest entries + 2 json files = 20
	if len(written) < 20 {
		t.Errorf("expected at least 20 files, got %d", len(written))
	}

	// Check key files exist
	mustExist := []string{
		"CLAUDE.md",
		".claude/SERVICE_CONTEXT.md",
		".claude/NEXT_STEPS.md",
		".claude/DECISIONS.md",
		".claude/KNOWN_ISSUES.md",
		".claude/rules/session-protocol.md",
		".claude/agents/strategist.md",
		".claude/agents/developer.md",
		".claude/agents/manager.md",
		".claude/agents/reviewer.md",
		".claude/agents/tester.md",
		".claude/agents/security-reviewer.md",
		".claude/skills/audit/SKILL.md",
		".claude/skills/commit/SKILL.md",
		".claude/skills/continue/SKILL.md",
		".claude/skills/doublecheck/SKILL.md",
		".claude/skills/fix/SKILL.md",
		".claude/skills/issue/SKILL.md",
		".claude/skills/owasp-review/SKILL.md",
		".claude/skills/secret-scan/SKILL.md",
		".claude/skills/dep-audit/SKILL.md",
		".claude/skills/security-scan/SKILL.md",
	}
	for _, rel := range mustExist {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist", rel)
		}
	}

	// Architect agent should NOT exist (replaced by strategist)
	architectPath := filepath.Join(dir, ".claude/agents/architect.md")
	if _, err := os.Stat(architectPath); err == nil {
		t.Error("architect.md should NOT exist (replaced by strategist)")
	}

	// Check CLAUDE.md has project name
	content, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if !strings.Contains(string(content), "test-project") {
		t.Error("CLAUDE.md should contain project name")
	}

	// Check session protocol has solo content
	spContent, _ := os.ReadFile(filepath.Join(dir, ".claude/rules/session-protocol.md"))
	if !strings.Contains(string(spContent), "Solo") {
		t.Error("session-protocol should contain Solo content for solo team")
	}

	// Check no raw template tags in rendered files.
	// Exclude doublecheck SKILL.md which intentionally contains template syntax
	// as documentation (e.g., "Are {{ .Field }} references guarded?").
	for _, rel := range mustExist {
		if strings.Contains(rel, "doublecheck") {
			continue
		}
		content, _ := os.ReadFile(filepath.Join(dir, rel))
		if strings.Contains(string(content), "{{") {
			t.Errorf("raw template tag found in %s", rel)
		}
	}
}

func TestGenerateEcosystem_AllTeamSizesHaveManager(t *testing.T) {
	for _, size := range []string{"solo", "small", "mid"} {
		t.Run(size, func(t *testing.T) {
			dir := t.TempDir()
			p := testProfile()
			p.TeamSize = size

			_, err := GenerateEcosystem(p, dir)
			if err != nil {
				t.Fatalf("GenerateEcosystem failed for %s team: %v", size, err)
			}

			managerPath := filepath.Join(dir, ".claude/agents/manager.md")
			if _, err := os.Stat(managerPath); err != nil {
				t.Errorf("%s team should have manager.md agent", size)
			}

			// Check manager has no raw template tags
			managerContent, _ := os.ReadFile(managerPath)
			if strings.Contains(string(managerContent), "{{") {
				t.Errorf("raw template tag found in manager.md for %s team", size)
			}
		})
	}
}

func TestGenerateEcosystem_FlutterDartLanguage(t *testing.T) {
	dir := t.TempDir()
	p := &ProjectProfile{
		ProjectName: "flutter-app",
		ProjectType: "single-app",
		TeamSize:    "solo",
		Stacks: []StackProfile{
			{
				Name:             "mobile",
				Language:         "dart",
				Framework:        "Flutter 3",
				ArchStyle:        "feature-based",
				TestingFramework: "flutter_test",
				HasQueue:         false,
				DataLayer:        "none",
			},
		},
		IsInternal:    false,
		CreatedDate: "2026-03-18",
		OutputPath:  dir,
	}

	_, err := GenerateEcosystem(p, dir)
	if err != nil {
		t.Fatalf("GenerateEcosystem failed for Flutter/Dart: %v", err)
	}

	// Developer agent should contain Flutter-specific content
	devContent, err := os.ReadFile(filepath.Join(dir, ".claude/agents/developer.md"))
	if err != nil {
		t.Fatalf("reading developer.md: %v", err)
	}
	if !strings.Contains(string(devContent), "flutter") && !strings.Contains(string(devContent), "Flutter") {
		t.Error("developer.md should contain Flutter-specific content for dart language")
	}

	// Tester agent should contain Flutter-specific content
	testerContent, err := os.ReadFile(filepath.Join(dir, ".claude/agents/tester.md"))
	if err != nil {
		t.Fatalf("reading tester.md: %v", err)
	}
	if !strings.Contains(string(testerContent), "flutter") && !strings.Contains(string(testerContent), "Flutter") {
		t.Error("tester.md should contain Flutter-specific content for dart language")
	}

	// Reviewer agent should contain Flutter-specific content
	reviewerContent, err := os.ReadFile(filepath.Join(dir, ".claude/agents/reviewer.md"))
	if err != nil {
		t.Fatalf("reading reviewer.md: %v", err)
	}
	if !strings.Contains(string(reviewerContent), "flutter") && !strings.Contains(string(reviewerContent), "Flutter") {
		t.Error("reviewer.md should contain Flutter/Dart-specific content for dart language")
	}

	// Go-specific content should NOT be present
	if strings.Contains(string(devContent), "go vet") {
		t.Error("developer.md should NOT contain Go-specific content for dart language")
	}
}

func TestLanguageExtension_Dart(t *testing.T) {
	if ext := languageExtension("dart"); ext != "dart" {
		t.Errorf("expected languageExtension(\"dart\") = \"dart\", got %q", ext)
	}
	if ext := languageExtension("flutter"); ext != "dart" {
		t.Errorf("expected languageExtension(\"flutter\") = \"dart\", got %q", ext)
	}
}
