package morpheus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// renderLoopAndRead generates all loop files to a temp dir and returns the content
// of the given relative path. It fails the test immediately on generation or read errors.
func renderLoopAndRead(t *testing.T, rel string) string {
	t.Helper()
	dir := t.TempDir()
	ctx := NewLoopContext("test-project", "build a REST API", CycleConfig{
		MaxCycles: 12, DeveloperMaxTurns: 30, TesterMaxTurns: 20, ReviewerMaxTurns: 12,
	})
	_, err := GenerateLoopFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateLoopFiles failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("could not read %s: %v", rel, err)
	}
	return string(content)
}

// readLoopSh reads the embedded loop.sh static template directly.
func readLoopSh(t *testing.T) string {
	t.Helper()
	data, err := morpheusFS.ReadFile("templates/shared/loop.sh")
	if err != nil {
		t.Fatalf("could not read embedded loop.sh: %v", err)
	}
	return string(data)
}

// ─── loop.sh behavioral checks ────────────────────────────────────────────────

// TestLoopSh_CheckSecurityApproved verifies the check_security_approved function in
// loop.sh uses an anchored, case-sensitive grep (grep -q "^## Verdict:.*SECURITY_APPROVED")
// against the cycle security file and follows the same structure as check_reviewer_approved.
func TestLoopSh_CheckSecurityApproved(t *testing.T) {
	content := readLoopSh(t)

	// Function must be defined
	if !strings.Contains(content, "check_security_approved()") {
		t.Fatal("loop.sh: check_security_approved() function not found")
	}

	// Must use anchored grep to match verdict line (prevents false-passes from body text)
	if !strings.Contains(content, `grep -q "^## Verdict:.*SECURITY_APPROVED"`) {
		t.Error(`loop.sh: check_security_approved must use grep -q "^## Verdict:.*SECURITY_APPROVED"`)
	}

	// Must reference the cycle security file with the correct naming pattern
	if !strings.Contains(content, "cycle-${cycle_padded}-security.md") {
		t.Error("loop.sh: check_security_approved must reference cycle-${cycle_padded}-security.md")
	}

	// On match must return 0 (success); no-file case returns 1
	// Verify the function handles missing file gracefully (return 1)
	if !strings.Contains(content, "No security file for cycle") {
		t.Error("loop.sh: check_security_approved must log warning when security file is missing")
	}
}

// TestLoopSh_ManagerPhaseGuard verifies that the manager phase is guarded with
// [ "$cycle" -gt 1 ] and is not run on the first cycle.
func TestLoopSh_ManagerPhaseGuard(t *testing.T) {
	content := readLoopSh(t)

	// Guard condition must be present
	if !strings.Contains(content, `[ "$cycle" -gt 1 ]`) {
		t.Error(`loop.sh: manager phase must be guarded with [ "$cycle" -gt 1 ]`)
	}

	// Manager prompt path must be referenced
	if !strings.Contains(content, "${PROMPTS_DIR}/manager.md") {
		t.Error("loop.sh: manager phase must reference ${PROMPTS_DIR}/manager.md")
	}

	// Manager must use $MANAGER_TOOLS and $MANAGER_MAX_TURNS
	if !strings.Contains(content, "$MANAGER_TOOLS") {
		t.Error("loop.sh: manager phase must use $MANAGER_TOOLS")
	}
	if !strings.Contains(content, "$MANAGER_MAX_TURNS") {
		t.Error("loop.sh: manager phase must use $MANAGER_MAX_TURNS")
	}
}

// TestLoopSh_StrategistPhaseCondition verifies that the strategist phase checks for
// decisions-pending.md containing lines starting with "## " before running.
func TestLoopSh_StrategistPhaseCondition(t *testing.T) {
	content := readLoopSh(t)

	// Must check for "^## " pattern in decisions-pending.md
	if !strings.Contains(content, `grep -q "^## " "${TASKS_DIR}/decisions-pending.md"`) {
		t.Error(`loop.sh: strategist phase must check grep -q "^## " in decisions-pending.md`)
	}

	// Strategist prompt path must be referenced
	if !strings.Contains(content, "${PROMPTS_DIR}/strategist.md") {
		t.Error("loop.sh: strategist phase must reference ${PROMPTS_DIR}/strategist.md")
	}

	// Strategist must use $STRATEGIST_TOOLS and $STRATEGIST_MAX_TURNS
	if !strings.Contains(content, "$STRATEGIST_TOOLS") {
		t.Error("loop.sh: strategist phase must use $STRATEGIST_TOOLS")
	}
	if !strings.Contains(content, "$STRATEGIST_MAX_TURNS") {
		t.Error("loop.sh: strategist phase must use $STRATEGIST_MAX_TURNS")
	}
}

// TestLoopSh_SecurityPhaseAfterReviewer verifies that the security phase is invoked
// after the reviewer phase in the main loop body.
func TestLoopSh_SecurityPhaseAfterReviewer(t *testing.T) {
	content := readLoopSh(t)

	// Security prompt path must be referenced
	if !strings.Contains(content, "${PROMPTS_DIR}/security.md") {
		t.Error("loop.sh: security phase must reference ${PROMPTS_DIR}/security.md")
	}

	// Security must use $SECURITY_TOOLS and $SECURITY_MAX_TURNS
	if !strings.Contains(content, "$SECURITY_TOOLS") {
		t.Error("loop.sh: security phase must use $SECURITY_TOOLS")
	}
	if !strings.Contains(content, "$SECURITY_MAX_TURNS") {
		t.Error("loop.sh: security phase must use $SECURITY_MAX_TURNS")
	}

	// Verify ordering: reviewer invocation must appear before security invocation
	reviewerIdx := strings.Index(content, `"reviewer"`)
	securityIdx := strings.Index(content, `"security"`)
	if reviewerIdx == -1 {
		t.Fatal("loop.sh: reviewer phase not found")
	}
	if securityIdx == -1 {
		t.Fatal("loop.sh: security phase not found")
	}
	if securityIdx <= reviewerIdx {
		t.Errorf("loop.sh: security phase (pos %d) must come AFTER reviewer phase (pos %d)", securityIdx, reviewerIdx)
	}
}

// TestLoopSh_DualGateTermination verifies that loop termination requires BOTH
// check_reviewer_approved AND check_security_approved to return 0.
func TestLoopSh_DualGateTermination(t *testing.T) {
	content := readLoopSh(t)

	// Verify the dual gate is atomic — both checks on the same if line
	compoundGate := `check_reviewer_approved "$cycle_padded" && check_security_approved "$cycle_padded"`
	if !strings.Contains(content, compoundGate) {
		t.Errorf("loop.sh does not contain compound dual-gate condition: %s", compoundGate)
	}

	// Verify the exit reason reflects dual-gate approval
	if !strings.Contains(content, "ALL_GATES_APPROVED") {
		t.Error("loop.sh: exit reason for successful termination must be ALL_GATES_APPROVED")
	}
}

// TestLoopSh_PreflightLogsSixTurnCounts verifies that the preflight log line mentions
// all 6 agent turn count variables (dev, test, review, security, manager, strategist).
func TestLoopSh_PreflightLogsSixTurnCounts(t *testing.T) {
	content := readLoopSh(t)

	// All 6 turn count variables must appear in preflight logging
	counts := []struct {
		varName string
		label   string
	}{
		{"DEVELOPER_MAX_TURNS", "developer turns"},
		{"TESTER_MAX_TURNS", "tester turns"},
		{"REVIEWER_MAX_TURNS", "reviewer turns"},
		{"SECURITY_MAX_TURNS", "security turns"},
		{"MANAGER_MAX_TURNS", "manager turns"},
		{"STRATEGIST_MAX_TURNS", "strategist turns"},
	}
	for _, c := range counts {
		if !strings.Contains(content, c.varName) {
			t.Errorf("loop.sh preflight: must reference %s (%s)", c.varName, c.label)
		}
	}
}

// ─── Loop config template (loop/config.sh.tmpl) content checks ───────────────

// TestLoopConfigTemplate_NewAgentVars verifies that the loop config template
// contains all 6 new config variables introduced by the 5-agent expansion.
func TestLoopConfigTemplate_NewAgentVars(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/config.sh")

	vars := []struct {
		varName string
		value   string
	}{
		{"MANAGER_MAX_TURNS", "MANAGER_MAX_TURNS=15"},
		{"STRATEGIST_MAX_TURNS", "STRATEGIST_MAX_TURNS=20"},
		{"SECURITY_MAX_TURNS", "SECURITY_MAX_TURNS=40"},
		{"MANAGER_TOOLS", "MANAGER_TOOLS="},
		{"STRATEGIST_TOOLS", "STRATEGIST_TOOLS="},
		{"SECURITY_TOOLS", "SECURITY_TOOLS="},
	}

	for _, v := range vars {
		if !strings.Contains(content, v.value) {
			t.Errorf("loop config.sh: missing %q (expected var %s to be set)", v.value, v.varName)
		}
	}
}

// TestLoopConfigTemplate_ManagerToolsPlanning verifies that MANAGER_TOOLS does not
// include git commit (planning-only constraint).
func TestLoopConfigTemplate_ManagerToolsPlanning(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/config.sh")

	// Extract the MANAGER_TOOLS line
	lines := strings.Split(content, "\n")
	var managerLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "MANAGER_TOOLS=") {
			managerLine = line
			break
		}
	}

	if managerLine == "" {
		t.Fatal("loop config.sh: MANAGER_TOOLS not found")
	}

	// Manager must NOT have git commit — planning only
	if strings.Contains(managerLine, "git commit") {
		t.Error("loop config.sh: MANAGER_TOOLS must not include git commit (planning only)")
	}

	// Manager must have Read to review previous cycles
	if !strings.Contains(managerLine, "Read") {
		t.Error("loop config.sh: MANAGER_TOOLS must include Read")
	}
}

// TestLoopConfigTemplate_StrategistToolsHasWebSearch verifies that STRATEGIST_TOOLS
// includes WebSearch and WebFetch for architectural research.
func TestLoopConfigTemplate_StrategistToolsHasWebSearch(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/config.sh")

	lines := strings.Split(content, "\n")
	var strategistLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "STRATEGIST_TOOLS=") {
			strategistLine = line
			break
		}
	}

	if strategistLine == "" {
		t.Fatal("loop config.sh: STRATEGIST_TOOLS not found")
	}

	if !strings.Contains(strategistLine, "WebSearch") {
		t.Error("loop config.sh: STRATEGIST_TOOLS must include WebSearch")
	}
	if !strings.Contains(strategistLine, "WebFetch") {
		t.Error("loop config.sh: STRATEGIST_TOOLS must include WebFetch")
	}
}

// TestLoopConfigTemplate_SecurityToolsReadOnly verifies that SECURITY_TOOLS is
// read-only (no git commit) and includes sequentialthinking.
func TestLoopConfigTemplate_SecurityToolsReadOnly(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/config.sh")

	lines := strings.Split(content, "\n")
	var securityLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "SECURITY_TOOLS=") {
			securityLine = line
			break
		}
	}

	if securityLine == "" {
		t.Fatal("loop config.sh: SECURITY_TOOLS not found")
	}

	// Security must not have git commit — read-only analysis
	if strings.Contains(securityLine, "git commit") {
		t.Error("loop config.sh: SECURITY_TOOLS must not include git commit (read-only)")
	}

	// Must include sequentialthinking for vulnerability chain analysis
	if !strings.Contains(securityLine, "sequentialthinking") {
		t.Error("loop config.sh: SECURITY_TOOLS must include sequentialthinking MCP")
	}
}

// ─── Go config template (go/.autonomous/config.sh.tmpl) content checks ────────

// TestGoConfigTemplate_NewAgentVars verifies that the originating project Go config template
// also contains all 6 new config variables. Uses GenerateMorpheusFiles to render it.
func TestGoConfigTemplate_NewAgentVars(t *testing.T) {
	dir := t.TempDir()
	ctx := testProjectContext()
	_, err := GenerateMorpheusFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".autonomous/config.sh"))
	if err != nil {
		t.Fatalf("could not read .autonomous/config.sh: %v", err)
	}

	vars := []string{
		"MANAGER_MAX_TURNS=15",
		"STRATEGIST_MAX_TURNS=20",
		"SECURITY_MAX_TURNS=40",
		"MANAGER_TOOLS=",
		"STRATEGIST_TOOLS=",
		"SECURITY_TOOLS=",
	}
	for _, v := range vars {
		if !strings.Contains(string(content), v) {
			t.Errorf("go config.sh: missing %q", v)
		}
	}
}

// ─── Prompt content checks ────────────────────────────────────────────────────

// TestLoopManagerPrompt_NeverCommitCode verifies that the loop manager prompt
// explicitly prohibits code commits and source code edits.
func TestLoopManagerPrompt_NeverCommitCode(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/prompts/manager.md")

	checks := []struct {
		substr string
		label  string
	}{
		{"NEVER commit code", "prohibition on committing code"},
		{"NEVER edit source code files", "prohibition on editing source code"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.substr) {
			t.Errorf("loop manager.md: missing %s: expected to find %q", c.label, c.substr)
		}
	}
}

// TestLoopStrategistPrompt_NeverModifySource verifies that the loop strategist prompt
// explicitly prohibits source code modification and requires resolving all pending decisions.
func TestLoopStrategistPrompt_NeverModifySource(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/prompts/strategist.md")

	checks := []struct {
		substr string
		label  string
	}{
		{"NEVER modify source code", "prohibition on modifying source code"},
		{"Resolve ALL pending decisions", "requirement to resolve all decisions"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.substr) {
			t.Errorf("loop strategist.md: missing %s: expected to find %q", c.label, c.substr)
		}
	}
}

// TestLoopSecurityPrompt_VerdictStrings verifies that the loop security prompt
// contains the canonical verdict strings that check_security_approved() grep-matches.
func TestLoopSecurityPrompt_VerdictStrings(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/prompts/security.md")

	verdicts := []struct {
		verdict string
		label   string
	}{
		{"SECURITY_APPROVED", "SECURITY_APPROVED verdict string"},
		{"SECURITY_BLOCKED", "SECURITY_BLOCKED verdict string"},
	}
	for _, v := range verdicts {
		if !strings.Contains(content, v.verdict) {
			t.Errorf("loop security.md: missing %s: expected to find %q", v.label, v.verdict)
		}
	}
}

// TestLoopSecurityPrompt_HasOWASPProtocol verifies that the security prompt contains
// the full OWASP investigation protocol (Phase 3b replacement of placeholder).
func TestLoopSecurityPrompt_HasOWASPProtocol(t *testing.T) {
	content := renderLoopAndRead(t, ".autonomous/prompts/security.md")

	checks := []struct {
		substr string
		label  string
	}{
		{"## Investigation Steps", "Investigation Steps header"},
		{"OWASP Scan", "OWASP Scan step"},
		{"ASI Scan", "ASI Scan step"},
		{"Pattern Search", "Pattern Search step"},
		{"Severity Formula", "Severity Formula section"},
		{"A01", "OWASP A01 reference"},
		{"ASI01", "ASI01 reference"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.substr) {
			t.Errorf("loop security.md: missing %s: expected to find %q", c.label, c.substr)
		}
	}
}

// TestGoManagerPrompt_NeverCommitCode verifies that the originating project Go manager prompt
// also prohibits code commits (same constraint as the generic loop variant).
func TestGoManagerPrompt_NeverCommitCode(t *testing.T) {
	dir := t.TempDir()
	ctx := testProjectContext()
	_, err := GenerateMorpheusFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".autonomous/prompts/manager.md"))
	if err != nil {
		t.Fatalf("could not read .autonomous/prompts/manager.md: %v", err)
	}

	checks := []struct {
		substr string
		label  string
	}{
		{"NEVER commit code", "prohibition on committing code"},
		{"NEVER edit source code files", "prohibition on editing source code"},
	}
	for _, c := range checks {
		if !strings.Contains(string(content), c.substr) {
			t.Errorf("go manager.md: missing %s: expected to find %q", c.label, c.substr)
		}
	}
}

// TestGoSecurityPrompt_VerdictStrings verifies that the originating project Go security prompt
// also contains the canonical verdict strings.
func TestGoSecurityPrompt_VerdictStrings(t *testing.T) {
	dir := t.TempDir()
	ctx := testProjectContext()
	_, err := GenerateMorpheusFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".autonomous/prompts/security.md"))
	if err != nil {
		t.Fatalf("could not read .autonomous/prompts/security.md: %v", err)
	}

	for _, verdict := range []string{"SECURITY_APPROVED", "SECURITY_BLOCKED"} {
		if !strings.Contains(string(content), verdict) {
			t.Errorf("go security.md: missing verdict string %q", verdict)
		}
	}

	if !strings.Contains(string(content), "## Investigation Steps") {
		t.Error("go security.md: must contain full OWASP investigation protocol (## Investigation Steps)")
	}
}

// TestGoStrategistPrompt_NeverModifySource verifies the originating project Go strategist prompt
// also prohibits source modification.
func TestGoStrategistPrompt_NeverModifySource(t *testing.T) {
	dir := t.TempDir()
	ctx := testProjectContext()
	_, err := GenerateMorpheusFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".autonomous/prompts/strategist.md"))
	if err != nil {
		t.Fatalf("could not read .autonomous/prompts/strategist.md: %v", err)
	}

	if !strings.Contains(string(content), "NEVER modify source code") {
		t.Error("go strategist.md: must contain 'NEVER modify source code'")
	}
}

// ─── Rendered prompt no-template-tags guard ────────────────────────────────────

// TestNewLoopPrompts_NoUnrenderedTags verifies that all 3 new loop prompts are fully
// rendered (no {{ . }} template directives left in output).
func TestNewLoopPrompts_NoUnrenderedTags(t *testing.T) {
	dir := t.TempDir()
	ctx := NewLoopContext("test-project", "", CycleConfig{
		MaxCycles: 5, DeveloperMaxTurns: 20, TesterMaxTurns: 15, ReviewerMaxTurns: 10,
	})
	_, err := GenerateLoopFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateLoopFiles failed: %v", err)
	}

	newPrompts := []string{
		".autonomous/prompts/manager.md",
		".autonomous/prompts/strategist.md",
		".autonomous/prompts/security.md",
	}
	for _, rel := range newPrompts {
		content, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Errorf("could not read %s: %v", rel, err)
			continue
		}
		if strings.Contains(string(content), "{{ .") {
			t.Errorf("%s contains unrendered template directives", rel)
		}
	}
}

// TestNewGoPrompts_NoUnrenderedTags verifies that all 3 new originating project Go prompts are fully
// rendered (no {{ . }} template directives left in output).
func TestNewGoPrompts_NoUnrenderedTags(t *testing.T) {
	dir := t.TempDir()
	ctx := testProjectContext()
	_, err := GenerateMorpheusFiles(ctx, dir)
	if err != nil {
		t.Fatalf("GenerateMorpheusFiles failed: %v", err)
	}

	newPrompts := []string{
		".autonomous/prompts/manager.md",
		".autonomous/prompts/strategist.md",
		".autonomous/prompts/security.md",
	}
	for _, rel := range newPrompts {
		content, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Errorf("could not read %s: %v", rel, err)
			continue
		}
		if strings.Contains(string(content), "{{ .") {
			t.Errorf("%s contains unrendered template directives", rel)
		}
	}
}
