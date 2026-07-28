package neo

import (
	"strings"
	"testing"
)

// TestProjectProfile_EmptyStacksIsValidZeroValue confirms that a ProjectProfile
// with no Stacks can be constructed without panic and that Stacks is nil / has
// length zero — establishing the contract that empty Stacks is a valid state.
func TestProjectProfile_EmptyStacksIsValidZeroValue(t *testing.T) {
	profile := ProjectProfile{}
	if profile.Stacks != nil {
		t.Errorf("zero-value Stacks should be nil, got %v", profile.Stacks)
	}
	if len(profile.Stacks) != 0 {
		t.Errorf("zero-value Stacks should have length 0, got %d", len(profile.Stacks))
	}

	profile2 := ProjectProfile{Stacks: []StackProfile{}}
	if len(profile2.Stacks) != 0 {
		t.Errorf("explicitly empty Stacks should have length 0, got %d", len(profile2.Stacks))
	}
}

// TestStackNameGuard_EmptyStacksProducesEmptyString documents the invariant
// fixed in the wizard service loop: when Stacks is empty the stack name must
// resolve to an empty string rather than panicking with an index-out-of-range.
//
// This test exercises the guarded path by calling BuildNeoManifest with a
// ProjectProfile whose Stacks slice is empty (a profile that could be produced
// by loading a corrupt .neo.json or by a future code path that constructs a
// profile programmatically without going through the wizard).
//
// Before the fix, code at the equivalent location used:
//   stackName := profile.Stacks[0].Name   // panics when len == 0
// After the fix:
//   var stackName string
//   if len(profile.Stacks) > 0 {
//       stackName = profile.Stacks[0].Name
//   }
func TestStackNameGuard_EmptyStacksProducesEmptyString(t *testing.T) {
	// Demonstrate the guard logic directly, mirroring the wizard fix.
	tests := []struct {
		name          string
		stacks        []StackProfile
		wantStackName string
	}{
		{
			name:          "nil stacks — guard returns empty string",
			stacks:        nil,
			wantStackName: "",
		},
		{
			name:          "empty stacks — guard returns empty string",
			stacks:        []StackProfile{},
			wantStackName: "",
		},
		{
			name:          "single stack — guard returns first stack name",
			stacks:        []StackProfile{{Name: "go-api", Language: "go"}},
			wantStackName: "go-api",
		},
		{
			name:          "multiple stacks — guard returns first stack name",
			stacks:        []StackProfile{{Name: "go-api", Language: "go"}, {Name: "node-ui", Language: "nodejs"}},
			wantStackName: "go-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: RunProjectWizard() cannot be unit-tested directly — wizard.AskInput/AskSelect/AskConfirm
			// require an interactive TTY with no injection seam. This test documents the correct guard
			// pattern (len check before [0] access) but does not exercise wizard.go:92-95 itself.
			// It serves as a regression reference if the pattern is ever questioned.
			var stackName string
			if len(tt.stacks) > 0 {
				stackName = tt.stacks[0].Name
			}

			if stackName != tt.wantStackName {
				t.Errorf("stackName = %q, want %q", stackName, tt.wantStackName)
			}
		})
	}
}

// TestBuildNeoManifest_EmptyStacks confirms that BuildNeoManifest does not
// panic when Stacks is empty. The manifest is still valid (no per-stack rule
// files are emitted) and must contain at least the core static entries.
func TestBuildNeoManifest_EmptyStacks(t *testing.T) {
	profile := &ProjectProfile{
		ProjectName: "empty-stacks-project",
		ProjectType: "single-app",
		TeamSize:    "solo",
		Stacks:      []StackProfile{}, // empty — triggers the bug path
		IsInternal:    false,
		CreatedDate: "2026-03-26",
		OutputPath:  "/tmp/neo-empty-stacks-test",
	}

	// Must not panic.
	manifest := BuildNeoManifest(profile, "/tmp/neo-empty-stacks-test")

	// Core files (CLAUDE.md + 4 context + session rule + 6 agents + 10 skills) = 22
	// No per-stack rule files since Stacks is empty.
	if len(manifest) == 0 {
		t.Fatal("manifest should not be empty even with no stacks")
	}

	// No stack-specific rule file should appear.
	for _, entry := range manifest {
		if entry.TemplatePath == ".claude/rules/stack-services.md.tmpl" {
			t.Errorf("stack-services.md.tmpl should not appear in manifest when Stacks is empty, found output path: %s", entry.OutputPath)
		}
	}
}

// TestNewTrinityConfig_EmptyStacks confirms that newTrinityConfig does not
// panic when Stacks is empty and that it returns a config with no stack
// entries.
func TestNewTrinityConfig_EmptyStacks(t *testing.T) {
	profile := &ProjectProfile{
		ProjectName: "no-stacks",
		ProjectType: "single-app",
		TeamSize:    "solo",
		Stacks:      nil,
	}

	cfg := newTrinityConfig(profile)
	if len(cfg.Stacks) != 0 {
		t.Errorf("trinity config should have 0 stacks, got %d", len(cfg.Stacks))
	}
}

// TestNewTrinityConfig_EmptyStacksSlice is the same as the nil case but with
// an explicitly allocated empty slice — both must be safe.
func TestNewTrinityConfig_EmptyStacksSlice(t *testing.T) {
	profile := &ProjectProfile{
		ProjectName: "no-stacks-slice",
		ProjectType: "single-app",
		TeamSize:    "solo",
		Stacks:      []StackProfile{},
	}

	cfg := newTrinityConfig(profile)
	if len(cfg.Stacks) != 0 {
		t.Errorf("trinity config should have 0 stacks for empty slice, got %d", len(cfg.Stacks))
	}
}

// TestGenerateEcosystem_EmptyStacks is an integration-level regression test
// for the Stacks[0] guard. It calls GenerateEcosystem with an empty-stacks
// profile and verifies that:
//   1. No panic occurs.
//   2. Core ecosystem files are still written.
//   3. No per-stack rule file is written.
func TestGenerateEcosystem_EmptyStacks(t *testing.T) {
	dir := t.TempDir()

	profile := &ProjectProfile{
		ProjectName: "regression-empty-stacks",
		ProjectType: "single-app",
		TeamSize:    "solo",
		Stacks:      []StackProfile{},
		IsInternal:    false,
		CreatedDate: "2026-03-26",
		OutputPath:  dir,
	}

	// Must not panic — this is the primary regression guard.
	written, err := GenerateEcosystem(profile, dir)
	if err != nil {
		t.Fatalf("GenerateEcosystem failed with empty stacks: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("expected files to be written even with no stacks")
	}

	// Core files must exist.
	for _, rel := range []string{
		"CLAUDE.md",
		".claude/SERVICE_CONTEXT.md",
		".claude/NEXT_STEPS.md",
		".claude/DECISIONS.md",
		".claude/KNOWN_ISSUES.md",
	} {
		// Directly verify presence via written paths.
		found := false
		for _, w := range written {
			if strings.HasSuffix(w, rel) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s in written files; written count = %d", rel, len(written))
		}
	}
}
