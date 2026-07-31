package morpheus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nonInternalContext returns a ProjectContext with IsInternal=false for generalization tests.
func nonInternalContext() *ProjectContext {
	ctx := testProjectContext()
	ctx.IsInternal = false
	ctx.ProjectType = "general"
	return ctx
}

// renderAndRead generates all morp files to a temp dir and returns the content of the
// given relative path. It fails the test immediately if generation or reading fails.
func renderAndRead(t *testing.T, ctx *ProjectContext, rel string) string {
	t.Helper()
	dir := t.TempDir()
	_, err := GenerateMorpheusFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("could not read %s: %v", rel, err)
	}
	return string(content)
}

// TestGenerateMorpheusFiles_IsInternalTrue_DeveloperPrompt verifies that the developer prompt
// rendered for a originating project project contains all originating project-specific stack references.
func TestGenerateMorpheusFiles_IsInternalTrue_DeveloperPrompt(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".autonomous/prompts/developer.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"zerolog", "zerolog logging library"},
		{"chi/v5", "chi/v5 router"},
		{"bun:", "Bun ORM struct tag"},
		{"Google Wire", "Google Wire DI"},
		{"Go Patterns (from originating project knowledge)", "originating project knowledge heading"},
		{"Dockerfile.api", "NEVER recreate Dockerfile.api rule"},
		{"Bun ORM", "Bun ORM in architecture section"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true developer prompt missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_DeveloperPrompt verifies that the developer prompt
// rendered for a non-originating project project strips all originating project-specific content and includes generic
// fallback content.
func TestGenerateMorpheusFiles_IsInternalFalse_DeveloperPrompt(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".autonomous/prompts/developer.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"zerolog", "zerolog (originating project-specific logging)"},
		{"chi/v5", "chi/v5 (originating project-specific router)"},
		{"Google Wire", "Google Wire (originating project-specific DI)"},
		{"Go Patterns (from originating project knowledge)", "originating project knowledge heading"},
		{"Dockerfile.api", "NEVER recreate Dockerfile.api (originating project-only rule)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false developer prompt should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	mustContain := []struct {
		substr string
		label  string
	}{
		{"Go Patterns (project-specific)", "project-specific patterns heading"},
		{"fmt.Errorf", "generic error wrapping rule"},
		{"standard logger", "generic logging rule"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false developer prompt missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_ReviewerPrompt verifies that the reviewer prompt
// rendered for a originating project project contains originating project-specific review checklist items.
func TestGenerateMorpheusFiles_IsInternalTrue_ReviewerPrompt(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".autonomous/prompts/reviewer.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"originating project-specific", "originating project-specific section heading"},
		{"zerolog", "zerolog check in section 2"},
		{"Ginkgo", "Ginkgo test framework check"},
		{"Gomock", "Gomock interface mocking check"},
		{"Bun ORM", "Bun ORM in architecture section"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true reviewer prompt missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_ReviewerPrompt verifies that the reviewer prompt
// rendered for a non-originating project project strips originating project-specific checklist items and provides
// generic fallback content.
func TestGenerateMorpheusFiles_IsInternalFalse_ReviewerPrompt(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".autonomous/prompts/reviewer.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"originating project-specific", "originating project-specific heading"},
		{"zerolog", "zerolog (originating project logging)"},
		{"Ginkgo", "Ginkgo (originating project test framework)"},
		{"Gomock", "Gomock (originating project mocking library)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false reviewer prompt should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	mustContain := []struct {
		substr string
		label  string
	}{
		{"project-specific", "project-specific patterns heading"},
		{"Test framework matches project convention", "generic test framework guidance"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false reviewer prompt missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_TesterPrompt verifies that the tester prompt
// rendered for a originating project project contains Ginkgo/Gomock-specific content.
func TestGenerateMorpheusFiles_IsInternalTrue_TesterPrompt(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".autonomous/prompts/tester.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"Ginkgo", "Ginkgo test framework"},
		{"gomock", "gomock mocking library"},
		{"Ginkgo/Gomega", "Ginkgo/Gomega in title"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true tester prompt missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_TesterPrompt verifies that the tester prompt
// rendered for a non-originating project project strips Ginkgo/Gomock content and includes generic fallbacks.
func TestGenerateMorpheusFiles_IsInternalFalse_TesterPrompt(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".autonomous/prompts/tester.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"Ginkgo", "Ginkgo (originating project test framework)"},
		{"Gomock", "Gomock (originating project mocking library, capitalized)"},
		{"gomock", "gomock (originating project mocking library, lowercase)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false tester prompt should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Verify generic fallback content is present.
	hasTestSuite := strings.Contains(content, "test suite")
	hasTestConventions := strings.Contains(content, "testing conventions")
	if !hasTestSuite && !hasTestConventions {
		t.Error("IsInternal=false tester prompt should contain generic guidance ('test suite' or 'testing conventions')")
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_GoServiceRules verifies that the go-service rules file
// rendered for a originating project project contains originating project-specific conventions.
func TestGenerateMorpheusFiles_IsInternalTrue_GoServiceRules(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/rules/go-service.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"zerolog", "zerolog logging standard"},
		{"4-layer strict", "4-layer strict architecture"},
		{"UUID primary keys", "UUID primary keys rule"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true go-service rules missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoServiceRules verifies that the go-service rules file
// rendered for a non-originating project project strips originating project-specific standards and includes generic fallbacks.
func TestGenerateMorpheusFiles_IsInternalFalse_GoServiceRules(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/rules/go-service.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"zerolog", "zerolog (originating project-specific)"},
		{"UUID primary keys", "UUID primary keys (originating project-specific)"},
		{"chi", "chi router (originating project-specific)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false go-service rules should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	mustContain := []struct {
		substr string
		label  string
	}{
		{"CLAUDE.md", "CLAUDE.md reference for generic rules"},
		{"fmt.Errorf", "general Go error wrapping rule"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false go-service rules missing %s: expected to find %q", tc.label, tc.substr)
		}
	}

	// The template tags must be fully rendered — no raw "{{ " should appear in output.
	if strings.Contains(content, "{{ ") {
		t.Error("IsInternal=false go-service rules contains unrendered template tag {{ ")
	}

	// The module path variable must have been substituted with the actual value.
	if !strings.Contains(content, "github.com/0merUfuk/test-service") {
		t.Errorf("go-service rules missing rendered module path: expected github.com/0merUfuk/test-service")
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_AgentReviewer verifies that the .claude/agents/reviewer.md
// rendered for a originating project project contains all originating project-specific review checklist items.
func TestGenerateMorpheusFiles_IsInternalTrue_AgentReviewer(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/agents/reviewer.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"zerolog", "zerolog logging check"},
		{"Bun struct", "Bun struct tags check"},
		{"Viper", "Viper config check"},
		{"Wire DI", "Wire DI check"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true agent reviewer missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_AgentReviewer verifies that the .claude/agents/reviewer.md
// rendered for a non-originating project project strips originating project-specific tools and uses generic alternatives.
func TestGenerateMorpheusFiles_IsInternalFalse_AgentReviewer(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/agents/reviewer.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"zerolog", "zerolog (originating project-specific logging)"},
		{"Bun struct", "Bun struct tags (originating project-specific ORM)"},
		{"Viper", "Viper (originating project-specific config)"},
		{"Wire DI", "Wire DI (originating project-specific DI)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false agent reviewer should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_AgentTester verifies that the .claude/agents/tester.md
// rendered for a originating project project contains Ginkgo/gomock-specific content.
func TestGenerateMorpheusFiles_IsInternalTrue_AgentTester(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/agents/tester.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"Ginkgo", "Ginkgo test framework"},
		{"gomock", "gomock mocking library"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true agent tester missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_AgentTester verifies that the .claude/agents/tester.md
// rendered for a non-originating project project strips Ginkgo/gomock content and includes generic fallbacks.
func TestGenerateMorpheusFiles_IsInternalFalse_AgentTester(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/agents/tester.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"Ginkgo", "Ginkgo (originating project test framework)"},
		{"gomock", "gomock (originating project mocking library)"},
		{"GinkgoT", "GinkgoT() controller helper"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false agent tester should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Verify generic fallback content references CLAUDE.md for project conventions.
	if !strings.Contains(content, "CLAUDE.md") {
		t.Error("IsInternal=false agent tester should reference CLAUDE.md for project conventions")
	}
}

// TestInfraCompose_IsInternal_False is a regression test for Issue #58.
// It verifies that when IsInternal=false the docker-compose.infra.yml output uses
// service-scoped names (no internal_ prefix on DB, no internal- prefix on container names).
func TestInfraCompose_IsInternal_False(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "docker-compose.infra.yml")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"internal_main_db", "originating project database name"},
		{"container_name: internal-", "internal- prefixed container name"},
		{"test-service-rabbitmq:5672", "RABBITMQ_URL must use service key 'rabbitmq', not container name"},
	}
	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false docker-compose.infra.yml should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	mustContain := []struct {
		substr string
		label  string
	}{
		{"DATABASE_USER: test-service", "service-scoped database user"},
		{"test-service_db", "service-scoped database name"},
		{"RABBITMQ_URL: amqp://test-service:dev_password@rabbitmq:5672/", "RABBITMQ_URL with service key hostname"},
	}
	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false docker-compose.infra.yml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestInfraCompose_IsInternal_True is a regression test for Issue #58.
// It verifies that when IsInternal=true the docker-compose.infra.yml output uses
// the shared internal_ database name and internal- prefixed container names.
func TestInfraCompose_IsInternal_True(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "docker-compose.infra.yml")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"internal_main_db", "shared originating project database name"},
		{"container_name: internal-", "internal- prefixed container name"},
	}
	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true docker-compose.infra.yml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestNoInternalToolsOutsideConditionals is a regression guard: reads the raw .tmpl source files
// and asserts that originating project-specific tool names (that must not appear in generic output) only
// appear inside {{ if .IsInternal }} blocks. This prevents future template edits from
// accidentally leaking originating project-specific tooling into non-originating project project output.
func TestNoInternalToolsOutsideConditionals(t *testing.T) {
	templatePaths := []string{
		"templates/go/.autonomous/prompts/developer.md.tmpl",
		"templates/go/.autonomous/prompts/reviewer.md.tmpl",
		"templates/go/.autonomous/prompts/tester.md.tmpl",
		"templates/go/.claude/rules/go-service.md.tmpl",
		"templates/go/.claude/agents/reviewer.md.tmpl",
		"templates/go/.claude/agents/tester.md.tmpl",
		"templates/go/.claude/agents/developer.md.tmpl",
		"templates/go/.claude/agents/manager.md.tmpl",
		"templates/go/.claude/agents/strategist.md.tmpl",
		"templates/go/.claude/skills/secret-scan/SKILL.md.tmpl",
		"templates/go/.claude/skills/dep-audit/SKILL.md.tmpl",
		"templates/go/.claude/skills/security-scan/SKILL.md.tmpl",
		"templates/node/.claude/skills/secret-scan/SKILL.md.tmpl",
		"templates/node/.claude/skills/dep-audit/SKILL.md.tmpl",
		"templates/node/.claude/skills/security-scan/SKILL.md.tmpl",
	}

	// Terms that must only appear inside {{ if .IsInternal }} blocks — not in generic sections.
	internalOnlyTerms := []string{
		"zerolog",
		"chi/v5",
		"chi v5", // display spelling used in developer agent template
		"wire ./cmd",
		"Wire DI",
		"Wire providers",
		"Ginkgo",
		"Gomock",
		"gomock", // lowercase spelling used in developer agent template
		"Viper",
		"Bun ORM",
		"Bun struct",
		"sk_live_", // Stripe live secret key prefix
		"amqp://",  // RabbitMQ connection string with embedded creds
	}

	for _, tmplPath := range templatePaths {
		data, err := morpheusFS.ReadFile(tmplPath)
		if err != nil {
			t.Errorf("could not read embedded template %s: %v", tmplPath, err)
			continue
		}
		assertInternalTermsInConditionals(t, tmplPath, string(data), internalOnlyTerms)
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_TaskDecisions verifies that tasks/DECISIONS.md
// rendered for a non-originating project project does NOT contain originating project-specific ADR content like
// Bun ORM, Google Wire, Supabase, originating project database, etc.
func TestGenerateMorpheusFiles_IsInternalFalse_TaskDecisions(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "tasks/DECISIONS.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"uptrace/bun", "Bun ORM"},
		{"Google Wire", "Google Wire DI"},
		{"Supabase", "Supabase"},
		{"internal_main_db", "originating project shared database"},
		{"originating project standard", "originating project standard reference"},
		{"originating project auth", "originating project auth service"},
		{"originating project Redis", "originating project Redis instance"},
		{"zerolog", "zerolog (originating project logging)"},
		{"Ginkgo", "Ginkgo (originating project testing)"},
		{"New Relic", "New Relic (originating project observability)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false tasks/DECISIONS.md should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Verify generic stubs are present
	mustContain := []struct {
		substr string
		label  string
	}{
		{"Add your rationale here", "generic rationale placeholder"},
		{"Architecture Pattern", "generic architecture decision"},
		{"Entity Design", "generic entity design decision"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false tasks/DECISIONS.md missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_TaskDecisions verifies that tasks/DECISIONS.md
// rendered for a originating project project preserves all originating project-specific ADR content.
func TestGenerateMorpheusFiles_IsInternalTrue_TaskDecisions(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "tasks/DECISIONS.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"uptrace/bun", "Bun ORM ADR"},
		{"Google Wire", "Google Wire DI ADR"},
		{"internal_main_db", "originating project shared database"},
		{"originating project standard", "originating project standard reference"},
		{"zerolog", "zerolog logging"},
		{"Ginkgo", "Ginkgo testing framework"},
		{"New Relic", "New Relic observability"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true tasks/DECISIONS.md missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// nodeProjectContext returns a ProjectContext for a originating project Node.js project (IsInternal=true, IsNode=true).
func nodeProjectContext() *ProjectContext {
	ctx := testProjectContext()
	ctx.Tech = "node"
	ctx.IsGo = false
	ctx.IsNode = true
	ctx.NodeArchitecture = "4-layer"
	return ctx
}

// nonInternalNodeContext returns a ProjectContext for a non-originating project Node.js project.
func nonInternalNodeContext() *ProjectContext {
	ctx := nodeProjectContext()
	ctx.IsInternal = false
	ctx.ProjectType = "general"
	return ctx
}

// ---------------------------------------------------------------------------
// go/.claude/agents/developer.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoAgentDeveloper verifies that the Go developer
// agent rendered for a originating project project contains chi/Bun/Wire/zerolog/Ginkgo stack references.
func TestGenerateMorpheusFiles_IsInternalTrue_GoAgentDeveloper(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/agents/developer.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"chi v5", "chi v5 router reference"},
		{"Bun ORM", "Bun ORM reference"},
		{"Google Wire", "Google Wire DI reference"},
		{"zerolog", "zerolog logging reference"},
		{"Ginkgo/Gomega", "Ginkgo/Gomega testing reference"},
		{"wire ./cmd", "wire generate command"},
		{"in the originating project platform", "originating project platform identifier"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Go developer agent missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoAgentDeveloper verifies that the Go developer
// agent rendered for a non-originating project project strips originating project stack references and uses generic fallbacks.
func TestGenerateMorpheusFiles_IsInternalFalse_GoAgentDeveloper(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/agents/developer.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"chi v5", "chi v5 (originating project-specific router)"},
		{"Bun ORM", "Bun ORM (originating project-specific)"},
		{"Google Wire", "Google Wire (originating project-specific DI)"},
		{"zerolog", "zerolog (originating project-specific logging)"},
		{"Ginkgo/Gomega", "Ginkgo/Gomega (originating project-specific test framework)"},
		{"wire ./cmd", "wire generate command (originating project-specific)"},
		{"in the originating project platform", "originating project platform identifier"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go developer agent should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	mustContain := []struct {
		substr string
		label  string
	}{
		{"project conventions", "generic project conventions fallback"},
		{"ORM/query layer", "generic ORM reference"},
		{"project's logging library", "generic logging reference"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go developer agent missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// go/.claude/agents/manager.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoAgentManager verifies that the Go manager
// agent rendered for a originating project project contains originating project platform and stack references.
func TestGenerateMorpheusFiles_IsInternalTrue_GoAgentManager(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/agents/manager.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"chi/Bun/Wire patterns", "originating project stack reference in team table"},
		{"Ginkgo/Gomega tests", "Ginkgo/Gomega in tester description"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Go manager agent missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoAgentManager verifies that the Go manager
// agent rendered for a non-originating project project strips originating project-specific team descriptions.
func TestGenerateMorpheusFiles_IsInternalFalse_GoAgentManager(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/agents/manager.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"chi/Bun/Wire patterns", "originating project stack reference in team table"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go manager agent should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	mustContain := []struct {
		substr string
		label  string
	}{
		{"implements features and fixes bugs", "generic developer description"},
		{"Writes tests, validates implementations", "generic tester description"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go manager agent missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// go/.claude/agents/security-reviewer.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoAgentSecurityReviewer verifies that the
// security-reviewer agent rendered for a originating project project contains originating project-specific
// security checks for JWT, HMAC, and RabbitMQ.
func TestGenerateMorpheusFiles_IsInternalTrue_GoAgentSecurityReviewer(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/agents/security-reviewer.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"originating project-specific checks", "originating project-specific checks heading"},
		{"JWT validation on all protected endpoints", "JWT check (testProjectContext has HasJWT=true)"},
		{"RabbitMQ connection strings", "RabbitMQ credential check (testProjectContext has HasRabbitMQ=true)"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Go security-reviewer missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoAgentSecurityReviewer verifies that the
// security-reviewer agent rendered for a non-originating project project omits all originating project-specific
// security checks while retaining the generic OWASP investigation protocol.
func TestGenerateMorpheusFiles_IsInternalFalse_GoAgentSecurityReviewer(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/agents/security-reviewer.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"originating project-specific checks", "originating project-specific checks heading"},
		{"JWT validation on all protected endpoints", "originating project JWT check"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go security-reviewer should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic OWASP protocol must still be present.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"OWASP Top 10", "OWASP Top 10 table"},
		{"Investigation Protocol", "investigation protocol section"},
		{"SECURITY_APPROVED", "verdict format"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go security-reviewer missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// go/.claude/agents/strategist.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoAgentStrategist verifies that the Go
// strategist agent rendered for a originating project project includes the full originating project stack reference.
func TestGenerateMorpheusFiles_IsInternalTrue_GoAgentStrategist(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/agents/strategist.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"Bun ORM", "Bun ORM in tech stack"},
		{"Wire DI", "Wire DI in tech stack"},
		{"zerolog", "zerolog in tech stack"},
		{"Ginkgo/Gomega", "Ginkgo/Gomega in tech stack"},
		{"PostgreSQL via Bun ORM", "PostgreSQL via Bun ORM"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Go strategist missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoAgentStrategist verifies that the Go
// strategist agent rendered for a non-originating project project uses generic stack references.
func TestGenerateMorpheusFiles_IsInternalFalse_GoAgentStrategist(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/agents/strategist.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"Bun ORM", "Bun ORM (originating project-specific)"},
		{"Wire DI", "Wire DI (originating project-specific)"},
		{"zerolog", "zerolog (originating project-specific)"},
		{"Ginkgo/Gomega", "Ginkgo/Gomega (originating project-specific)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go strategist should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic fallback must be present.
	if !strings.Contains(content, "CLAUDE.md for project-specific stack") {
		t.Error("IsInternal=false Go strategist missing generic stack reference: expected 'CLAUDE.md for project-specific stack'")
	}
}

// ---------------------------------------------------------------------------
// go/.claude/skills/owasp-review/SKILL.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoSkillOwaspReview verifies that the
// owasp-review skill rendered for a originating project project includes Stripe and JWT triggers.
func TestGenerateMorpheusFiles_IsInternalTrue_GoSkillOwaspReview(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/skills/owasp-review/SKILL.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"Stripe webhook signature verification", "Stripe webhook trigger"},
		{"JWT token validation or session management", "JWT trigger"},
		{"RabbitMQ message handling or queue configuration", "RabbitMQ trigger (HasRabbitMQ=true)"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Go owasp-review skill missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoSkillOwaspReview verifies that the
// owasp-review skill rendered for a non-originating project project omits originating project-specific triggers.
func TestGenerateMorpheusFiles_IsInternalFalse_GoSkillOwaspReview(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/skills/owasp-review/SKILL.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"Stripe webhook signature verification", "Stripe webhook (originating project-specific)"},
		{"JWT token validation or session management", "JWT trigger (originating project-specific)"},
		{"RabbitMQ message handling or queue configuration", "RabbitMQ trigger (originating project-specific)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go owasp-review skill should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic OWASP reference table must be present.
	if !strings.Contains(content, "OWASP Top 10:2025 Reference") {
		t.Error("IsInternal=false Go owasp-review skill missing OWASP reference table")
	}
}

// ---------------------------------------------------------------------------
// go/.claude/skills/secret-scan/SKILL.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoSkillSecretScan verifies that the
// secret-scan skill rendered for a originating project project includes Stripe and JWT patterns.
func TestGenerateMorpheusFiles_IsInternalTrue_GoSkillSecretScan(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), ".claude/skills/secret-scan/SKILL.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"originating project-specific patterns", "originating project-specific patterns heading"},
		{"sk_live_", "Stripe live key pattern"},
		{"JWT_SECRET", "JWT secret pattern"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Go secret-scan skill missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoSkillSecretScan verifies that the
// secret-scan skill rendered for a non-originating project project omits originating project-specific secret patterns.
func TestGenerateMorpheusFiles_IsInternalFalse_GoSkillSecretScan(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), ".claude/skills/secret-scan/SKILL.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"originating project-specific patterns", "originating project-specific patterns heading"},
		{"sk_live_", "Stripe live key pattern (originating project-specific)"},
		{"JWT_SECRET", "JWT secret pattern (originating project-specific)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go secret-scan skill should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic AWS and GitHub token patterns must be present.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"gitleaks detect", "gitleaks invocation"},
		{"AKIA", "AWS credential pattern"},
		{"ghp_", "GitHub token pattern"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Go secret-scan skill missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// go/CLAUDE.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoCLAUDEmd verifies that CLAUDE.md rendered
// for a originating project project references the shared internal_main_db and internal network.
func TestGenerateMorpheusFiles_IsInternalTrue_GoCLAUDEmd(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "CLAUDE.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"shared internal_main_db", "shared originating project database name"},
		{"internal-network", "internal network reference"},
		{"with shared infra", "make dev with shared infra description"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true CLAUDE.md missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoCLAUDEmd verifies that CLAUDE.md rendered
// for a non-originating project project uses service-scoped database name and no internal network.
func TestGenerateMorpheusFiles_IsInternalFalse_GoCLAUDEmd(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "CLAUDE.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"shared internal_main_db", "originating project shared database name"},
		{"internal-network", "originating project network name"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false CLAUDE.md should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Service-scoped database name must be present.
	if !strings.Contains(content, "test-service_db") {
		t.Error("IsInternal=false CLAUDE.md missing service-scoped database name: expected 'test-service_db'")
	}
}

// ---------------------------------------------------------------------------
// go/docker-compose.yml.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoDockerCompose verifies that docker-compose.yml
// rendered for a originating project project uses internal- prefixed containers and shared network.
func TestGenerateMorpheusFiles_IsInternalTrue_GoDockerCompose(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "docker-compose.yml")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"internal-network", "shared internal network"},
		{"DATABASE_NAME: internal_main_db", "shared originating project database name"},
		{"DATABASE_USER: internal", "originating project database user"},
		{"internal-test-service-api", "internal-prefixed service name"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true docker-compose.yml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoDockerCompose verifies that docker-compose.yml
// rendered for a non-originating project project uses service-scoped names and a standalone network.
func TestGenerateMorpheusFiles_IsInternalFalse_GoDockerCompose(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "docker-compose.yml")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"internal-network", "originating project shared network"},
		{"internal_main_db", "originating project shared database name"},
		{"DATABASE_USER: internal", "originating project database user"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false docker-compose.yml should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	mustContain := []struct {
		substr string
		label  string
	}{
		{"test-service-network", "service-scoped network"},
		{"DATABASE_NAME: test-service_db", "service-scoped database name"},
		{"DATABASE_USER: test-service", "service-scoped database user"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false docker-compose.yml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// go/Makefile.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoMakefile verifies that Makefile rendered
// for a originating project project uses internal_main_db DSN and includes Wire targets.
func TestGenerateMorpheusFiles_IsInternalTrue_GoMakefile(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "Makefile")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"internal_main_db", "originating project shared database in DATABASE_DSN"},
		{"generate-wire", "Wire generate target"},
		{"wire ./cmd/api", "wire invocation for api"},
		{"github.com/google/wire/cmd/wire@latest", "wire install in install-tools"},
		{"ginkgo@latest", "ginkgo install in install-tools"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Makefile missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoMakefile verifies that Makefile rendered
// for a non-originating project project uses service-scoped DSN and omits Wire targets.
func TestGenerateMorpheusFiles_IsInternalFalse_GoMakefile(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "Makefile")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"internal_main_db", "originating project shared database (originating project-specific)"},
		{"generate-wire", "Wire generate target (originating project-specific)"},
		{"wire ./cmd", "wire invocation (originating project-specific)"},
		{"github.com/google/wire", "Wire install (originating project-specific)"},
		{"ginkgo@latest", "ginkgo install (originating project-specific)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Makefile should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Service-scoped DSN must be present.
	if !strings.Contains(content, "test-service_db") {
		t.Error("IsInternal=false Makefile missing service-scoped database name: expected 'test-service_db'")
	}
}

// ---------------------------------------------------------------------------
// go/external-configs/api-local.yaml.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoApiLocalYaml verifies that api-local.yaml
// rendered for a originating project project uses the shared internal user and internal_main_db.
func TestGenerateMorpheusFiles_IsInternalTrue_GoApiLocalYaml(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "external-configs/api/api-local.yaml")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"user: internal", "originating project database user"},
		{"name: internal_main_db", "originating project shared database name"},
		{`url: "amqp://internal:dev_password@localhost:5672/"`, "originating project RabbitMQ URL (HasRabbitMQ=true)"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true api-local.yaml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoApiLocalYaml verifies that api-local.yaml
// rendered for a non-originating project project uses env-var placeholders instead of hardcoded internal credentials.
func TestGenerateMorpheusFiles_IsInternalFalse_GoApiLocalYaml(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "external-configs/api/api-local.yaml")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"user: internal", "originating project database user (hardcoded)"},
		{"name: internal_main_db", "originating project shared database name"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false api-local.yaml should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic env-var placeholders must be present.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"${DATABASE_USER:", "env-var placeholder for DATABASE_USER"},
		{"${DATABASE_NAME:", "env-var placeholder for DATABASE_NAME"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false api-local.yaml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// go/external-configs/worker-local.yaml.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_GoWorkerLocalYaml verifies that worker-local.yaml
// rendered for a originating project project uses the shared internal user and internal_main_db.
func TestGenerateMorpheusFiles_IsInternalTrue_GoWorkerLocalYaml(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "external-configs/worker/worker-local.yaml")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"user: internal", "originating project database user"},
		{"name: internal_main_db", "originating project shared database name"},
		{`url: "amqp://internal:dev_password@localhost:5672/"`, "originating project RabbitMQ URL (HasRabbitMQ=true)"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true worker-local.yaml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_GoWorkerLocalYaml verifies that worker-local.yaml
// rendered for a non-originating project project uses env-var placeholders for database credentials.
func TestGenerateMorpheusFiles_IsInternalFalse_GoWorkerLocalYaml(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "external-configs/worker/worker-local.yaml")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"user: internal", "originating project database user (hardcoded)"},
		{"name: internal_main_db", "originating project shared database name"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false worker-local.yaml should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	if !strings.Contains(content, "${DATABASE_USER:") {
		t.Error("IsInternal=false worker-local.yaml missing env-var placeholder for DATABASE_USER")
	}
}

// ---------------------------------------------------------------------------
// node/.claude/agents/developer.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentDeveloper verifies that the Node
// developer agent rendered for a originating project project identifies the originating project platform.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentDeveloper(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/agents/developer.md")

	if !strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=true Node developer agent missing originating project platform identifier")
	}
	// Generic Node stack must always be present regardless of IsInternal.
	if !strings.Contains(content, "Express") {
		t.Error("IsInternal=true Node developer agent missing Express reference")
	}
	if !strings.Contains(content, "Prisma") {
		t.Error("IsInternal=true Node developer agent missing Prisma reference")
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentDeveloper verifies that the Node
// developer agent rendered for a non-originating project project omits the originating project platform reference.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentDeveloper(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/agents/developer.md")

	if strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=false Node developer agent should NOT contain originating project platform identifier")
	}

	// Generic Node stack must still be present.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"Express", "Express framework reference"},
		{"Prisma", "Prisma ORM reference"},
		{"Pino", "Pino logging reference"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node developer agent missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// node/.claude/agents/manager.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentManager verifies that the Node
// manager agent rendered for a originating project project identifies the originating project platform.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentManager(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/agents/manager.md")

	if !strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=true Node manager agent missing originating project platform identifier")
	}
	if !strings.Contains(content, "Node.js/Express microservice") {
		t.Error("IsInternal=true Node manager agent missing Node.js/Express microservice reference")
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentManager verifies that the Node
// manager agent rendered for a non-originating project project omits the originating project platform reference.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentManager(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/agents/manager.md")

	if strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=false Node manager agent should NOT contain originating project platform identifier")
	}
	// Must still identify as Node.js service.
	if !strings.Contains(content, "Node.js/Express microservice") {
		t.Error("IsInternal=false Node manager agent missing Node.js/Express microservice reference")
	}
}

// ---------------------------------------------------------------------------
// node/.claude/agents/reviewer.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentReviewer verifies that the Node
// reviewer agent rendered for a originating project project identifies the originating project platform.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentReviewer(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/agents/reviewer.md")

	if !strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=true Node reviewer agent missing originating project platform identifier")
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentReviewer verifies that the Node
// reviewer agent rendered for a non-originating project project omits the originating project platform reference.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentReviewer(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/agents/reviewer.md")

	if strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=false Node reviewer agent should NOT contain originating project platform identifier")
	}

	// Generic review checklist must be present regardless.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"4-layer architecture compliance", "4-layer architecture checklist"},
		{"Pino for ALL logging", "Pino logging check"},
		{"Constructor DI", "Constructor DI check"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node reviewer agent missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// node/.claude/agents/security-reviewer.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentSecurityReviewer verifies that the
// Node security-reviewer rendered for a originating project project contains originating project-specific checks.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentSecurityReviewer(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/agents/security-reviewer.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"originating project-specific checks", "originating project-specific checks heading"},
		{"JWT validation on all protected endpoints", "JWT check (nodeProjectContext has HasJWT=true)"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Node security-reviewer missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentSecurityReviewer verifies that the
// Node security-reviewer rendered for a non-originating project project omits originating project-specific checks.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentSecurityReviewer(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/agents/security-reviewer.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"in the originating project platform", "originating project platform identifier"},
		{"originating project-specific checks", "originating project-specific checks heading"},
		{"JWT validation on all protected endpoints", "originating project JWT check"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node security-reviewer should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic OWASP protocol must still be present.
	if !strings.Contains(content, "OWASP Top 10:2025") {
		t.Error("IsInternal=false Node security-reviewer missing OWASP Top 10:2025 reference")
	}
}

// ---------------------------------------------------------------------------
// node/.claude/agents/strategist.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentStrategist verifies that the Node
// strategist agent rendered for a originating project project identifies the originating project platform.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentStrategist(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/agents/strategist.md")

	if !strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=true Node strategist agent missing originating project platform identifier")
	}
	// Node stack reference must be present.
	if !strings.Contains(content, "Node.js, Express, Prisma, Pino, Jest, Zod") {
		t.Error("IsInternal=true Node strategist agent missing Node.js stack reference")
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentStrategist verifies that the Node
// strategist agent rendered for a non-originating project project omits the originating project platform reference.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentStrategist(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/agents/strategist.md")

	if strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=false Node strategist agent should NOT contain originating project platform identifier")
	}
	// Node stack must still be present — strategist.md.tmpl has a fixed stack line.
	if !strings.Contains(content, "Node.js, Express, Prisma, Pino, Jest, Zod") {
		t.Error("IsInternal=false Node strategist agent missing Node.js stack reference")
	}
}

// ---------------------------------------------------------------------------
// node/.claude/agents/tester.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentTester verifies that the Node
// tester agent rendered for a originating project project identifies the originating project platform.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeAgentTester(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/agents/tester.md")

	if !strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=true Node tester agent missing originating project platform identifier")
	}
	// Jest must be referenced regardless of IsInternal.
	if !strings.Contains(content, "Jest") {
		t.Error("IsInternal=true Node tester agent missing Jest reference")
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentTester verifies that the Node
// tester agent rendered for a non-originating project project omits the originating project platform reference.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeAgentTester(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/agents/tester.md")

	if strings.Contains(content, "in the originating project platform") {
		t.Error("IsInternal=false Node tester agent should NOT contain originating project platform identifier")
	}

	// Generic Jest patterns must still be present.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"Jest", "Jest test framework reference"},
		{"Supertest", "Supertest reference"},
		{"jest.fn()", "jest.fn() mock pattern"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node tester agent missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// node/.claude/skills/owasp-review/SKILL.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeSkillOwaspReview verifies that the Node
// owasp-review skill rendered for a originating project project includes Stripe and JWT triggers.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeSkillOwaspReview(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/skills/owasp-review/SKILL.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"Stripe webhook signature verification", "Stripe webhook trigger"},
		{"JWT token validation or session management", "JWT trigger"},
		{"RabbitMQ message handling or queue configuration", "RabbitMQ trigger (HasRabbitMQ=true)"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Node owasp-review skill missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeSkillOwaspReview verifies that the Node
// owasp-review skill rendered for a non-originating project project omits originating project-specific triggers.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeSkillOwaspReview(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/skills/owasp-review/SKILL.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"Stripe webhook signature verification", "Stripe webhook (originating project-specific)"},
		{"JWT token validation or session management", "JWT trigger (originating project-specific)"},
		{"RabbitMQ message handling or queue configuration", "RabbitMQ trigger (originating project-specific)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node owasp-review skill should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic OWASP reference table must be present.
	if !strings.Contains(content, "OWASP Top 10:2025 Reference") {
		t.Error("IsInternal=false Node owasp-review skill missing OWASP reference table")
	}
}

// ---------------------------------------------------------------------------
// node/.claude/skills/secret-scan/SKILL.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeSkillSecretScan verifies that the Node
// secret-scan skill rendered for a originating project project includes Stripe and JWT patterns.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeSkillSecretScan(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), ".claude/skills/secret-scan/SKILL.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"originating project-specific patterns", "originating project-specific patterns heading"},
		{"sk_live_", "Stripe live key pattern"},
		{"JWT_SECRET", "JWT secret pattern"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Node secret-scan skill missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeSkillSecretScan verifies that the Node
// secret-scan skill rendered for a non-originating project project omits originating project-specific secret patterns.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeSkillSecretScan(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), ".claude/skills/secret-scan/SKILL.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"originating project-specific patterns", "originating project-specific patterns heading"},
		{"sk_live_", "Stripe live key pattern (originating project-specific)"},
		{"JWT_SECRET", "JWT secret pattern (originating project-specific)"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node secret-scan skill should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic patterns must still be present.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"gitleaks detect", "gitleaks invocation"},
		{"ghp_", "GitHub token pattern"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node secret-scan skill missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// ---------------------------------------------------------------------------
// node/tasks/DECISIONS.md.tmpl
// ---------------------------------------------------------------------------

// TestGenerateMorpheusFiles_IsInternalTrue_NodeTaskDecisions verifies that the Node
// tasks/DECISIONS.md rendered for a originating project project contains originating project ADR content.
func TestGenerateMorpheusFiles_IsInternalTrue_NodeTaskDecisions(t *testing.T) {
	content := renderAndRead(t, nodeProjectContext(), "tasks/DECISIONS.md")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"internal_main_db", "originating project shared database name"},
		{"originating project standard for Node.js microservices", "originating project standard reference"},
		{"originating project Node.js standard", "originating project Node.js standard reference"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true Node tasks/DECISIONS.md missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_NodeTaskDecisions verifies that the Node
// tasks/DECISIONS.md rendered for a non-originating project project uses generic placeholder content.
func TestGenerateMorpheusFiles_IsInternalFalse_NodeTaskDecisions(t *testing.T) {
	content := renderAndRead(t, nonInternalNodeContext(), "tasks/DECISIONS.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"internal_main_db", "originating project shared database"},
		{"originating project standard", "originating project standard reference"},
	}

	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node tasks/DECISIONS.md should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Generic stubs must be present.
	mustContain := []struct {
		substr string
		label  string
	}{
		{"Add your rationale here", "generic rationale placeholder"},
		{"Architecture Pattern", "generic architecture decision"},
	}

	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false Node tasks/DECISIONS.md missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// assertInternalTermsInConditionals scans template source text and verifies that each
// occurrence of a originating project-specific term appears only within a {{ if .IsInternal }} block.
//
// The scanner uses a stack that tracks ALL {{ if ... }} opens (not just IsInternal ones).
// Each stack frame records whether it was opened by {{ if .IsInternal }}. A term is
// considered "inside IsInternal" if any frame on the stack has isIsInternal=true. This
// correctly handles nested non-IsInternal conditionals (e.g., {{ if .HasWorker }} inside
// an {{ if .IsInternal }} block) that would confuse a simple depth counter.
//
// Known limitations:
//   - Plain {{ else }} IS handled: when encountered, the top stack frame's isIsInternal
//     is flipped to false, correctly modeling the transition from the IsInternal branch
//     to the generic branch.
//   - {{ else if .IsInternal }} is NOT handled — only {{ if .IsInternal }} opens are
//     recognized as IsInternal frames. An {{ else if }} would need to pop the current
//     frame and push a new one, which this scanner does not implement.
func assertInternalTermsInConditionals(t *testing.T, tmplPath, content string, terms []string) {
	t.Helper()

	const isInternalToken = "{{ if .IsInternal }}"
	const ifPrefix = "{{ if "
	const endToken = "{{ end }}"
	const elseToken = "{{ else }}"
	const elseTokenCompact = "{{else}}"

	type stackFrame struct {
		isIsInternal bool
	}

	// Build a per-position "insideIsInternal" map using a stack of all if/end blocks.
	insideIsInternal := make([]bool, len(content))
	var stack []stackFrame
	pos := 0

	stackHasIsInternal := func() bool {
		for _, f := range stack {
			if f.isIsInternal {
				return true
			}
		}
		return false
	}

	for pos < len(content) {
		if strings.HasPrefix(content[pos:], isInternalToken) {
			stack = append(stack, stackFrame{isIsInternal: true})
			for i := pos; i < pos+len(isInternalToken) && i < len(content); i++ {
				insideIsInternal[i] = stackHasIsInternal()
			}
			pos += len(isInternalToken)
		} else if strings.HasPrefix(content[pos:], ifPrefix) {
			stack = append(stack, stackFrame{isIsInternal: false})
			inside := stackHasIsInternal()
			// Advance past the entire {{ if ... }} token (find the closing "}}")
			closeIdx := strings.Index(content[pos:], "}}")
			tokenLen := 2 // fallback: just advance past "{{ if " prefix chars
			if closeIdx >= 0 {
				tokenLen = closeIdx + 2
			}
			for i := pos; i < pos+tokenLen && i < len(content); i++ {
				insideIsInternal[i] = inside
			}
			pos += tokenLen
		} else if strings.HasPrefix(content[pos:], elseToken) || strings.HasPrefix(content[pos:], elseTokenCompact) {
			// Plain {{ else }}: flip the top frame's isIsInternal to false. This models
			// crossing from the {{ if .IsInternal }} branch into the generic {{ else }}
			// branch. Content after this point is NOT inside the IsInternal conditional.
			// Note: {{ else if ... }} is not handled — see function doc comment.
			tokenLen := len(elseToken)
			if strings.HasPrefix(content[pos:], elseTokenCompact) {
				tokenLen = len(elseTokenCompact)
			}
			if len(stack) > 0 {
				stack[len(stack)-1].isIsInternal = false
			}
			inside := stackHasIsInternal()
			for i := pos; i < pos+tokenLen && i < len(content); i++ {
				insideIsInternal[i] = inside
			}
			pos += tokenLen
		} else if strings.HasPrefix(content[pos:], endToken) {
			inside := stackHasIsInternal()
			for i := pos; i < pos+len(endToken) && i < len(content); i++ {
				insideIsInternal[i] = inside
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			pos += len(endToken)
		} else {
			insideIsInternal[pos] = stackHasIsInternal()
			pos++
		}
	}

	// For each term, find all occurrences and check whether they are inside an IsInternal block.
	for _, term := range terms {
		searchFrom := 0
		for {
			idx := strings.Index(content[searchFrom:], term)
			if idx == -1 {
				break
			}
			absIdx := searchFrom + idx
			if !insideIsInternal[absIdx] {
				// Compute line number for the error message.
				lineNum := strings.Count(content[:absIdx], "\n") + 1
				lineStart := strings.LastIndex(content[:absIdx], "\n") + 1
				lineEnd := strings.Index(content[absIdx:], "\n")
				var lineText string
				if lineEnd == -1 {
					lineText = content[lineStart:]
				} else {
					lineText = content[lineStart : absIdx+lineEnd]
				}
				t.Errorf("%s line %d: %q found outside {{ if .IsInternal }} block: %s",
					tmplPath, lineNum, term, strings.TrimSpace(lineText))
			}
			searchFrom = absIdx + len(term)
		}
	}
}
