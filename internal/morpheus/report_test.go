package morpheus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── splitLessonSections ────────────────────────────────────────────────────

func TestSplitLessonSections_NoHeadings(t *testing.T) {
	content := "Just some plain text without any headings."
	sections := splitLessonSections(content)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0] != content {
		t.Errorf("expected section to equal input, got %q", sections[0])
	}
}

func TestSplitLessonSections_EmptyString(t *testing.T) {
	sections := splitLessonSections("")
	if len(sections) != 1 {
		t.Fatalf("expected 1 section for empty input, got %d", len(sections))
	}
}

func TestSplitLessonSections_ThreeH2Headings(t *testing.T) {
	content := `## First
content of first

## Second
content of second

## Third
content of third
`
	sections := splitLessonSections(content)
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}
	for i, sec := range sections {
		if !strings.HasPrefix(sec, "## ") {
			t.Errorf("section %d should start with '## ', got %q", i, sec[:10])
		}
	}
	if !strings.Contains(sections[0], "First") {
		t.Errorf("section 0 should contain 'First'")
	}
	if !strings.Contains(sections[1], "Second") {
		t.Errorf("section 1 should contain 'Second'")
	}
	if !strings.Contains(sections[2], "Third") {
		t.Errorf("section 2 should contain 'Third'")
	}
}

func TestSplitLessonSections_MixedH2AndH3(t *testing.T) {
	content := `## Top Level
some content

### Sub Level
sub content

## Another Top
more content
`
	sections := splitLessonSections(content)
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections (## and ### both split), got %d", len(sections))
	}
	if !strings.HasPrefix(sections[0], "## Top Level") {
		t.Errorf("section 0 should start with '## Top Level', got %q", sections[0][:20])
	}
	if !strings.HasPrefix(sections[1], "### Sub Level") {
		t.Errorf("section 1 should start with '### Sub Level', got %q", sections[1][:20])
	}
	if !strings.HasPrefix(sections[2], "## Another Top") {
		t.Errorf("section 2 should start with '## Another Top', got %q", sections[2][:20])
	}
}

// ─── detectKnowledgeGaps ────────────────────────────────────────────────────

func TestDetectKnowledgeGaps_NoLessonsFile(t *testing.T) {
	dir := t.TempDir()
	gaps := detectKnowledgeGaps(dir)
	if gaps != nil {
		t.Fatalf("expected nil when no lessons.md, got %d gaps", len(gaps))
	}
}

func TestDetectKnowledgeGaps_EmptyLessonsFile(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "tasks/lessons.md", "   \n\n  ", 0644)
	gaps := detectKnowledgeGaps(dir)
	if gaps != nil {
		t.Fatalf("expected nil for whitespace-only lessons.md, got %d gaps", len(gaps))
	}
}

func TestDetectKnowledgeGaps_ErrorHandlingMatch(t *testing.T) {
	dir := t.TempDir()
	content := `## Error Handling Patterns

We discovered that error wrapping was inconsistent across handlers.
The wrap function was not used properly.
`
	createFile(t, dir, "tasks/lessons.md", content, 0644)

	gaps := detectKnowledgeGaps(dir)

	found := false
	for _, g := range gaps {
		if g.Topic == "04-error-handling" {
			found = true
			if g.Lessons < 1 {
				t.Errorf("expected at least 1 lesson for 04-error-handling, got %d", g.Lessons)
			}
			// Verify matched keywords include "error" and "wrap"
			hasError, hasWrap := false, false
			for _, kw := range g.Matched {
				if kw == "error" {
					hasError = true
				}
				if kw == "wrap" {
					hasWrap = true
				}
			}
			if !hasError {
				t.Error("expected 'error' in matched keywords")
			}
			if !hasWrap {
				t.Error("expected 'wrap' in matched keywords")
			}
			break
		}
	}
	if !found {
		t.Error("expected 04-error-handling in gaps but not found")
	}
}

func TestDetectKnowledgeGaps_MultipleTopicsMatched(t *testing.T) {
	dir := t.TempDir()
	content := `## Redis Lua Scripting Issue

The redis lua script was failing silently.

## Test Coverage Gap

Test coverage dropped below 80%. Need more mock fixtures.
`
	createFile(t, dir, "tasks/lessons.md", content, 0644)

	gaps := detectKnowledgeGaps(dir)

	topicSet := map[string]bool{}
	for _, g := range gaps {
		topicSet[g.Topic] = true
	}

	if !topicSet["11-redis-caching"] {
		t.Error("expected 11-redis-caching in gaps")
	}
	if !topicSet["05-testing"] {
		t.Error("expected 05-testing in gaps")
	}
}

func TestDetectKnowledgeGaps_SortedByLessonCountDesc(t *testing.T) {
	dir := t.TempDir()
	// Create lessons where "test" appears in 3 sections and "redis" in 1
	content := `## Test Section One

We need test coverage for the handler layer.

## Test Section Two

Mock fixtures in tests are incomplete.

## Test Section Three

Test flakiness in CI.

## Redis Issue

Redis cache TTL was wrong.
`
	createFile(t, dir, "tasks/lessons.md", content, 0644)

	gaps := detectKnowledgeGaps(dir)
	if len(gaps) < 2 {
		t.Fatalf("expected at least 2 gaps, got %d", len(gaps))
	}

	// Find positions
	testIdx, redisIdx := -1, -1
	for i, g := range gaps {
		if g.Topic == "05-testing" {
			testIdx = i
		}
		if g.Topic == "11-redis-caching" {
			redisIdx = i
		}
	}
	if testIdx == -1 || redisIdx == -1 {
		t.Fatalf("missing expected topics: test=%d, redis=%d", testIdx, redisIdx)
	}
	if testIdx > redisIdx {
		t.Errorf("05-testing (3 lessons) should appear before 11-redis-caching (1 lesson), but test=%d redis=%d", testIdx, redisIdx)
	}
}

func TestDetectKnowledgeGaps_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	content := `## UPPERCASE ERRORS

We had an ERROR wrapping problem. WRAP was missing.
`
	createFile(t, dir, "tasks/lessons.md", content, 0644)

	gaps := detectKnowledgeGaps(dir)
	found := false
	for _, g := range gaps {
		if g.Topic == "04-error-handling" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 04-error-handling to match uppercase 'ERROR' — case-insensitive matching failed")
	}
}

// ─── GenerateLoopReport with knowledge gaps ─────────────────────────────────

func TestGenerateLoopReport_WithGaps(t *testing.T) {
	dir := t.TempDir()

	// Set up minimal structure for GenerateLoopReport
	createFile(t, dir, ".autonomous/loop.sh", "#!/bin/bash\nexit 0", 0755)
	createFile(t, dir, ".autonomous/config.sh", `MAX_CYCLES=3
DEVELOPER_MAX_TURNS=30
TESTER_MAX_TURNS=20
REVIEWER_MAX_TURNS=12
`, 0644)

	// Create a cycle directory with one cycle of data
	cycleData := cycleOutput{
		NumTurns:     10,
		TotalCostUSD: 0.50,
		DurationMs:   60000,
		Subtype:      "success",
	}
	data, _ := json.Marshal(cycleData)
	createFile(t, dir, "tasks/cycles/cycle-1-developer-output.json", string(data), 0644)

	// Create lessons.md with redis and testing content
	lessons := `## Redis Performance

Redis lua script timeouts are causing issues in the cache layer.

## Testing Gaps

Test coverage is insufficient for the handler endpoint.
`
	createFile(t, dir, "tasks/lessons.md", lessons, 0644)

	// Create todo.md for task counting
	createFile(t, dir, "tasks/todo.md", "- [x] task one\n- [ ] task two\n", 0644)

	report, err := GenerateLoopReport(dir)
	if err != nil {
		t.Fatalf("GenerateLoopReport failed: %v", err)
	}

	if !strings.Contains(report, "Knowledge Gap Suggestions (Loop A)") {
		t.Error("report should contain 'Knowledge Gap Suggestions (Loop A)' section")
	}

	// Should find redis and/or testing topics
	hasRedis := strings.Contains(report, "11-redis-caching")
	hasTesting := strings.Contains(report, "05-testing")
	if !hasRedis || !hasTesting {
		t.Error("report should contain both 11-redis-caching and 05-testing")
	}

	// Should contain oracle update command suggestion
	if !strings.Contains(report, "oracle update") {
		t.Error("report should contain 'oracle update' suggested command")
	}
}

func TestGenerateLoopReport_NoGaps(t *testing.T) {
	dir := t.TempDir()

	// Set up minimal structure for GenerateLoopReport
	createFile(t, dir, ".autonomous/loop.sh", "#!/bin/bash\nexit 0", 0755)
	createFile(t, dir, ".autonomous/config.sh", `MAX_CYCLES=3
DEVELOPER_MAX_TURNS=30
TESTER_MAX_TURNS=20
REVIEWER_MAX_TURNS=12
`, 0644)

	// Create a cycle directory with one cycle of data
	cycleData := cycleOutput{
		NumTurns:     5,
		TotalCostUSD: 0.25,
		DurationMs:   30000,
		Subtype:      "success",
	}
	data, _ := json.Marshal(cycleData)
	createFile(t, dir, "tasks/cycles/cycle-1-developer-output.json", string(data), 0644)

	// No lessons.md — should show "no gaps" message
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0755); err != nil {
		t.Fatal(err)
	}

	report, err := GenerateLoopReport(dir)
	if err != nil {
		t.Fatalf("GenerateLoopReport failed: %v", err)
	}

	if !strings.Contains(report, "Knowledge Gap Suggestions (Loop A)") {
		t.Error("report should contain 'Knowledge Gap Suggestions (Loop A)' section")
	}
	if !strings.Contains(report, "No knowledge gaps detected") {
		t.Error("report should contain 'No knowledge gaps detected' when no lessons.md exists")
	}
}
