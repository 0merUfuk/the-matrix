package morpheus

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestWizardCycleConfig_PropagatesAskInputErrors is a regression test for
// P0-C1 (audit 2026-04-17): the Manual override branches in RunInternalWizard
// and RunInternalWizardWithContext previously used `mcStr, _ := wizard.AskInput(...)`
// and then `strconv.Atoi(...)` with the error blank-identifier-discarded. Both
// Ctrl+C (huh.ErrUserAborted) and a broken TTY therefore produced a 0-valued
// MaxCycles / DeveloperMaxTurns / TesterMaxTurns / ReviewerMaxTurns, which
// rendered loop.sh with zeros and hung the loop silently.
//
// The fix must propagate every AskInput error with fmt.Errorf wrapping and
// fall back to the auto-calculated default when Atoi yields a non-positive
// value. This test inspects the source of wizard.go to verify both properties
// hold at the Manual override sites — a full TUI-driven test is impractical
// since huh requires a TTY, and refactoring is out-of-scope for this PR.
func TestWizardCycleConfig_PropagatesAskInputErrors(t *testing.T) {
	src, err := os.ReadFile("wizard.go")
	if err != nil {
		t.Fatalf("reading wizard.go: %v", err)
	}
	source := string(src)

	helper, err := os.ReadFile("cycle_wizard.go")
	if err != nil {
		t.Fatalf("reading cycle_wizard.go: %v", err)
	}
	helperSource := string(helper)
	combined := source + "\n" + helperSource

	// Every AskInput call in the shared cycle-value helper must propagate its
	// error rather than discarding it.
	badPattern := regexp.MustCompile(`, _ := wizard\.AskInput`)
	if matches := badPattern.FindAllString(combined, -1); len(matches) > 0 {
		t.Errorf("found %d discarded AskInput errors in cycle wizard code: %v", len(matches), matches)
	}

	// All cycle fields use the shared helper, whose single guard preserves the
	// auto-calculated default for invalid or non-positive input.
	if !strings.Contains(helperSource, "if value <= 0") {
		t.Error("cycle_wizard.go is missing the non-positive fallback guard")
	}
	if calls := strings.Count(source, "askCycleValue("); calls < 8 {
		t.Errorf("expected at least 8 shared cycle-value calls in wizard.go, found %d", calls)
	}
	loopWizard, err := os.ReadFile("loop_wizard.go")
	if err != nil {
		t.Fatalf("reading loop_wizard.go: %v", err)
	}
	if calls := strings.Count(string(loopWizard), "askCycleValue("); calls != 4 {
		t.Errorf("expected 4 shared cycle-value calls in loop_wizard.go, found %d", calls)
	}

	// The cycle-config error messages must wrap the underlying error with %w so
	// callers can errors.Is / errors.As on huh.ErrUserAborted etc.
	wrapPattern := regexp.MustCompile(`fmt\.Errorf\("reading [^"]+: %w", err\)`)
	wraps := wrapPattern.FindAllString(source, -1)
	if len(wraps) < 8 {
		t.Errorf("expected at least 8 wrapped error returns for cycle-config AskInput sites, found %d:\n%v", len(wraps), wraps)
	}
}
