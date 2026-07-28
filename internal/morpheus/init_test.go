package morpheus

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestApplyContextDefaults_FillsWizardFields is a regression guard for the
// `morpheus init --context` no-TTY bug (2026-04-28 dogfood test): when neo
// init invokes `morpheus init --context ...` as a subprocess in a no-TTY
// environment (CI, scripted automation), the delta wizards (huh-based)
// silently fail and the previous code-path called os.Exit(0) with "Aborted."
// — producing no files while looking like the wizard was skipped.
//
// ApplyContextDefaults must populate every field the wizard would otherwise
// prompt for, so the chain `neo init` → `morpheus init --context` produces
// a complete scaffold end-to-end without a TTY.
func TestApplyContextDefaults_FillsWizardFields(t *testing.T) {
	tests := []struct {
		name    string
		profile ContextProfile
	}{
		{
			name: "internal-go",
			profile: ContextProfile{
				ProjectName: "billing-api",
				IsInternal:    true,
				Stacks:      []ContextStack{{Name: "go-chi", Language: "go", Framework: "chi"}},
			},
		},
		{
			name: "general-go",
			profile: ContextProfile{
				ProjectName: "my-go-oss",
				IsInternal:    false,
				Stacks:      []ContextStack{{Name: "go-chi", Language: "go", Framework: "chi"}},
			},
		},
		{
			name: "general-node",
			profile: ContextProfile{
				ProjectName: "api-gateway",
				IsInternal:    false,
				Stacks:      []ContextStack{{Name: "express-api", Language: "nodejs", Framework: "express"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := MapContextToProject(&tc.profile)
			ApplyContextDefaults(ctx)

			// Identity preserved.
			if ctx.ServiceName != tc.profile.ProjectName {
				t.Errorf("ServiceName = %q, want %q", ctx.ServiceName, tc.profile.ProjectName)
			}

			// Wizard-skipped fields must all be populated.
			if ctx.Description == "" {
				t.Error("Description is empty — wizard default not applied")
			}
			if ctx.ApiPort == 0 {
				t.Error("ApiPort = 0 — port default not applied (loop.sh would render with port 0)")
			}
			if ctx.ModulePath == "" {
				t.Error("ModulePath is empty — module path default not applied")
			}
			if ctx.MaxCycles <= 0 {
				t.Errorf("MaxCycles = %d — cycle config not applied (loop.sh would hang at zero)", ctx.MaxCycles)
			}
			if ctx.DeveloperMaxTurns <= 0 {
				t.Errorf("DeveloperMaxTurns = %d — cycle config not applied", ctx.DeveloperMaxTurns)
			}
			if ctx.TesterMaxTurns <= 0 {
				t.Errorf("TesterMaxTurns = %d — cycle config not applied", ctx.TesterMaxTurns)
			}
			if ctx.ReviewerMaxTurns <= 0 {
				t.Errorf("ReviewerMaxTurns = %d — cycle config not applied", ctx.ReviewerMaxTurns)
			}

			// Go projects need GoVersion for Dockerfile base image.
			if ctx.IsGo && ctx.GoVersion == "" {
				t.Error("GoVersion is empty for Go project — Dockerfile would have no base image tag")
			}

			// Module path must match wizard defaults (org prefix derived from IsInternal).
			expectedOrg := "yourorg"
			if ctx.IsInternal {
				expectedOrg = "internal"
			}
			if !strings.Contains(ctx.ModulePath, expectedOrg) {
				t.Errorf("ModulePath = %q, expected to contain %q", ctx.ModulePath, expectedOrg)
			}
		})
	}
}

// TestApplyContextDefaults_PreservesCallerOverrides ensures that fields the
// caller already populated (e.g. via flags or non-zero defaults from a richer
// future .neo.json schema) are not clobbered by the helper.
func TestApplyContextDefaults_PreservesCallerOverrides(t *testing.T) {
	ctx := &ProjectContext{
		ServiceName:       "custom-svc",
		Tech:              "go",
		IsGo:              true,
		GoVersion:         "1.23",
		Description:       "custom description",
		ApiPort:           9999,
		ModulePath:        "github.com/custom/svc",
		MaxCycles:         42,
		DeveloperMaxTurns: 10,
		TesterMaxTurns:    11,
		ReviewerMaxTurns:  12,
	}

	ApplyContextDefaults(ctx)

	if ctx.GoVersion != "1.23" {
		t.Errorf("GoVersion overwritten: got %q, want %q", ctx.GoVersion, "1.23")
	}
	if ctx.Description != "custom description" {
		t.Errorf("Description overwritten: got %q", ctx.Description)
	}
	if ctx.ApiPort != 9999 {
		t.Errorf("ApiPort overwritten: got %d, want 9999", ctx.ApiPort)
	}
	if ctx.ModulePath != "github.com/custom/svc" {
		t.Errorf("ModulePath overwritten: got %q", ctx.ModulePath)
	}
	if ctx.MaxCycles != 42 {
		t.Errorf("MaxCycles overwritten: got %d", ctx.MaxCycles)
	}
	// The current cycle guard short-circuits when MaxCycles is non-zero, so
	// these fields are preserved by side effect today. Assert each explicitly
	// so a future refactor that tightens the guard to per-field cannot
	// silently regress preservation of the per-agent turn budgets.
	if ctx.DeveloperMaxTurns != 10 {
		t.Errorf("DeveloperMaxTurns: caller override 10 not preserved, got %d", ctx.DeveloperMaxTurns)
	}
	if ctx.TesterMaxTurns != 11 {
		t.Errorf("TesterMaxTurns: caller override 11 not preserved, got %d", ctx.TesterMaxTurns)
	}
	if ctx.ReviewerMaxTurns != 12 {
		t.Errorf("ReviewerMaxTurns: caller override 12 not preserved, got %d", ctx.ReviewerMaxTurns)
	}
}

// TestRunInit_ContextFlow_HasNoTTYBranch is a source-inspection guard that
// init.go's --context branch checks term.IsTerminal(os.Stdin.Fd()) before
// invoking the delta wizard. Without this guard, no-TTY callers (subprocess
// invocation by neo init, CI pipelines) hit the huh "no TTY" error path and
// silently os.Exit(0). A full TUI-driven test is impractical since huh
// requires a TTY — the source check is the practical regression gate.
func TestRunInit_ContextFlow_HasNoTTYBranch(t *testing.T) {
	src, err := os.ReadFile("init.go")
	if err != nil {
		t.Fatalf("reading init.go: %v", err)
	}
	source := string(src)

	// term.IsTerminal must be invoked on os.Stdin.Fd() inside the --context
	// branch. The exact form is `term.IsTerminal(os.Stdin.Fd())`.
	ttyCheck := regexp.MustCompile(`term\.IsTerminal\(os\.Stdin\.Fd\(\)\)`)
	if !ttyCheck.MatchString(source) {
		t.Error("init.go is missing the no-TTY guard `term.IsTerminal(os.Stdin.Fd())` — " +
			"`morpheus init --context` will silently abort when invoked without a TTY")
	}

	// ApplyContextDefaults must be called inside the --context flow so the
	// scaffold is complete when the delta wizard is bypassed.
	if !strings.Contains(source, "ApplyContextDefaults(") {
		t.Error("init.go does not call ApplyContextDefaults — no-TTY branch will leave " +
			"wizard-collected fields (port, module path, cycles) at zero values")
	}
}
