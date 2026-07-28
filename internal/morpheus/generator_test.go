package morpheus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testProjectContext() *ProjectContext {
	return &ProjectContext{
		ServiceName:       "test-service",
		ServiceNamePascal: "TestService",
		Description:       "A test microservice",
		ModulePath:        "github.com/0merUfuk/test-service",
		ProjectType:       "internal",
		Tech:              "go",
		IsGo:              true,
		IsNode:            false,
		IsInternal:          true,
		GoVersion:         "1.24",
		CreatedDate:       "2026-03-15",
		ApiPort:           8090,
		WorkerPort:        8091,
		HasWorker:         true,
		Entities:          []string{"User", "Order"},
		EntityCount:       2,
		CoreOperations:    "CRUD operations for users and orders",
		HasPostgres:       true,
		HasRedis:          true,
		HasRabbitMQ:       true,
		ExchangeName:      "test-service",
		Queues:            []string{"test-service.deliveries", "test-service.retry", "test-service.dlq"},
		HasJWT:            true,
		HasInternalKey:    false,
		AuthRequirements:  []string{"jwt"},
		HasRetry:          true,
		DeadLetterMechanism: "rabbitmq-dlq",
		RetryEntity:       "Order",
		RetryAttemptField: "attempt",
		MaxCycles:         12,
		DeveloperMaxTurns: 28,
		TesterMaxTurns:    20,
		ReviewerMaxTurns:  12,
	}
}

func TestBuildMorpheusManifest_GoWithWorker(t *testing.T) {
	ctx := testProjectContext()
	manifest := BuildMorpheusManifest(ctx, "/tmp/test-output")

	// Go+worker: 6 shared + 11 tech + 6 agents + 6 skills + 1 rule + 2 tasks + 8 go infra = 40
	if len(manifest) < 38 {
		t.Errorf("expected at least 38 manifest entries for Go+worker, got %d", len(manifest))
	}

	// Check static file
	hasStatic := false
	for _, m := range manifest {
		if m.IsStatic && strings.Contains(m.TemplatePath, "loop.sh") {
			hasStatic = true
		}
	}
	if !hasStatic {
		t.Error("expected static loop.sh in manifest")
	}

	// Check worker-specific files
	hasWorkerDockerfile := false
	hasWorkerConfig := false
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "Dockerfile.worker") {
			hasWorkerDockerfile = true
		}
		if strings.Contains(m.OutputPath, "worker-local.yaml") {
			hasWorkerConfig = true
		}
	}
	if !hasWorkerDockerfile {
		t.Error("expected Dockerfile.worker in manifest (hasWorker=true)")
	}
	if !hasWorkerConfig {
		t.Error("expected worker-local.yaml in manifest (hasWorker=true)")
	}
}

func TestBuildMorpheusManifest_GoWithoutWorker(t *testing.T) {
	ctx := testProjectContext()
	ctx.HasWorker = false
	manifest := BuildMorpheusManifest(ctx, "/tmp/test-output")

	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "Dockerfile.worker") {
			t.Error("should NOT have Dockerfile.worker when hasWorker=false")
		}
		if strings.Contains(m.OutputPath, "worker-local.yaml") {
			t.Error("should NOT have worker-local.yaml when hasWorker=false")
		}
	}
}

func TestBuildMorpheusManifest_Node(t *testing.T) {
	ctx := testProjectContext()
	ctx.Tech = "node"
	ctx.IsGo = false
	ctx.IsNode = true
	manifest := BuildMorpheusManifest(ctx, "/tmp/test-output")

	// Node.js should NOT have Go-specific files
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "Makefile") {
			t.Error("should NOT have Makefile for Node.js")
		}
		if strings.Contains(m.OutputPath, "go.mod") {
			t.Error("should NOT have go.mod for Node.js")
		}
		if strings.Contains(m.OutputPath, "Dockerfile") {
			t.Error("should NOT have Dockerfile for Node.js")
		}
	}

	// Should have node-service rules
	hasNodeRules := false
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, "node-service.md") {
			hasNodeRules = true
		}
	}
	if !hasNodeRules {
		t.Error("expected node-service.md rules for Node.js project")
	}

	// Should have node agents (5 total: developer, tester, reviewer, strategist, manager)
	hasDevAgent := false
	hasTesterAgent := false
	hasReviewerAgent := false
	hasStrategistAgent := false
	hasManagerAgent := false
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, ".claude/agents/developer.md") {
			hasDevAgent = true
		}
		if strings.Contains(m.OutputPath, ".claude/agents/tester.md") {
			hasTesterAgent = true
		}
		if strings.Contains(m.OutputPath, ".claude/agents/reviewer.md") {
			hasReviewerAgent = true
		}
		if strings.Contains(m.OutputPath, ".claude/agents/strategist.md") {
			hasStrategistAgent = true
		}
		if strings.Contains(m.OutputPath, ".claude/agents/manager.md") {
			hasManagerAgent = true
		}
	}
	if !hasDevAgent {
		t.Error("expected developer agent for Node.js project")
	}
	if !hasTesterAgent {
		t.Error("expected tester agent for Node.js project")
	}
	if !hasReviewerAgent {
		t.Error("expected reviewer agent for Node.js project")
	}
	if !hasStrategistAgent {
		t.Error("expected strategist agent for Node.js project")
	}
	if !hasManagerAgent {
		t.Error("expected manager agent for Node.js project")
	}
}

func TestBuildMorpheusManifest_HasAgentsAndSkills(t *testing.T) {
	ctx := testProjectContext()
	manifest := BuildMorpheusManifest(ctx, "/tmp/test-output")

	agentCount := 0
	skillCount := 0
	for _, m := range manifest {
		if strings.Contains(m.OutputPath, ".claude/agents/") {
			agentCount++
		}
		if strings.Contains(m.OutputPath, ".claude/skills/") {
			skillCount++
		}
	}
	if agentCount != 6 {
		t.Errorf("expected 6 agent files, got %d", agentCount)
	}
	if skillCount != 6 {
		t.Errorf("expected 6 skill files, got %d", skillCount)
	}
}

func TestGenerateMorpheusFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := testProjectContext()

	written, err := GenerateMorpheusFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}

	if len(written) < 32 {
		t.Errorf("expected at least 32 files written, got %d", len(written))
	}

	// Check key files exist
	mustExist := []string{
		".autonomous/loop.sh",
		".autonomous/config.sh",
		".autonomous/finalizer.sh",
		".autonomous/prompts/developer.md",
		".autonomous/prompts/tester.md",
		".autonomous/prompts/reviewer.md",
		".autonomous/prompts/finalizer.md",
		"CLAUDE.md",
		".claude/settings.json",
		".claude/SERVICE_CONTEXT.md",
		".claude/NEXT_STEPS.md",
		".claude/DECISIONS.md",
		".claude/KNOWN_ISSUES.md",
		".claude/agents/developer.md",
		".claude/agents/tester.md",
		".claude/agents/reviewer.md",
		".claude/agents/strategist.md",
		".claude/agents/manager.md",
		".claude/agents/security-reviewer.md",
		".claude/skills/continue/SKILL.md",
		".claude/skills/commit/SKILL.md",
		".claude/skills/owasp-review/SKILL.md",
		".claude/skills/secret-scan/SKILL.md",
		".claude/skills/dep-audit/SKILL.md",
		".claude/skills/security-scan/SKILL.md",
		".claude/rules/go-service.md",
		"tasks/DECISIONS.md",
		"tasks/TARGET_STATE.md",
		"Makefile",
		"go.mod",
		"docker-compose.yml",
		".gitignore",
	}
	for _, rel := range mustExist {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist", rel)
		}
	}

	// Check no raw EJS tags in any file
	for _, rel := range mustExist {
		content, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			continue
		}
		if strings.Contains(string(content), "<%") {
			t.Errorf("raw EJS tag found in %s", rel)
		}
	}

	// Check no raw Go template tags in agent/skill files
	agentFiles := []string{
		".claude/agents/developer.md",
		".claude/agents/tester.md",
		".claude/agents/reviewer.md",
		".claude/agents/strategist.md",
		".claude/agents/manager.md",
		".claude/agents/security-reviewer.md",
		".claude/skills/continue/SKILL.md",
		".claude/skills/commit/SKILL.md",
		".claude/skills/owasp-review/SKILL.md",
		".claude/skills/secret-scan/SKILL.md",
		".claude/skills/dep-audit/SKILL.md",
		".claude/skills/security-scan/SKILL.md",
	}
	for _, rel := range agentFiles {
		content, _ := os.ReadFile(filepath.Join(dir, rel))
		if strings.Contains(string(content), "{{") {
			t.Errorf("raw template tag found in %s", rel)
		}
	}

	// Check loop.sh is executable
	info, _ := os.Stat(filepath.Join(dir, ".autonomous/loop.sh"))
	if info.Mode()&0111 == 0 {
		t.Error("loop.sh should be executable")
	}

	// Check config.sh has service name
	configContent, _ := os.ReadFile(filepath.Join(dir, ".autonomous/config.sh"))
	if !strings.Contains(string(configContent), "test-service") {
		t.Error("config.sh should contain service name")
	}

	// Check CLAUDE.md has service name and agent references
	claudeContent, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	claudeStr := string(claudeContent)
	if !strings.Contains(claudeStr, "test-service") {
		t.Error("CLAUDE.md should contain service name")
	}
	if !strings.Contains(claudeStr, "## Agents") {
		t.Error("CLAUDE.md should contain Agents section")
	}
	if !strings.Contains(claudeStr, "## Skills") {
		t.Error("CLAUDE.md should contain Skills section")
	}

	// Check developer agent has service name
	devContent, _ := os.ReadFile(filepath.Join(dir, ".claude/agents/developer.md"))
	if !strings.Contains(string(devContent), "test-service") {
		t.Error("developer agent should contain service name")
	}
}

func TestBuildSkeletonTodo(t *testing.T) {
	ctx := testProjectContext()
	todo := BuildSkeletonTodo(ctx)

	if !strings.Contains(todo, "# Tasks: TestService") {
		t.Error("expected header with PascalCase service name")
	}
	if !strings.Contains(todo, "skeleton template") {
		t.Error("expected skeleton template note")
	}
	if !strings.Contains(todo, "Phase 1") {
		t.Error("expected Phase 1")
	}
	if !strings.Contains(todo, "User") {
		t.Error("expected User entity tasks")
	}
	if !strings.Contains(todo, "Order") {
		t.Error("expected Order entity tasks")
	}
	if !strings.Contains(todo, "go mod init") {
		t.Error("expected go mod init for Go project")
	}
}

func TestBuildProjectContext(t *testing.T) {
	ctx := testProjectContext()
	doc := BuildProjectContext(ctx)

	if !strings.Contains(doc, "SERVICE: test-service") {
		t.Error("expected SERVICE field")
	}
	if !strings.Contains(doc, "TECH: GO") {
		t.Error("expected TECH: GO")
	}
	if !strings.Contains(doc, "API_PORT: 8090") {
		t.Error("expected API_PORT")
	}
	if !strings.Contains(doc, "WORKER_PORT: 8091") {
		t.Error("expected WORKER_PORT")
	}
	if !strings.Contains(doc, "RETRY: exponential-backoff") {
		t.Error("expected RETRY pattern")
	}
}
