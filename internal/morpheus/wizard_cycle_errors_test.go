package morpheus

import (
	"os"
	"regexp"
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

	// Every AskInput call in the Manual override block must assign to `err`
	// rather than discarding it. The prior bug pattern `mcStr, _ := wizard.AskInput`
	// must not appear anywhere in the file.
	badPattern := regexp.MustCompile(`, _ := wizard\.AskInput`)
	if matches := badPattern.FindAllString(source, -1); len(matches) > 0 {
		t.Errorf("found %d discarded AskInput errors in wizard.go — every call must propagate err:\n%v", len(matches), matches)
	}

	// Every Manual override cycle field must have a non-positive guard. We
	// require at least 8 guards total: 4 fields × 2 wizards (RunInternalWizard
	// and RunInternalWizardWithContext).
	guardPattern := regexp.MustCompile(`ctx\.(MaxCycles|DeveloperMaxTurns|TesterMaxTurns|ReviewerMaxTurns) <= 0`)
	guards := guardPattern.FindAllString(source, -1)
	if len(guards) < 8 {
		t.Errorf("expected at least 8 non-positive guards (4 fields × 2 wizards), found %d", len(guards))
	}

	// The cycle-config error messages must wrap the underlying error with %w so
	// callers can errors.Is / errors.As on huh.ErrUserAborted etc.
	wrapPattern := regexp.MustCompile(`fmt\.Errorf\("reading [^"]+: %w", err\)`)
	wraps := wrapPattern.FindAllString(source, -1)
	if len(wraps) < 8 {
		t.Errorf("expected at least 8 wrapped error returns for cycle-config AskInput sites, found %d:\n%v", len(wraps), wraps)
	}
}
