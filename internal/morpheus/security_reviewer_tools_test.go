package morpheus

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// extractReviewerTools extracts the value of the REVIEWER_TOOLS= assignment from
// a rendered config.sh. Returns the comma-separated allowlist (without quotes).
func extractReviewerTools(t *testing.T, content string) string {
	t.Helper()
	// Match: REVIEWER_TOOLS="..."
	re := regexp.MustCompile(`(?m)^REVIEWER_TOOLS="([^"]*)"`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		t.Fatal("REVIEWER_TOOLS assignment not found in config.sh")
	}
	return m[1]
}

// TestReviewerTools_GoInitTemplate_NoEdit (H-01) verifies that the morpheus Go
// init template does NOT grant Edit in REVIEWER_TOOLS. Edit is a targeted in-place
// source modification tool; granting it to a read-only reviewer contradicts
// ecosystem-conventions.md and the documented loop-template design.
func TestReviewerTools_GoInitTemplate_NoEdit(t *testing.T) {
	dir := t.TempDir()
	ctx := testProjectContext()
	if _, err := GenerateMorpheusFiles(ctx, dir); err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".autonomous/config.sh"))
	if err != nil {
		t.Fatalf("reading rendered config.sh: %v", err)
	}
	tools := extractReviewerTools(t, string(data))

	// Parse the allowlist — entries are separated by commas. A pure "Edit"
	// token must not appear. We check token-level equality to avoid matching
	// substrings like "NotebookEdit".
	for _, tok := range strings.Split(tools, ",") {
		if strings.TrimSpace(tok) == "Edit" {
			t.Errorf("go init REVIEWER_TOOLS must NOT grant Edit (H-01): got %q", tools)
		}
	}

	// Reviewer must retain Read, Glob, Grep, Agent to do its job.
	for _, required := range []string{"Read", "Glob", "Grep", "Agent"} {
		if !tokenInList(tools, required) {
			t.Errorf("go init REVIEWER_TOOLS missing required read-only tool %q; got %q", required, tools)
		}
	}
}

// TestReviewerTools_NodeInitTemplate_NoEdit (H-01) same as the go variant but for
// the Node.js init template.
func TestReviewerTools_NodeInitTemplate_NoEdit(t *testing.T) {
	dir := t.TempDir()
	ctx := nodeProjectContext()
	if _, err := GenerateMorpheusFiles(ctx, dir); err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".autonomous/config.sh"))
	if err != nil {
		t.Fatalf("reading rendered config.sh: %v", err)
	}
	tools := extractReviewerTools(t, string(data))

	for _, tok := range strings.Split(tools, ",") {
		if strings.TrimSpace(tok) == "Edit" {
			t.Errorf("node init REVIEWER_TOOLS must NOT grant Edit (H-01): got %q", tools)
		}
	}

	for _, required := range []string{"Read", "Glob", "Grep", "Agent"} {
		if !tokenInList(tools, required) {
			t.Errorf("node init REVIEWER_TOOLS missing required read-only tool %q; got %q", required, tools)
		}
	}
}

// tokenInList returns true if the exact token appears in a comma-separated list.
// Bash(...) entries are treated as opaque tokens (matched whole).
func tokenInList(list, token string) bool {
	for _, t := range strings.Split(list, ",") {
		if strings.TrimSpace(t) == token {
			return true
		}
	}
	return false
}
