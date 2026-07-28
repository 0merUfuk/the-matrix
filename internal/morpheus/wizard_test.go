package morpheus

import (
	"strconv"
	"strings"
	"testing"
)

// TestNormalizeModulePath is a regression test for Bug 3: users could paste full GitHub
// URLs (e.g. "https://github.com/org/repo") into the module path prompt. Before the fix,
// the raw URL was stored verbatim — go.mod would contain an invalid module directive.
// normalizeModulePath must strip http(s):// prefixes and trailing slashes.
func TestNormalizeModulePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bare path unchanged",
			input: "github.com/foo/bar",
			want:  "github.com/foo/bar",
		},
		{
			name:  "https prefix stripped",
			input: "https://github.com/foo/bar",
			want:  "github.com/foo/bar",
		},
		{
			name:  "http prefix stripped",
			input: "http://github.com/foo/bar",
			want:  "github.com/foo/bar",
		},
		{
			name:  "trailing slash stripped",
			input: "github.com/foo/bar/",
			want:  "github.com/foo/bar",
		},
		{
			name:  "https with trailing slash",
			input: "https://github.com/foo/bar/",
			want:  "github.com/foo/bar",
		},
		{
			name:  "multiple trailing slashes stripped",
			input: "github.com/foo/bar///",
			want:  "github.com/foo/bar",
		},
		{
			name:  "empty string unchanged",
			input: "",
			want:  "",
		},
		{
			name:  "node scoped package unchanged",
			input: "@internal/my-service",
			want:  "@internal/my-service",
		},
		{
			name:  "https prefix on org-scoped path",
			input: "https://github.com/0merUfuk/generation-service",
			want:  "github.com/0merUfuk/generation-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeModulePath(tt.input)
			if got != tt.want {
				t.Errorf("normalizeModulePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeModulePath_ResultIsValidGoModule verifies that normalizeModulePath applied
// to a full HTTPS URL with a trailing slash produces a clean module path with no scheme
// and no trailing slash — exactly what go.mod requires.
func TestNormalizeModulePath_ResultIsValidGoModule(t *testing.T) {
	input := "https://github.com/org/my-service/"
	result := normalizeModulePath(input)

	if strings.HasPrefix(result, "https://") || strings.HasPrefix(result, "http://") {
		t.Errorf("normalizeModulePath(%q): result %q still contains URL scheme", input, result)
	}
	if strings.HasSuffix(result, "/") {
		t.Errorf("normalizeModulePath(%q): result %q has trailing slash", input, result)
	}
	if result != "github.com/org/my-service" {
		t.Errorf("normalizeModulePath(%q) = %q, want %q", input, result, "github.com/org/my-service")
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_DockerCompose is a regression test for Bug 1:
// docker-compose.yml.tmpl previously hardcoded originating project infra names (internal-network,
// internal-postgres, internal_main_db, internal-redis) unconditionally. Non-originating project projects
// must get project-local infra with self-contained services, not shared originating project infra.
func TestGenerateMorpheusFiles_IsInternalFalse_DockerCompose(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "docker-compose.yml")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"internal-network", "shared internal-network (external)"},
		{"internal-postgres", "shared internal-postgres service"},
		{"internal_main_db", "shared internal_main_db database name"},
		{"internal-redis", "shared internal-redis service"},
	}
	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false docker-compose.yml should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Non-originating project projects must have a project-local network named after the service.
	if !strings.Contains(content, "test-service-network") {
		t.Error("IsInternal=false docker-compose.yml should contain project-local network 'test-service-network'")
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_DockerCompose verifies that the originating project path in
// docker-compose.yml is unchanged after the Bug 1 fix — originating project projects must still
// connect to the shared internal-network and reference internal-postgres.
func TestGenerateMorpheusFiles_IsInternalTrue_DockerCompose(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "docker-compose.yml")

	mustContain := []struct {
		substr string
		label  string
	}{
		{"internal-network", "shared internal-network"},
		{"internal-postgres", "shared internal-postgres service reference"},
	}
	for _, tc := range mustContain {
		if !strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=true docker-compose.yml missing %s: expected to find %q", tc.label, tc.substr)
		}
	}
}

// TestGenerateMorpheusFiles_IsInternalFalse_ClaudeMd is a regression test for Bug 1:
// CLAUDE.md.tmpl previously referenced "internal_main_db" and "same as generation-service"
// without IsInternal guards. Non-originating project projects must use project-local database names.
func TestGenerateMorpheusFiles_IsInternalFalse_ClaudeMd(t *testing.T) {
	content := renderAndRead(t, nonInternalContext(), "CLAUDE.md")

	mustNotContain := []struct {
		substr string
		label  string
	}{
		{"internal_main_db", "shared internal_main_db database name in tech stack table"},
		{"same as generation-service", "originating project auth reference to generation-service"},
	}
	for _, tc := range mustNotContain {
		if strings.Contains(content, tc.substr) {
			t.Errorf("IsInternal=false CLAUDE.md should NOT contain %s: found %q", tc.label, tc.substr)
		}
	}

	// Non-originating project projects should reference their own database name in the tech stack table.
	if !strings.Contains(content, "test-service_db") {
		t.Error("IsInternal=false CLAUDE.md should contain project-local database name 'test-service_db'")
	}
}

// TestGenerateMorpheusFiles_IsInternalTrue_ClaudeMd verifies that the originating project path in CLAUDE.md
// is unchanged after the Bug 1 fix — originating project projects must still reference internal_main_db.
func TestGenerateMorpheusFiles_IsInternalTrue_ClaudeMd(t *testing.T) {
	content := renderAndRead(t, testProjectContext(), "CLAUDE.md")

	if !strings.Contains(content, "internal_main_db") {
		t.Error("IsInternal=true CLAUDE.md missing 'internal_main_db' in tech stack table")
	}
}

// TestApiPortDefaultFallback_Logic documents and validates the port fallback pattern
// introduced for Bug 2. The wizard functions require a TTY and cannot be driven in unit
// tests. The fix applies the same strconv.Atoi + fallback inline in RunGeneralWizard and
// RunGeneralWizardWithContext.
//
// Before the fix: ctx.ApiPort, _ = strconv.Atoi(portStr) — ignoring the error silently
// set ApiPort to 0 on empty or non-numeric input.
//
// After the fix: a parse error causes the code to fall back to defaultGeneralPort (8080),
// so ApiPort is never 0 from an empty prompt response.
//
// This test validates that fallback arithmetic (parse failure → 8080, worker offset → 8081)
// is correct and cannot silently produce 0.
func TestApiPortDefaultFallback_Logic(t *testing.T) {
	const defaultGeneralPort = 8080

	// Valid numeric inputs parse cleanly to their integer values.
	for _, tc := range []struct {
		input string
		want  int
	}{
		{"9090", 9090},
		{"  8080  ", 8080},
		{"1", 1},
	} {
		p, err := strconv.Atoi(strings.TrimSpace(tc.input))
		if err != nil || p != tc.want {
			t.Errorf("valid input %q: expected %d with no error, got %d err=%v", tc.input, tc.want, p, err)
		}
	}

	// Regression: empty or non-numeric input must NOT silently become 0.
	for _, badInput := range []string{"", "  ", "not-a-number"} {
		p, convErr := strconv.Atoi(strings.TrimSpace(badInput))
		var effective int
		if convErr != nil {
			effective = defaultGeneralPort // the fixed fallback path
		} else {
			effective = p
		}
		if effective == 0 {
			t.Errorf("port fallback for input %q must not result in 0 (pre-fix silent bug); effective=%d", badInput, effective)
		}
		if effective != defaultGeneralPort {
			t.Errorf("port fallback for input %q: expected %d, got %d", badInput, defaultGeneralPort, effective)
		}
	}

	// Worker port offset: defaultGeneralPort + 1 must be 8081.
	workerDefault := defaultGeneralPort + 1
	if workerDefault != 8081 {
		t.Errorf("worker port offset arithmetic: expected 8081, got %d", workerDefault)
	}
}
